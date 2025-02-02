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

package main

import (
	"context"
	"flag"
	"log"
	"time"

	common_pb "github.com/futurewei-cloud/merak/api/proto/v1/common"
	pb "github.com/futurewei-cloud/merak/api/proto/v1/network"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	defaultName = "world"
)

var (
	addr = flag.String("addr", "localhost:40053", "the address to connect to")
	name = flag.String("name", defaultName, "Name to greet")
)

func main() {
	testInternalSecurityGroupRulelnfo := pb.InternalSecurityGroupRulelnfo{
		OperationType: common_pb.OperationType_CREATE,
		//Id:             "1",
		Name:           "InternalSecurityGroupRulelnfo",
		Description:    "InternalSecurityGroupRulelnfo_description",
		Ethertype:      "5",
		Direction:      "6",
		Protocol:       "7",
		PortRange:      "8",
		RemoteGroupId:  "9",
		RemoteIpPrefix: "10",
	}
	testInternalRouterInfo := pb.InternalRouterInfo{
		OperationType: common_pb.OperationType_CREATE,
		//Id:            "1",
		Name:    "InternalRouterInfo",
		Subnets: []string{"10.8.1.0/24", "10.8.2.0/24"},
	}
	testInternalGatewayInfo := pb.InternalGatewayInfo{
		OperationType: common_pb.OperationType_CREATE,
		//Id:            "1",
		Name: "InternalGatewayInfo",
		Ips:  []string{"10.8.1.1", "10.8.2.1"},
	}
	testInternalSecurityGroupInfo := pb.InternalSecurityGroupInfo{
		OperationType: common_pb.OperationType_CREATE,
		//Id:            "1",
		Name:      "InternalSecurityGroupInfo",
		TenantId:  "123456789",
		ProjectId: "123456789",
		Rules:     []*pb.InternalSecurityGroupRulelnfo{&testInternalSecurityGroupRulelnfo},
		ApplyTo:   []string{"ApplyTo"},
	}
	testInternalSubnetInfo := common_pb.InternalSubnetInfo{
		//SubnetId:   "SubnetId1",
		SubnetCidr: "10.8.1.0/24",
		SubnetGw:   "10.8.1.1",
		NumberVms:  0,
	}
	testInternalSubnetInfo2 := common_pb.InternalSubnetInfo{
		//SubnetId:   "SubnetId2",
		SubnetCidr: "10.8.2.0/24",
		SubnetGw:   "10.8.2.1",
		NumberVms:  0,
	}
	testInternalVpcInfo := common_pb.InternalVpcInfo{
		//VpcId:     "VpcId1",
		TenantId:  "123456789",
		ProjectId: "123456789",
		VpcCidr:   "10.8.0.0/16",
		Subnets:   []*common_pb.InternalSubnetInfo{&testInternalSubnetInfo, &testInternalSubnetInfo2},
	}
	testInternalNetworkInfo := pb.InternalNetworkInfo{
		OperationType: common_pb.OperationType_CREATE,
		//Id:                     "1",
		Name:                   "InternalNetworkInfo",
		NumberOfVpcs:           1,
		NumberOfSubnetPerVpc:   2,
		Vpcs:                   []*common_pb.InternalVpcInfo{&testInternalVpcInfo},
		NumberOfSecurityGroups: 1,
		Routers:                []*pb.InternalRouterInfo{&testInternalRouterInfo},
		Gateways:               []*pb.InternalGatewayInfo{&testInternalGatewayInfo},
		SecurityGroups:         []*pb.InternalSecurityGroupInfo{&testInternalSecurityGroupInfo},
	}
	testInternalServiceInfo1 := common_pb.InternalServiceInfo{
		OperationType: common_pb.OperationType_CREATE,
		Id:            "1",
		Name:          "Service 1",
		Cmd:           "curl",
		Url:           "http://10.213.43.224:30001/segments/createDefaultTable",
		Parameters:    []string{"-X POST", "-H 'Content-Type: application/json'", "-H 'Accept: */*'"},
		ReturnCode:    nil,
		ReturnString:  []string{"ReturnString"},
		WhenToRun:     "INIT",
		WhereToRun:    "NETWORK",
	}
	testInternalServiceInfo3 := common_pb.InternalServiceInfo{
		OperationType: common_pb.OperationType_CREATE,
		Id:            "3",
		Name:          "Service 3",
		Cmd:           "InternalServiceInfo CMD",
		Url:           "InternalServiceInfo URL",
		Parameters:    []string{"Parameters"},
		ReturnCode:    nil,
		ReturnString:  []string{"ReturnString"},
		WhenToRun:     "AFTER:Service 2",
		WhereToRun:    "network",
	}
	testInternalServiceInfo2 := common_pb.InternalServiceInfo{
		OperationType: common_pb.OperationType_CREATE,
		Id:            "2",
		Name:          "Service 2",
		Cmd:           "InternalServiceInfo CMD",
		Url:           "InternalServiceInfo URL",
		Parameters:    []string{"Parameters"},
		ReturnCode:    nil,
		ReturnString:  []string{"ReturnString"},
		WhenToRun:     "AFTER:Service 1",
		WhereToRun:    "network",
	}
	testInternalComputeInfo1 := common_pb.InternalComputeInfo{
		OperationType: common_pb.OperationType_CREATE,
		Id:            "YM_node5_id",
		Name:          "YM_node5",
		DatapathIp:    "192.168.10.15",
		Mac:           "36:db:23:8c:4a:c5",
		Veth:          "eth1",
	}
	testInternalComputeInfo2 := common_pb.InternalComputeInfo{
		OperationType: common_pb.OperationType_CREATE,
		Id:            "YM_node6_id",
		Name:          "YM_node6",
		DatapathIp:    "192.168.10.16",
		Mac:           "36:db:23:8c:4a:c6",
		Veth:          "eth1",
	}
	testInternalStorageInfo := pb.InternalStorageInfo{Info: "InternalStorageInfo"}
	testInternalNetConfigExtraInfo := pb.InternalNetConfigExtraInfo{Info: "InternalNetConfigExtraInfo"}
	testInternalNetConfigConfiguration := pb.InternalNetConfigConfiguration{
		FormatVersion:  0,
		RevisionNumber: 0,
		RequestId:      "InternalNetConfigConfigurationRequestId",
		NetconfigId:    "InternalNetConfigConfigurationNetconfigId",
		MessageType:    0,
		Services:       []*common_pb.InternalServiceInfo{&testInternalServiceInfo1, &testInternalServiceInfo3, &testInternalServiceInfo2},
		Computes:       []*common_pb.InternalComputeInfo{&testInternalComputeInfo1, &testInternalComputeInfo2},
		Network:        &testInternalNetworkInfo,
		Storage:        &testInternalStorageInfo,
		ExtraInfo:      &testInternalNetConfigExtraInfo,
	}
	testInternalNetConfigInfo := pb.InternalNetConfigInfo{
		OperationType: common_pb.OperationType_CREATE,
		Config:        &testInternalNetConfigConfiguration,
	}

	flag.Parse()
	// Set up a connection to the server.
	conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("did not connect: %v", err)
	}
	defer conn.Close()
	// c := pb.NewGreeterClient(conn)
	c := pb.NewMerakNetworkServiceClient(conn)

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// r, err := c.SayHello(ctx, &pb.HelloRequest{Name: *name})
	r, err := c.NetConfigHandler(ctx, &testInternalNetConfigInfo)
	if err != nil {
		log.Printf("could not greet: %v", err)
	}
	//log.Printf("Greeting: %s", r.GetMessage())
	log.Printf("Return: %s", r)
}
