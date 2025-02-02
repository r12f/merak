/*
MIT License
Copyright(c) 2022 Futurewei Cloud
    Permission is hereby granted,
    free of charge, to any person obtaining a copy of this software and associated documentation files(the "Software"), to deal in the Software without restriction,
    including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and / or sell copies of the Software, and to permit persons
    to whom the Software is furnished to do so, subject to the following conditions:
    The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.
    THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
    FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY,
    WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/

package handler

import (
	"context"
	"log"

	"time"

	"github.com/futurewei-cloud/merak/services/merak-topo/database"

	"fmt"

	"strings"

	pb_common "github.com/futurewei-cloud/merak/api/proto/v1/common"
	pb "github.com/futurewei-cloud/merak/api/proto/v1/topology"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

//function CREATE
/* save the part of gw creation and mac learning for future requirment, comment the related code now*/
func Create(k8client *kubernetes.Clientset, topo_id string, aca_num uint32, rack_num uint32, aca_per_rack uint32, cgw_num uint32, data_plane_cidr string, ports_per_vswitch uint32, images []*pb.InternalTopologyImage, aca_parameters string, returnMessage *pb.ReturnTopologyMessage) error {

	start_time := time.Now()
	log.Println("CREATE: ====== Start: Parse gRPC message ======= ")
	log.Printf("Vhost number is: %v\n", aca_num)
	log.Printf("Rack number is: %v\n", rack_num)
	log.Printf("Vhosts per rack is: %v\n", aca_per_rack)

	log.Printf("Ports per vswitch is: %v\n", ports_per_vswitch)

	log.Println("CREATE:====== Start: Generate multi-layer topology structure ======= ")

	err_create, topo := Create_multiple_layers_vswitches(int(aca_num), int(rack_num), int(aca_per_rack), int(ports_per_vswitch), data_plane_cidr)
	if err_create != nil {
		return fmt.Errorf("create multiple layers vswitches error %s", err_create)
	}

	topo.Topology_id = topo_id

	elaps0 := time.Since(start_time)
	start0 := time.Now()

	log.Printf("CREATE:=== Complete: Generate topology data === %v", elaps0)

	err1 := Topo_save(topo)
	if err1 != nil {
		return fmt.Errorf("save topology to redis error %s", err1)
	}

	elaps1 := time.Since(start0)
	start1 := time.Now()
	log.Printf("CREATE:=== Complete: Save topology to redis DB === %v", elaps1)

	k8s_nodes, err1 := k8client.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})

	if err1 != nil {
		return fmt.Errorf("CREATE: failed to list k8s host nodes info %s", err1)
	}

	for _, s := range k8s_nodes.Items {
		var hnode database.HostNode

		node_yaml, err2 := k8client.CoreV1().Nodes().Get(Ctx, s.Name, metav1.GetOptions{})
		if err2 != nil {
			return fmt.Errorf("CREATE: get k8s host node info from k8s cluster error %s", err2)
		}

		for _, c := range s.Status.Conditions {
			if c.Type == corev1.NodeReady {
				hnode.Status = database.STATUS_READY
				break
			}

		}

		for _, res := range node_yaml.Status.Addresses {
			if res.Type == "InternalIP" {
				hnode.Ip = res.Address
				break
			}
		}

		log.Printf("create== hnode %v", hnode)

		// Make return message with k8s cluster nodes info
		err := database.SetValue(topo_id+":"+s.Name, hnode)
		if err != nil {
			return fmt.Errorf("updatecomputenode: save host node info in DB error %s", err.Error())
		}

	}

	for _, s := range k8s_nodes.Items {
		var hnode database.HostNode
		var hrm pb_common.InternalHostInfo

		h, err := database.FindHostEntity(topo_id+":"+s.Name, "")

		hnode = h

		log.Printf("create== hnode= get from DB %v", hnode)

		if err != nil {
			log.Printf("INFO: get host node info from DB error %s", err)
		} else {
			hrm.Ip = hnode.Ip
			if hnode.Status == database.STATUS_READY {
				hrm.Status = pb_common.Status_READY

			} else {
				hrm.Status = pb_common.Status_NONE
			}

			hrm.RoutingRules = hnode.Routing_rule
			if err1 != nil {
				log.Printf("INFO: make host node return message error %s", err1)
			}

			returnMessage.Hosts = append(returnMessage.Hosts, &hrm)
		}

	}

	log.Println("CREATE: === Return deployed compute nodes information ===")

	for _, node := range topo.Vnodes {
		var cnode database.ComputeNode
		if strings.Contains(node.Name, "vhost") {
			cnode.Name = node.Name
			cnode.Id = node.Id
			cnode.DatapathIp = strings.Split(node.Nics[len(node.Nics)-1].Ip, "/")[0]
			cnode.Veth = node.Nics[len(node.Nics)-1].Intf

			cnode.OperationType = database.OPERATION_CREATE
			cnode.Mac = "ff:ff:ff:ff:ff:ff"
			cnode.Status = database.STATUS_DEPLOYING

			log.Printf("create== cnode %v", cnode)

			err := database.SetValue(topo_id+":"+cnode.Name, cnode)
			if err != nil {
				return fmt.Errorf("updatecomputenode: save compute node error %s", err.Error())
			}
		}

	}

	for _, node := range topo.Vnodes {
		if strings.Contains(node.Name, "vhost") {
			var cnode database.ComputeNode
			var crm pb_common.InternalComputeInfo

			c, err := database.FindComputeEntity(topo_id+":"+node.Name, "")

			cnode = c

			log.Printf("create== cnode get from DB %v", cnode)

			if err != nil {
				log.Printf("INFO: check compute node from DB error %s", err.Error())
			}

			crm.Id = cnode.Id
			crm.Name = cnode.Name
			crm.DatapathIp = cnode.DatapathIp
			crm.ContainerIp = cnode.ContainerIp
			crm.Mac = cnode.Mac
			crm.Veth = cnode.Veth
			if cnode.Status == database.STATUS_READY {
				crm.Status = pb_common.Status_READY
			} else if cnode.Status == database.STATUS_DELETING {
				crm.Status = pb_common.Status_DELETING
			} else if cnode.Status == database.STATUS_DEPLOYING {
				crm.Status = pb_common.Status_DEPLOYING
			} else {
				crm.Status = pb_common.Status_NONE
			}
			if cnode.OperationType == database.OPERATION_CREATE {
				crm.OperationType = pb_common.OperationType_CREATE
			} else if cnode.OperationType == database.OPERATION_INFO {
				crm.OperationType = pb_common.OperationType_INFO
			} else if cnode.OperationType == database.OPERATION_DELETE {
				crm.OperationType = pb_common.OperationType_DELETE
			} else if cnode.OperationType == database.OPERATION_UPDATE {
				crm.OperationType = pb_common.OperationType_UPDATE
			}

			returnMessage.ComputeNodes = append(returnMessage.ComputeNodes, &crm)

		}

	}
	elaps2 := time.Since(start1)

	log.Printf("CREATE: === Complete: Return compute nodes and k8s cluster information to Scenario Manager=== %v", elaps2)

	log.Println("CREATE:=== Start: Deploy topology structure in k8s cluster === ")

	log.Println("=== Topology Deployment === ")
	var aca_image string
	var ovs_image string

	for _, img := range images {
		if strings.Contains(img.Name, "ACA") {
			aca_image = img.Registry
		} else if strings.Contains(img.Name, "OVS") {
			ovs_image = img.Registry
		}
	}

	go Topo_deploy(k8client, aca_image, ovs_image, topo, aca_parameters)

	return nil

}

