package message

type Handler interface {
	Handle(message *Message)
}

type DefaultHandler struct {
}

func (dh *DefaultHandler) Handle(message *Message) {
	log.Warningf("No handler for type %s, using default...", message.Type.String())
}
