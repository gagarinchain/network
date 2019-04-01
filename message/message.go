package message

import (
	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/poslibp2p/message/protobuff"
)

type Message struct {
	source *Peer
	sm     []byte
	*pb.Message
}

func (m *Message) Source() *Peer {
	return m.source
}

func CreateMessage(messageType pb.Message_MessageType, payload *any.Any, source *Peer) *Message {
	m := &Message{Message: &pb.Message{}}

	m.Type = messageType
	m.Payload = payload
	m.source = source
	return m
}

func CreateMessageFromProto(message *pb.Message, source *Peer) *Message {
	m := &Message{Message: message, source: source}
	return m
}

func CreateFromSerialized(serializedMessage []byte, source *Peer) *Message {
	var m = &Message{sm: serializedMessage}

	e := proto.Unmarshal(serializedMessage, m)
	if e != nil {
		log.Warning("Can't deserialize message", e)
	}

	m.source = source
	return m
}
