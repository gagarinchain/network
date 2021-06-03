package run

import (
	"context"
	"fmt"
	"github.com/gagarinchain/common"
	pb "github.com/gagarinchain/common/protobuff"
	"github.com/gagarinchain/common/rpc"
)

func RpcSend(s *common.Settings) {
	address := fmt.Sprintf("%v:%v", s.Rpc.Host, s.Rpc.Port)
	client := rpc.InitCommonClient(address)

	if view, err := client.Pbc().GetCurrentView(context.Background(), &pb.GetCurrentViewRequest{}); err != nil {
		log.Error(err)
		return
	} else {
		log.Info(view)
	}

}