func UpdateComputenodeInfo(k8client *kubernetes.Clientset, topo_id string) error {
	start_time := time.Now()
	topo, err := database.FindTopoEntity(topo_id, "")

	if err != nil {
		return fmt.Errorf("update compute node info: query topology_id error %s", err)
	}

	k8s_nodes, err1 := k8client.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err1 != nil {
		return fmt.Errorf("UpdateComputenodeInfo: failed to list k8s host nodes info %s", err1)
	}

	for _, s := range k8s_nodes.Items {
		var hnode database.HostNode

		node_yaml, err2 := k8client.CoreV1().Nodes().Get(Ctx, s.Name, metav1.GetOptions{})
		if err2 != nil {
			return fmt.Errorf("CREATE: failed to get k8s host node info %s", err2)
		}

		for _, c := range s.Status.Conditions {
			if c.Type == corev1.NodeReady {
				hnode.Status = database.STATUS_READY
				break
			}

		}

		for _, res := range node_yaml.Status.Addresses {
			if res.Type == "InternalIP" {
				hnode.Ip = res.Address
				break
			}
		}

		// Make return message with k8s cluster nodes info
		err := database.SetValue(topo_id+":"+s.Name, hnode)
		if err != nil {
			return fmt.Errorf("updatecomputenode: save host node error %s", err.Error())
		}

	}

	elaps0 := time.Since(start_time)
	start_time0 := time.Now()

	log.Printf("updatecomputenode:=========compelete updating host node info in DB =========== %v", elaps0)

	/*comment mac learning function*/

	// err3 := QueryMac(client, topo_id)

	// if err3 != nil {
	// 	return fmt.Errorf("query mac error %s", err3)
	// }

	// log.Printf("updatecomputenode:=========Update mac addresses ===========")

	for _, node := range topo.Vnodes {

		if strings.Contains(node.Name, "vhost") {

			var cnode database.ComputeNode

			res, err := k8client.CoreV1().Pods("default").Get(Ctx, node.Name, metav1.GetOptions{})

			if err != nil {
				log.Printf("updatecomputenode: get pod info from k8s error %s", err)
			} else {
				cnode.Name = res.Name
				cnode.Id = string(res.UID)

				cnode.DatapathIp = strings.Split(node.Nics[len(node.Nics)-1].Ip, "/")[0]
				cnode.Veth = node.Nics[len(node.Nics)-1].Intf
				if res.Status.PodIP != "" {
					cnode.ContainerIp = res.Status.PodIP
				} else {
					log.Printf("pod ip is not ready %v", res.Name)
				}

				/*comment mac learning function*/
				// mac, err := database.Get(topo_id + ":" + cnode.Ip)
				// if err != nil {
				// 	log.Printf("updatecomputenode: mac address is not available")
				// } else {
				// 	cnode.Mac = strings.Trim(mac, "\"")
				// 	cnode.OperationType = pb.OperationType_INFO
				// }

				cnode.Mac = "ff:ff:ff:ff:ff:ff"
				cnode.OperationType = database.OPERATION_INFO

				if len(res.Status.ContainerStatuses) == 0 {
					cnode.Status = database.STATUS_NONE
					log.Printf("updatecomputenode: container status is not available %v", res.Name)
				} else {
					if res.Status.ContainerStatuses[len(res.Status.ContainerStatuses)-1].Ready {
						cnode.Status = database.STATUS_READY
					}

				}

				err_db := database.SetValue(topo_id+":"+node.Name, cnode)
				if err_db != nil {
					return fmt.Errorf("updatecomputenode: save compute node error %s", err_db.Error())
				}

			}

		}

	}

	elaps := time.Since(start_time0)

	log.Printf("updatecomputenode:=========complete updating compute nodes info in DB =========== %v", elaps)

	return nil
}

