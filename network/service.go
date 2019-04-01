package network

import (
	"context"
	protoio "github.com/gogo/protobuf/io"
	"github.com/gogo/protobuf/proto"
	"github.com/jbenet/go-context/io"
	"github.com/libp2p/go-libp2p-net"
	"github.com/libp2p/go-libp2p-peer"
	"github.com/libp2p/go-libp2p-protocol"
	msg "github.com/poslibp2p/message"
	"github.com/poslibp2p/message/protobuff"
	"io"
	"math/rand"
)

type Service interface {
	//Send message to particular peer

	SendMessageTriggered(peer *msg.Peer, msg *msg.Message, trigger chan interface{})

	SendMessage(peer *msg.Peer, msg *msg.Message) (resp chan *msg.Message)

	//Send message to a random peer
	SendRequestToRandomPeer(req *msg.Message) (resp chan *msg.Message)

	//Broadcast message to all peers
	Broadcast(msg *msg.Message)
}

const Libp2pProtocol protocol.ID = "/Libp2pProtocol/1.0.0"
const Topic string = "/hotstuff"

//TODO find out whether we have to cache streams and synchronize access to them
type ServiceImpl struct {
	node       *Node
	dispatcher *msg.Dispatcher
	//streams map[peer.ID]net.Stream
}

func CreateService(node *Node, dispatcher *msg.Dispatcher) Service {
	impl := &ServiceImpl{
		node:       node,
		dispatcher: dispatcher,
	}

	impl.node.Host.SetStreamHandler(Libp2pProtocol, impl.handleNewStream)
	return impl
}

func (s *ServiceImpl) SendMessage(peer *msg.Peer, m *msg.Message) (resp chan *msg.Message) {
	resp = make(chan *msg.Message)

	go func() {
		stream, e := s.node.Host.NewStream(context.Background(), peer.GetPeerInfo().ID, Libp2pProtocol)
		if e != nil {
			log.Error("Can't open stream to peer", e)
			return
		}
		writer := protoio.NewDelimitedWriter(stream)
		if err := writer.WriteMsg(m); err != nil {
			log.Error("Can't write message to stream", e)
			return
		}
		close(resp)
	}()

	return resp
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

		writer := protoio.NewDelimitedWriter(stream)
		if err := writer.WriteMsg(req); err != nil {
			log.Error("Can't write message to stream", e)
			close(resp)
		}

		cr := ctxio.NewReader(context.Background(), stream)
		r := protoio.NewDelimitedReader(cr, net.MessageSizeMax) //TODO decide on msg size

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

		e = s.node.PubSub.Publish(context.Background(), Topic, bytes)
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
	r := protoio.NewDelimitedReader(cr, net.MessageSizeMax)

	m := &pb.Message{}
	if err := r.ReadMsg(m); err != nil {
		stream.Reset()
		log.Error("Error while reading message", err)
	}
	s.dispatcher.Dispatch(&msg.Message{
		Message: m,
	})
}

func (n *Node) SubscribeAndListen(msgChan chan *msg.Message) {
	// Subscribe to the topic
	sub, err := n.PubSub.SubscribeAndProvide(context.Background(), Topic)
	if err != nil {
		log.Fatal(err)
	}
	log.Info("Listening topic...")
	for {
		m, err := sub.Next(context.Background())
		if err == io.EOF || err == context.Canceled {
			break
		} else if err != nil {
			log.Error(err)
			break
		}
		pid, err := peer.IDFromBytes(m.From)
		if err != nil {
			log.Fatal(err)
		}

		log.Infof("Received Pubsub message: %s from %s\n", string(m.Data), pid.Pretty())

		//We do several very easy checks here and give control to dispatcher
		info := n.Host.Peerstore().PeerInfo(pid)
		message := msg.CreateFromSerialized(m.Data, msg.CreatePeer(nil, nil, &info))
		msgChan <- message
	}

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
