package message

import (
	"github.com/op/go-logging"
	"github.com/poslibp2p/message/protobuff"
)

var log = logging.MustGetLogger("cmd")
var DEFAULT_HANDLER Handler = &DefaultHandler{}

type Dispatcher struct {
	Handlers map[pb.Message_MessageType]Handler
	MsgChan  chan *Message
}

//Asynchronously dispatching message to appropriate handler
func (d *Dispatcher) Dispatch(msg *Message) {
	d.MsgChan <- msg
}

func (d *Dispatcher) StartUp() {
	for {
		msg := <-d.MsgChan

		handler, ok := d.Handlers[msg.Type]

		if !ok {
			handler = DEFAULT_HANDLER
		}

		handler.Handle(msg)
	}
}