func Info(k8client *kubernetes.Clientset, topo_id string, returnMessage *pb.ReturnTopologyMessage) error {

	topo, err := database.FindTopoEntity(topo_id, "")

	if err != nil {
		return fmt.Errorf("update compute node info: query topology_id error %s", err)
	}

	k8s_nodes, err1 := k8client.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})

	if err1 != nil {
		return fmt.Errorf("INFO: list k8s host nodes info error %s", err1)
	}

	for _, s := range k8s_nodes.Items {
		var hnode database.HostNode
		var hrm pb_common.InternalHostInfo

		h, err := database.FindHostEntity(topo_id+":"+s.Name, "")
		hnode = h
		if err != nil {
			log.Printf("INFO: get host node info from DB error %s", err)
		} else {
			hrm.Ip = hnode.Ip
			if hnode.Status == database.STATUS_READY {
				hrm.Status = pb_common.Status_READY

			} else {
				hrm.Status = pb_common.Status_NONE
			}

			hrm.RoutingRules = hnode.Routing_rule
			if err1 != nil {
				log.Printf("INFO: make host node return message error %s", err1)
			}

			returnMessage.Hosts = append(returnMessage.Hosts, &hrm)
		}

	}

	for _, node := range topo.Vnodes {
		if strings.Contains(node.Name, "vhost") {
			var cnode database.ComputeNode
			var crm pb_common.InternalComputeInfo

			c, err := database.FindComputeEntity(topo_id+":"+node.Name, "")
			cnode = c
			if err != nil {
				log.Printf("INFO: check compute node from DB error %s", err.Error())
			}

			crm.Id = cnode.Id
			crm.Name = cnode.Name
			crm.DatapathIp = cnode.DatapathIp
			crm.ContainerIp = cnode.ContainerIp
			crm.Mac = cnode.Mac
			crm.Veth = cnode.Veth
			if cnode.Status == database.STATUS_READY {
				crm.Status = pb_common.Status_READY
			} else if cnode.Status == database.STATUS_DELETING {
				crm.Status = pb_common.Status_DELETING
			} else if cnode.Status == database.STATUS_DEPLOYING {
				crm.Status = pb_common.Status_DEPLOYING
			} else {
				crm.Status = pb_common.Status_NONE
			}
			if cnode.OperationType == database.OPERATION_CREATE {
				crm.OperationType = pb_common.OperationType_CREATE
			} else if cnode.OperationType == database.OPERATION_INFO {
				crm.OperationType = pb_common.OperationType_INFO
			} else if cnode.OperationType == database.OPERATION_DELETE {
				crm.OperationType = pb_common.OperationType_DELETE
			} else if cnode.OperationType == database.OPERATION_UPDATE {
				crm.OperationType = pb_common.OperationType_UPDATE
			}

			returnMessage.ComputeNodes = append(returnMessage.ComputeNodes, &crm)

		}

	}

	go UpdateComputenodeInfo(k8client, topo_id)

	return nil
}

