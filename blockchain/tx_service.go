package blockchain

import (
	"context"
	net "github.com/gagarinchain/network"
	"github.com/gagarinchain/network/common"
	"github.com/gagarinchain/network/common/api"
	common2 "github.com/gagarinchain/network/common/eth/common"
	"github.com/gagarinchain/network/common/message"
	"github.com/gagarinchain/network/common/protobuff"
	"github.com/gagarinchain/network/common/tx"
	"github.com/gagarinchain/network/network"
	"github.com/golang/protobuf/ptypes"
)

type TxService struct {
	validator net.Validator
	txPool    TransactionPool
	netserv   network.Service
	bc        api.Blockchain
	me        *common.Peer
}

func NewService(validator net.Validator, txPool TransactionPool, netserv network.Service, bc api.Blockchain, me *common.Peer) *TxService {
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

			if t.TxType() == tx.Settlement {
				if err := s.SendAgreement(ctx, t); err != nil {
					log.Error("Can't send agreement", err)
					continue
				}
				//we change to, since we don't need default address any more
				t.SetTo(common2.BytesToAddress(t.Hash().Bytes()[12:]))
			}

			s.txPool.Add(t)

		case <-ctx.Done():
			log.Info("Stopping transaction service")
			return
		}
	}
}

func (s *TxService) SendAgreement(ctx context.Context, t *tx.Transaction) error {
	rec := s.bc.GetHeadRecord()
	acc, found := rec.Get(s.me.GetAddress())
	var nonce uint64
	if found {
		nonce = acc.Nonce()
	}

	agreement := tx.CreateAgreement(t, nonce, nil)
	if err := agreement.CreateProof(s.me.GetPrivateKey()); err != nil {
		return err
	}

	agreement.Sign(s.me.GetPrivateKey())
	proto := agreement.GetMessage()
	payload, e := ptypes.MarshalAny(proto)
	if e != nil {
		return e
	}
	go s.netserv.BroadcastTransaction(ctx, message.CreateMessage(pb.Message_TRANSACTION, payload, s.me))

	return nil
}
