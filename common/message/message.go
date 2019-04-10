package message

import (
	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/poslibp2p/common"
	"github.com/poslibp2p/common/protobuff"
)

type Message struct {
	source *common.Peer
	sm     []byte
	*pb.Message
}

func (m *Message) Source() *common.Peer {
	return m.source
}

func CreateMessage(messageType pb.Message_MessageType, payload *any.Any, source *common.Peer) *Message {
	m := &Message{Message: &pb.Message{}}

	m.Type = messageType
	m.Payload = payload
	m.source = source
	return m
}

func CreateMessageFromProto(message *pb.Message, source *common.Peer) *Message {
	m := &Message{Message: message, source: source}
	return m
}

func CreateFromSerialized(serializedMessage []byte, source *common.Peer) *Message {
	var m = &Message{sm: serializedMessage}

	e := proto.Unmarshal(serializedMessage, m)
	if e != nil {
		log.Warning("Can't deserialize message", e)
	}

	m.source = source
	return m
}
