package message

import (
	"crypto/ecdsa"
	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/poslibp2p/eth/crypto"
	"github.com/poslibp2p/message/protobuff"
)

type Message struct {
	sm []byte
	*pb.Message
}

func CreateMessage(messageType pb.Message_MessageType, privateKey *ecdsa.PrivateKey, payload *any.Any) *Message {
	bytes := payload.Value
	hash := crypto.Keccak256(bytes)

	m := &Message{Message: &pb.Message{}}

	m.Type = messageType
	m.Payload = payload
	sig, err := crypto.Sign(hash, privateKey)

	if err != nil {
		log.Error("Can't sign message", err)
	}
	m.Signature = sig

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
