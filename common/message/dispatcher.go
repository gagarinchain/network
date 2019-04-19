package message

import (
	"github.com/op/go-logging"
	"github.com/poslibp2p"
	"github.com/poslibp2p/common/protobuff"
)

var log = logging.MustGetLogger("cmd")

type Dispatcher struct {
	validators        []poslibp2p.Validator
	hotstuffChan      chan *Message
	blockProtocolChan chan *Message
}

func NewDispatcher(validators []poslibp2p.Validator, hotstuffChan chan *Message, blockProtocolChan chan *Message) *Dispatcher {
	return &Dispatcher{validators: validators, hotstuffChan: hotstuffChan, blockProtocolChan: blockProtocolChan}
}

//Dispatch makes simple message validations and choose channel to send message
func (d *Dispatcher) Dispatch(msg *Message) {
	go func() {
		switch msg.Type {
		case pb.Message_VOTE:
			fallthrough
		case pb.Message_EPOCH_START:
			fallthrough
		case pb.Message_PROPOSAL:
			d.hotstuffChan <- msg
		case pb.Message_HELLO_REQUEST:
			fallthrough
		case pb.Message_BLOCK_REQUEST:
			d.blockProtocolChan <- msg
		case pb.Message_HELLO_RESPONSE:
			fallthrough
		case pb.Message_BLOCK_RESPONSE:
			log.Warningf("Received message %d, without request, ignoring", msg.Type.String())
		}
	}()
}
