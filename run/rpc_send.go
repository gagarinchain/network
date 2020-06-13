package run

import (
	"context"
	"github.com/gagarinchain/common"
	pb "github.com/gagarinchain/common/protobuff"
	"github.com/gagarinchain/common/rpc"
)

func RpcSend(s *common.Settings) {
	client := rpc.InitOnNextViewClient(s.Rpc.Address)

	if _, err := client.Pbc().OnNextView(context.Background(), &pb.OnNextViewRequest{}); err != nil {
		log.Error(err)
		return
	}
}