func Delete(k8client *kubernetes.Clientset, topo_id string, returnMessage *pb.ReturnTopologyMessage) error {
	topo, err_db := database.FindTopoEntity(topo_id, "")

	if err_db != nil {
		return fmt.Errorf("query topology_id error %s", err_db)
	}

	k8s_nodes, err1 := k8client.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})

	if err1 != nil {
		return fmt.Errorf("INFO: list k8s host nodes info error %s", err1)
	}

	for _, s := range k8s_nodes.Items {
		var hnode database.HostNode
		var hrm pb_common.InternalHostInfo

		h, err := database.FindHostEntity(topo_id+":"+s.Name, "")

		hnode = h

		if err != nil {
			log.Printf("INFO: get host node info from DB error %s", err)
		} else {
			hrm.Ip = hnode.Ip
			if hnode.Status == database.STATUS_READY {
				hrm.Status = pb_common.Status_READY

			} else {
				hrm.Status = pb_common.Status_NONE
			}

			hrm.RoutingRules = hnode.Routing_rule
			if err1 != nil {
				log.Printf("INFO: make host node return message error %s", err1)
			}

			returnMessage.Hosts = append(returnMessage.Hosts, &hrm)
		}

	}

	for _, node := range topo.Vnodes {
		if strings.Contains(node.Name, "vhost") {
			var cnode database.ComputeNode
			var crm pb_common.InternalComputeInfo

			c, err := database.FindComputeEntity(topo_id+":"+node.Name, "")

			cnode = c

			if err != nil {
				log.Printf("INFO: check compute node from DB error %s", err.Error())
			}

			crm.Id = cnode.Id
			crm.Name = cnode.Name
			crm.DatapathIp = cnode.DatapathIp
			crm.ContainerIp = cnode.ContainerIp
			crm.Mac = cnode.Mac
			crm.Veth = cnode.Veth
			crm.Status = pb_common.Status_DELETING
			crm.OperationType = pb_common.OperationType_DELETE

			returnMessage.ComputeNodes = append(returnMessage.ComputeNodes, &crm)

		}

	}

	go Topo_delete(k8client, topo)

	return nil
}

/*comment this function for query mac address*/
// func QueryMac(k8client *kubernetes.Clientset, topo_id string) error {

// 	topo_data, err := database.FindTopoEntity(topo_id, "")

// 	if err != nil {
// 		return fmt.Errorf("failed to retrieve topology data from DB %s", err)
// 	}
// 	count_ping := 0

// 	for _, node := range topo_data.Vnodes {

// 		if strings.Contains(node.Name, "cgw") && count_ping < 1 {
// 			count_ping = count_ping + 1

// 			pod, err := database.FindPodEntity(topo_id+":"+node.Name, "")

// 			if err != nil {
// 				return fmt.Errorf("failed to retrieve pod data from DB %s", err)
// 			}

// 			ip_list, err_db := database.FindIPEntity(topo_id+":ips", "")
// 			if err_db != nil {
// 				return fmt.Errorf("failed to retrieve ip address data from DB %s", err_db)
// 			}

// 			for index := len(ip_list) - 1; index >= 0; index-- {
// 				ip := ip_list[index]
// 				ip = strings.Split(ip, "/")[0]
// 				cmd1 := []string{
// 					"ping",
// 					"-c 1",
// 					ip,
// 				}
// 				Pod_query(k8client, pod, cmd1)

// 			}

// 			cmd2 := []string{
// 				"arp",
// 				"-a",
// 			}
// 			out2, err3 := Pod_query(k8client, pod, cmd2)

// 			if err3 != nil {
// 				return fmt.Errorf("failed to query pod compute node info from K8s %s", err)
// 			}

// 			re, _ := regexp.Compile(`^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$`)

// 			re2, _ := regexp.Compile(`^[a-fA-F0-9]{2}(:[a-fA-F0-9]{2}){5}$`)
// 			var ip string
// 			var mac string
// 			count := 0

// 			for _, s := range strings.Split(out2, " ") {
// 				s1 := strings.Trim(s, "(")
// 				s2 := strings.Trim(s1, ")")
// 				s2 = strings.Trim(s2, "\"")

// 				if re.MatchString(s2) {
// 					ip = s2
// 					count = count + 1
// 				} else if re2.MatchString(s) {
// 					mac = s2
// 					count = count + 1
// 					if count == 2 {
// 						database.SetValue(topo_id+":"+ip, mac)
// 						count = 0
// 					}
// 				}
// 			}
// 			if count != 0 {
// 				log.Printf("arp output parsing error. ip and mac doesn't match")
// 			}
// 		}
// 	}
// 	return nil
// }
