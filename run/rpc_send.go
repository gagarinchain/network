package run

import (
	"context"
	"github.com/gagarinchain/common"
	pb "github.com/gagarinchain/common/protobuff"
	"github.com/gagarinchain/common/rpc"
)

func RpcSend(s *common.Settings) {
	client := rpc.InitCommonClient(s.Rpc.Address)

	if view, err := client.Pbc().GetCurrentView(context.Background(), &pb.GetCurrentViewRequest{}); err != nil {
		log.Error(err)
		return
	} else {
		log.Info(view)
	}

}
