package message

import (
	"github.com/gagarinchain/network/common"
	"github.com/gagarinchain/network/common/protobuff"
	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/libp2p/go-libp2p-net"
)

type Message struct {
	source *common.Peer
	stream net.Stream
	sm     []byte
	*pb.Message
}

func (m *Message) SetStream(stream net.Stream) {
	m.stream = stream
}

func (m *Message) Stream() net.Stream {
	return m.stream
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

func CreateMessageFromProto(message *pb.Message, source *common.Peer, stream net.Stream) *Message {
	m := &Message{Message: message, source: source, stream: stream}
	return m
}

func CreateFromSerialized(serializedMessage []byte, source *common.Peer) *Message {
	m := &pb.Message{}

	e := proto.Unmarshal(serializedMessage, m)
	if e != nil {
		log.Warning("Can't deserialize message", e)
	}

	return &Message{sm: serializedMessage, Message: m, source: source}
}
