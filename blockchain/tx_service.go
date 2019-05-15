package blockchain

import (
	"context"
	"github.com/golang/protobuf/ptypes"
	"github.com/poslibp2p"
	"github.com/poslibp2p/common/message"
	"github.com/poslibp2p/common/protobuff"
	"github.com/poslibp2p/common/tx"
)

type Service struct {
	validator poslibp2p.Validator
	txPool    *TransactionPool
}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Run(ctx context.Context, tchan chan *message.Message) {
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
			}
			//1. Nonce of transaction should be higher then known by one
			//2. Amount + fee should be less or equal to current amount

		case <-ctx.Done():
			log.Info("Stopping transaction service")
			return
		}
	}
}
