package message

import (
	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/poslibp2p/message/protobuff"
)

type Message struct {
	sm []byte
	*pb.Message
}

func CreateMessage(messageType pb.Message_MessageType, payload *any.Any) *Message {
	m := &Message{Message: &pb.Message{}}

	m.Type = messageType
	m.Payload = payload
	return m
}

func CreateFromSerialized(serializedMessage []byte) *Message {
	var m = &Message{sm: serializedMessage}

	e := proto.Unmarshal(serializedMessage, m)
	if e != nil {
		log.Warning("Can't deserialize message", e)
	}

	return m
}
