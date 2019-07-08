package blockchain

import (
	"context"
	"github.com/golang/protobuf/ptypes"
	"github.com/poslibp2p"
	"github.com/poslibp2p/common/message"
	"github.com/poslibp2p/common/protobuff"
	"github.com/poslibp2p/common/tx"
)

type TxService struct {
	validator poslibp2p.Validator
	txPool    TransactionPool
}

func NewService(validator poslibp2p.Validator, txPool TransactionPool) *TxService {
	return &TxService{validator: validator, txPool: txPool}
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
			t, e := tx.CreateTransactionFromMessage(pbt)
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
