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

package tests

import (
	"context"
	"strconv"
	"strings"
	"testing"

	pb "github.com/futurewei-cloud/merak/api/proto/v1/topology"
	constants "github.com/futurewei-cloud/merak/services/common"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

//grpc test
func TestGrpcClient(t *testing.T) {
	var topology_address strings.Builder
	topology_address.WriteString(constants.TOPLOGY_GRPC_SERVER_ADDRESS)
	topology_address.WriteString(":")
	topology_address.WriteString(strconv.Itoa(constants.TOPLOGY_GRPC_SERVER_PORT))

	conn, err := grpc.Dial(topology_address.String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to dial gRPC server address!: %v", err)
	}
	defer conn.Close()

	client := pb.NewMerakTopologyServiceClient(conn)
	resp, err := client.TopologyHandler(context.Background(), &pb.InternalTopologyInfo{})
	if err != nil {
		t.Fatalf("gRPCTestHandler failed: %v", err)
	}
	t.Logf("Response: %+v", resp)
	defer conn.Close()
}
