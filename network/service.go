package network

import (
	msg "github.com/poslibp2p/message"
)

type Service interface {
	//Send message to particular peer
	SendMessage(peer *Peer, msg *msg.Message)

	//Send message to a random peer
	SendMessageToRandomPeerAndGetResponse(req *msg.Message) (resp *msg.Message)

	//Broadcast message to all peers
	Broadcast(msg *msg.Message)
}

type ServiceImpl struct {
}

func (s *ServiceImpl) SendMessage(peer *Peer, msg *msg.Message) {

}
func (s *ServiceImpl) SendMessageToRandomPeerAndGetResponse(msg *msg.Message) (resp *msg.Message) {
	return nil
}

func (s *ServiceImpl) Broadcast(msg *msg.Message) {

}
