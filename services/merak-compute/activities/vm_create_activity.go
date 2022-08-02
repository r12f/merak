package activities

import (
	"context"
	"strconv"
	"strings"

	pb "github.com/futurewei-cloud/merak/api/proto/v1/merak"
	constants "github.com/futurewei-cloud/merak/services/common"
	"github.com/futurewei-cloud/merak/services/merak-compute/common"
	"go.temporal.io/sdk/activity"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func VmCreate(ctx context.Context) (*pb.ReturnComputeMessage, error) {
	logger := activity.GetLogger(ctx)
	//logger = log.With(logger)
	ids := common.RedisClient.SMembers(ctx, constants.COMPUTE_REDIS_NODE_IP_SET)
	if ids.Err() != nil {
		logger.Error("Unable get node IDs from redis", ids.Err())

		return &pb.ReturnComputeMessage{
			ReturnCode:    pb.ReturnCode_FAILED,
			ReturnMessage: "Unable get node IDs from redis",
		}, ids.Err()
	}
	vms := []*pb.InternalVMInfo{}
	var agent_address strings.Builder
	logger.Info("Success in getting Node IDs! " + ids.String())
	for _, podID := range ids.Val() {
		vmIDsList := common.RedisClient.LRange(ctx, podID, 0, -1)
		if vmIDsList.Err() != nil {
			logger.Error("Unable get node vmIDsList from redis", vmIDsList.Err())
			return &pb.ReturnComputeMessage{
				ReturnCode:    pb.ReturnCode_FAILED,
				ReturnMessage: "Unable get node vmIDsList from redis",
			}, vmIDsList.Err()
		}
		agent_address.Reset()
		agent_address.WriteString(common.RedisClient.HGet(ctx, podID, "ip").Val())
		agent_address.WriteString(":")
		agent_address.WriteString(strconv.Itoa(constants.AGENT_GRPC_SERVER_PORT))
		conn, err := grpc.Dial(agent_address.String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			logger.Info("Failed to dial gRPC server address: "+agent_address.String(), err)
		}
		client := pb.NewMerakAgentServiceClient(conn)

		logger.Info("Pod IDs " + ids.String() + "\n")
		logger.Info("VM Ids " + vmIDsList.String() + "\n")

		for _, vmID := range vmIDsList.Val() {

			logger.Info("Looking up ID " + vmID)
			vm := common.RedisClient.HGetAll(ctx, vmID)
			if vm.Err() != nil {
				logger.Error("Unable get node VM from redis for vmID "+vmID, vm.Err())
				return &pb.ReturnComputeMessage{
					ReturnCode:    pb.ReturnCode_FAILED,
					ReturnMessage: "Unable get node VM from redis for vmID " + vmID,
				}, vm.Err()
			}
			logger.Info("Sending to agent " + vm.String())
			go func(vmID string) {
				port := pb.InternalPortConfig{
					OperationType: pb.OperationType_CREATE,
					Name:          common.RedisClient.HGet(ctx, vmID, "name").Val(),
					Vpcid:         common.RedisClient.HGet(ctx, vmID, "vpc").Val(),
					Tenantid:      common.RedisClient.HGet(ctx, vmID, "tenantID").Val(),
					Projectid:     common.RedisClient.HGet(ctx, vmID, "projectID").Val(),
					Subnetid:      common.RedisClient.HGet(ctx, vmID, "subnetID").Val(),
					Gw:            common.RedisClient.HGet(ctx, vmID, "gw").Val(),
					Sg:            common.RedisClient.HGet(ctx, vmID, "sg").Val(),
					Cidr:          common.RedisClient.HGet(ctx, vmID, "cidr").Val(),
					Hostname:      common.RedisClient.HGet(ctx, vmID, "hostname").Val(),
				}
				resp, err := client.PortHandler(ctx, &port)
				if err != nil {
					logger.Error("Unable create vm ID " + common.RedisClient.HGet(ctx, vmID, "hostIP").Val() + "Reason: " + resp.GetReturnMessage() + "\n")
				}
				logger.Info("Appending VM ", vm)
				return_vm := pb.InternalVMInfo{
					Name:            common.RedisClient.HGet(ctx, vmID, "name").Val(),
					VpcId:           common.RedisClient.HGet(ctx, vmID, "vpc").Val(),
					Ip:              resp.Port.GetIp(),
					SecurityGroupId: common.RedisClient.HGet(ctx, vmID, "sh").Val(),
					SubnetId:        common.RedisClient.HGet(ctx, vmID, "subnetID").Val(),
					DefaultGateway:  common.RedisClient.HGet(ctx, vmID, "gw").Val(),
					Status:          resp.Port.GetStatus(),
				}
				common.RedisClient.HSet(ctx,
					vmID,
					"ip",
					resp.Port.Ip,
					"status",
					resp.Port.Status,
					"deviceID",
					resp.Port.Deviceid,
					"remoteID",
					resp.Port.Remoteid,
				)
				vms = append(vms, &return_vm)
				logger.Info("Response from agent at address: " + resp.GetReturnMessage())
			}(vmID)
		}
		defer conn.Close()
	}

	return &pb.ReturnComputeMessage{
		ReturnCode:    pb.ReturnCode_OK,
		ReturnMessage: "Success!",
		Vms:           vms,
	}, nil
}
