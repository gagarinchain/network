package tx

import (
	"context"
	"github.com/gagarinchain/common"
	"github.com/gagarinchain/common/api"
	"github.com/gagarinchain/common/message"
	"github.com/gagarinchain/common/network"
	"github.com/gagarinchain/common/protobuff"
	"github.com/gagarinchain/common/tx"
	"github.com/golang/protobuf/ptypes"
)

type TxService struct {
	validator api.Validator
	txPool    TransactionPool
	netserv   network.Service
	bc        api.Blockchain
	me        *common.Peer
}

func NewService(validator api.Validator, txPool TransactionPool, netserv network.Service, bc api.Blockchain, me *common.Peer) *TxService {
	return &TxService{validator: validator, txPool: txPool, netserv: netserv, bc: bc, me: me}
}

func (s *TxService) Run(ctx context.Context, tchan chan *message.Message) {
	for {
		select {
		case m := <-tchan:
			pbt := &pb.Transaction{}
			if err := ptypes.UnmarshalAny(m.GetPayload(), pbt); err != nil {
				log.Error("Can't parse transaction message", err)
				continue
			}
			t, e := tx.CreateTransactionFromMessage(pbt, false)
			if e != nil {
				log.Error("Can't create transaction", e)
				continue
			}
			b, e := s.validator.IsValid(t)
			if !b {
				log.Error("Transaction is not valid", e)
				continue
			}

			s.txPool.Add(t)

		case <-ctx.Done():
			log.Info("Stopping transaction service")
			return
		}
	}
}
