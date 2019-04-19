package network

import (
	"context"
	"github.com/davecgh/go-spew/spew"
	protoio "github.com/gogo/protobuf/io"
	"github.com/gogo/protobuf/proto"
	"github.com/jbenet/go-context/io"
	"github.com/libp2p/go-libp2p-net"
	"github.com/libp2p/go-libp2p-peer"
	"github.com/libp2p/go-libp2p-protocol"
	"github.com/libp2p/go-libp2p-pubsub"
	"github.com/poslibp2p/common"
	msg "github.com/poslibp2p/common/message"
	"github.com/poslibp2p/common/protobuff"
	"io"
	"math/rand"
)

type Service interface {
	//Send message to particular peer

	SendMessageTriggered(ctx context.Context, peer *common.Peer, msg *msg.Message, trigger chan interface{})

	SendMessage(ctx context.Context, peer *common.Peer, msg *msg.Message) (resp chan *msg.Message, err chan error)

	//Send message to a random peer
	SendRequestToRandomPeer(ctx context.Context, req *msg.Message) (resp chan *msg.Message, err chan error)

	//Broadcast message to all peers
	Broadcast(ctx context.Context, msg *msg.Message)

	Bootstrap(ctx context.Context) (chan int, chan error)
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

func (s *ServiceImpl) SendMessage(ctx context.Context, peer *common.Peer, m *msg.Message) (resp chan *msg.Message, err chan error) {
	resp = make(chan *msg.Message)
	err = make(chan error)

	go func() {
		log.Debug("Sending response")
		stream := m.Stream()
		if stream == nil {
			log.Debug("Open new stream")
			var e error
			stream, e = s.node.Host.NewStream(ctx, peer.GetPeerInfo().ID, Libp2pProtocol)
			if e != nil {
				err <- e
				close(resp)
				return
			}
		}
		writer := protoio.NewDelimitedWriter(stream)
		spew.Dump(m)
		if e := writer.WriteMsg(m.Message); e != nil {
			log.Debug("Response not sent", e)

			err <- e
			close(resp)
			return
		}
		log.Debug("Response sent")

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
			return
		}

		writer := protoio.NewDelimitedWriter(stream)
		if e := writer.WriteMsg(req.Message); e != nil {
			err <- e
			close(resp)
			return
		}

		cr := ctxio.NewReader(ctx, stream)
		r := protoio.NewDelimitedReader(cr, net.MessageSizeMax) //TODO decide on msg size

		respMsg := &pb.Message{}
		if e := r.ReadMsg(respMsg); e != nil {
			_ = stream.Reset()
			err <- e
			close(resp)
			return
		}
		info := s.node.Host.Peerstore().PeerInfo(stream.Conn().RemotePeer())
		p := common.CreatePeer(nil, nil, &info)
		resp <- msg.CreateMessageFromProto(respMsg, p, nil)
	}()

	return resp, err
}

func (s *ServiceImpl) SendMessageTriggered(ctx context.Context, peer *common.Peer, msg *msg.Message, trigger chan interface{}) {
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
	cr := ctxio.NewReader(ctx, stream)
	r := protoio.NewDelimitedReader(cr, net.MessageSizeMax)

	for {
		m := &pb.Message{}
		err := r.ReadMsg(m)
		if err != nil {
			if err != io.EOF {
				stream.Reset()
				log.Infof("error reading message from %s: %s", stream.Conn().RemotePeer(), err)
			} else {
				// Just be nice. They probably won't read this
				// but it doesn't hurt to send it.
				stream.Close()
			}
			return
		}

		info := s.node.Host.Peerstore().PeerInfo(stream.Conn().RemotePeer())
		p := common.CreatePeer(nil, nil, &info)
		s.dispatcher.Dispatch(msg.CreateMessageFromProto(m, p, stream))
	}
}

func (s *ServiceImpl) handleNewStreamWithContext(ctx context.Context) net.StreamHandler {
	return func(stream net.Stream) {
		s.handleNewMessage(ctx, stream)
	}
}

func (s *ServiceImpl) Subscribe(ctx context.Context) (*pubsub.Subscription, error) {
	// Subscribe to the topic
	return s.node.PubSub.SubscribeAndProvide(ctx, Topic)
}

func (s *ServiceImpl) Listen(ctx context.Context, sub *pubsub.Subscription) {
	log.Info("Listening topic...")
	for {
		e := s.handleMessage(ctx, sub)
		if e == context.Canceled {
			break
		}
		if e != nil {
			log.Error(e)
		}
	}

}

func (s *ServiceImpl) handleMessage(ctx context.Context, sub *pubsub.Subscription) error {
	defer func() {
		if r := recover(); r != nil {
			log.Error("panic occurred: ", r)
		}
	}()

	m, err := sub.Next(ctx)
	if err != nil {
		return err
	}
	pid, err := peer.IDFromBytes(m.From)
	if err != nil {
		return err
	}

	log.Infof("Received Pubsub message from %s\n", pid.Pretty())

	//We do several very easy checks here and give control to dispatcher
	info := s.node.Host.Peerstore().PeerInfo(pid)
	message := msg.CreateFromSerialized(m.Data, common.CreatePeer(nil, nil, &info))
	s.dispatcher.Dispatch(message)

	return nil
}

func (s *ServiceImpl) Bootstrap(ctx context.Context) (chan int, chan error) {
	statusChan := make(chan int)
	errChan := make(chan error)
	go func() {
		sub, e := s.Subscribe(ctx)
		if e != nil {
			errChan <- e
			return
		}
		statusChan <- 1
		s.Listen(ctx, sub)
	}()

	return statusChan, errChan
}

func randomSubsetOfIds(ids []peer.ID, max int) (out []peer.ID) {
	n := IntMin(max, len(ids))
	log.Info(n)
	for _, val := range rand.Perm(len(ids)) {
		out = append(out, ids[val])
		if len(out) >= n {
			break
		}
	}
	return out
}
