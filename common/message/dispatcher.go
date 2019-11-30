package message

import (
	"github.com/gagarinchain/network/common/protobuff"
	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("cmd")

type Dispatcher interface {
	Dispatch(msg *Message)
}

type HotstuffDispatcher struct {
	hotstuffChan      chan *Message
	epochChan         chan *Message
	blockProtocolChan chan *Message
}
type TxDispatcher struct {
	txChan chan *Message
}

func NewHotstuffDispatcher(hotstuffChan chan *Message, epochChan chan *Message, blockProtocolChan chan *Message) Dispatcher {
	return &HotstuffDispatcher{hotstuffChan: hotstuffChan, epochChan: epochChan, blockProtocolChan: blockProtocolChan}
}
func NewTxDispatcher(txChan chan *Message) Dispatcher {
	return &TxDispatcher{txChan: txChan}
}

//Dispatch makes simple message validations and choose channel to send message
func (d *HotstuffDispatcher) Dispatch(msg *Message) {
	go func() {
		switch msg.Type {
		case pb.Message_EPOCH_START:
			d.epochChan <- msg
		case pb.Message_VOTE:
			fallthrough
		case pb.Message_PROPOSAL:
			d.hotstuffChan <- msg
		case pb.Message_HELLO_REQUEST:
			fallthrough
		case pb.Message_BLOCK_REQUEST:
			d.blockProtocolChan <- msg
		case pb.Message_BLOCK_HEADER_BATCH_REQUEST:
			d.blockProtocolChan <- msg
		case pb.Message_HELLO_RESPONSE:
			fallthrough
		case pb.Message_BLOCK_RESPONSE:
			log.Warningf("Received message %v, without request, ignoring", msg.Type.String())
		}
	}()
}

func (d *TxDispatcher) Dispatch(msg *Message) {
	go func() {
		if msg.Type == pb.Message_TRANSACTION {
			log.Debugf("received transaction from %v", msg.source.GetPeerInfo().ID.Pretty())
			d.txChan <- msg
		}
	}()
}
