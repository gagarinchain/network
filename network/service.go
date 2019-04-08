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

	SendMessageTriggered(ctx context.Context, peer *msg.Peer, msg *msg.Message, trigger chan interface{})

	SendMessage(ctx context.Context, peer *msg.Peer, msg *msg.Message) (resp chan *msg.Message, err chan error)

	//Send message to a random peer
	SendRequestToRandomPeer(ctx context.Context, req *msg.Message) (resp chan *msg.Message, err chan error)

	//Broadcast message to all peers
	Broadcast(ctx context.Context, msg *msg.Message)
}

const Libp2pProtocol protocol.ID = "/Libp2pProtocol/1.0.0"
const Topic string = "/hotstuff"

//TODO find out whether we have to cache streams and synchronize access to them
//TODO handle contexts correctly
type ServiceImpl struct {
	node       *Node
	dispatcher *msg.Dispatcher
	//streams map[peer.ID]net.Stream
}

func CreateService(ctx context.Context, node *Node, dispatcher *msg.Dispatcher) Service {
	impl := &ServiceImpl{
		node:       node,
		dispatcher: dispatcher,
	}

	impl.node.Host.SetStreamHandler(Libp2pProtocol, impl.handleNewStreamWithContext(ctx))
	return impl
}

func (s *ServiceImpl) SendMessage(ctx context.Context, peer *msg.Peer, m *msg.Message) (resp chan *msg.Message, err chan error) {
	resp = make(chan *msg.Message)
	err = make(chan error)

	go func() {
		stream, e := s.node.Host.NewStream(ctx, peer.GetPeerInfo().ID, Libp2pProtocol)
		if e != nil {
			err <- e
			close(resp)
			return
		}
		writer := protoio.NewDelimitedWriter(stream)
		if e := writer.WriteMsg(m); e != nil {
			err <- e
			close(resp)
			return
		}
		close(resp)
	}()

	return resp, err
}

func (s *ServiceImpl) SendRequestToRandomPeer(ctx context.Context, req *msg.Message) (resp chan *msg.Message, err chan error) {
	resp = make(chan *msg.Message)
	err = make(chan error)

	go func() {
		connected := s.node.Host.Network().Peers()
		pid := randomSubsetOfIds(connected, 1)[0]

		stream, e := s.node.Host.NewStream(ctx, pid, Libp2pProtocol)
		if e != nil {
			err <- e
			close(resp)
		}

		writer := protoio.NewDelimitedWriter(stream)
		if e := writer.WriteMsg(req); e != nil {
			err <- e
			close(resp)
		}

		cr := ctxio.NewReader(ctx, stream)
		r := protoio.NewDelimitedReader(cr, net.MessageSizeMax) //TODO decide on msg size

		respMsg := &pb.Message{}
		if e := r.ReadMsg(respMsg); e != nil {
			_ = stream.Reset()
			err <- e
			close(resp)
		}
		resp <- &msg.Message{Message: respMsg}
	}()

	return resp, err
}

func (s *ServiceImpl) SendMessageTriggered(ctx context.Context, peer *msg.Peer, msg *msg.Message, trigger chan interface{}) {
	<-trigger
	s.SendMessage(ctx, peer, msg)
}

func (s *ServiceImpl) Broadcast(ctx context.Context, msg *msg.Message) {
	go func() {
		bytes, e := proto.Marshal(msg.Message)
		if e != nil {
			log.Error("Can't marshall message", e)
		}

		e = s.node.PubSub.Publish(ctx, Topic, bytes)
		if e != nil {
			log.Error("Can't broadcast message", e)
		}
	}()
}

func (s *ServiceImpl) handleNewMessage(ctx context.Context, stream net.Stream) {
	defer stream.Close() //TODO not sure we must close it here, find out
	cr := ctxio.NewReader(ctx, stream)
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

func (s *ServiceImpl) handleNewStreamWithContext(ctx context.Context) net.StreamHandler {
	return func(stream net.Stream) {
		go s.handleNewMessage(ctx, stream)
	}
}

func (n *Node) SubscribeAndListen(ctx context.Context, msgChan chan *msg.Message) {
	// Subscribe to the topic
	sub, err := n.PubSub.SubscribeAndProvide(ctx, Topic)
	if err != nil {
		log.Fatal(err)
	}
	log.Info("Listening topic...")
	for {
		m, err := sub.Next(ctx)
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
