package network

import (
	"context"
	"github.com/gogo/protobuf/io"
	"github.com/gogo/protobuf/proto"
	"github.com/jbenet/go-context/io"
	"github.com/libp2p/go-libp2p-net"
	"github.com/libp2p/go-libp2p-peer"
	"github.com/libp2p/go-libp2p-protocol"
	msg "github.com/poslibp2p/message"
	"github.com/poslibp2p/message/protobuff"
	"math/rand"
)

type Service interface {
	//Send message to particular peer

	SendMessageTriggered(peer *msg.Peer, msg *msg.Message, trigger chan interface{})

	SendMessage(peer *msg.Peer, msg *msg.Message)

	//Send message to a random peer
	SendRequestToRandomPeer(req *msg.Message) (resp chan *msg.Message)

	//Broadcast message to all peers
	Broadcast(msg *msg.Message)
}

const Libp2pProtocol protocol.ID = "/Libp2pProtocol/1.0.0"

type ServiceImpl struct {
	node       *Node
	dispatcher *msg.Dispatcher
}

func CreateService(node *Node, dispatcher *msg.Dispatcher) Service {
	impl := &ServiceImpl{
		node:       node,
		dispatcher: dispatcher,
	}

	impl.node.Host.SetStreamHandler(Libp2pProtocol, impl.handleNewStream)
	return impl
}

func (s *ServiceImpl) SendMessage(peer *msg.Peer, msg *msg.Message) {
	go func() {
		stream, e := s.node.Host.NewStream(context.Background(), peer.GetPeerInfo().ID, Libp2pProtocol)
		if e != nil {
			log.Error("Can't open stream to peer", e)
			return
		}
		writer := io.NewDelimitedWriter(stream)
		if err := writer.WriteMsg(msg); err != nil {
			log.Error("Can't write message to stream", e)
			return
		}
	}()

}

func (s *ServiceImpl) SendRequestToRandomPeer(req *msg.Message) (resp chan *msg.Message) {
	resp = make(chan *msg.Message)

	go func() {
		connected := s.node.Host.Network().Peers()
		pid := randomSubsetOfIds(connected, 1)[0]

		stream, e := s.node.Host.NewStream(context.Background(), pid, Libp2pProtocol)
		if e != nil {
			log.Error("Can't open stream to peer", e)
			close(resp)
		}

		writer := io.NewDelimitedWriter(stream)
		if err := writer.WriteMsg(req); err != nil {
			log.Error("Can't write message to stream", e)
			close(resp)
		}

		cr := ctxio.NewReader(context.Background(), stream)
		r := io.NewDelimitedReader(cr, net.MessageSizeMax) //TODO decide on msg size

		respMsg := &pb.Message{}
		if err := r.ReadMsg(respMsg); err != nil {
			_ = stream.Reset()
			log.Error("Got error while reading response", e)
			close(resp)
		}
		resp <- &msg.Message{Message: respMsg}
	}()

	return resp
}

func (s *ServiceImpl) SendMessageTriggered(peer *msg.Peer, msg *msg.Message, trigger chan interface{}) {
	<-trigger
	s.SendMessage(peer, msg)
}

func (s *ServiceImpl) Broadcast(msg *msg.Message) {
	go func() {
		bytes, e := proto.Marshal(msg.Message)
		if e != nil {
			log.Error("Can't marshall message", e)
		}

		e = s.node.PubSub.Publish(context.Background(), "shard", bytes)
		if e != nil {
			log.Error("Can't broadcast message", e)
		}
	}()
}

func (s *ServiceImpl) handleNewStream(stream net.Stream) {
	go s.handleNewMessage(stream)
}

func (s *ServiceImpl) handleNewMessage(stream net.Stream) {
	defer stream.Close() //TODO not sure we must close it here, find out
	cr := ctxio.NewReader(context.Background(), stream)
	r := io.NewDelimitedReader(cr, net.MessageSizeMax)

	m := &pb.Message{}
	if err := r.ReadMsg(m); err != nil {
		stream.Reset()
		log.Error("Error while reading message", err)
	}
	s.dispatcher.Dispatch(&msg.Message{
		Message: m,
	})
}

func randomSubsetOfIds(ids []peer.ID, max int) (out []peer.ID) {
	n := IntMin(max, len(ids))
	for _, val := range rand.Perm(len(ids)) {
		out = append(out, ids[val])
		if len(out) >= n {
			break
		}
	}
	return out
}
