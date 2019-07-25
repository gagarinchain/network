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

	SendMessage(ctx context.Context, peer *common.Peer, msg *msg.Message)

	SendResponse(ctx context.Context, msg *msg.Message)

	SendRequest(ctx context.Context, peer *common.Peer, msg *msg.Message) (resp chan *msg.Message, err chan error)

	//Send message to a random peer
	SendRequestToRandomPeer(ctx context.Context, req *msg.Message) (resp chan *msg.Message, err chan error)

	//Broadcast message to all peers
	Broadcast(ctx context.Context, msg *msg.Message)

	Bootstrap(ctx context.Context) (chan int, chan error)
}

type TopicListener interface {
	Listen(ctx context.Context, sub *pubsub.Subscription)
	Subscribe(ctx context.Context) (*pubsub.Subscription, error)
}

const Libp2pProtocol protocol.ID = "/Libp2pProtocol/1.0.0"
const HotstuffTopic string = "/hotstuff"
const TransactionTopic string = "/tx"

//TODO find out whether we have to cache streams and synchronize access to them
//TODO handle contexts correctly
type ServiceImpl struct {
	node             *Node
	dispatcher       msg.Dispatcher
	hotstuffListener TopicListener
	txListener       TopicListener
}

type TopicListenerImpl struct {
	node       *Node
	topicName  string
	dispatcher msg.Dispatcher
}

func NewTopicListenerImpl(node *Node, topicName string, dispatcher msg.Dispatcher) TopicListener {
	return &TopicListenerImpl{node: node, topicName: topicName, dispatcher: dispatcher}
}

func CreateService(ctx context.Context, node *Node, dispatcher msg.Dispatcher, txDispatcher msg.Dispatcher) Service {
	listener := NewTopicListenerImpl(node, HotstuffTopic, dispatcher)
	txListener := NewTopicListenerImpl(node, TransactionTopic, txDispatcher)

	impl := &ServiceImpl{
		node:             node,
		hotstuffListener: listener,
		txListener:       txListener,
		dispatcher:       dispatcher,
	}

	impl.node.Host.SetStreamHandler(Libp2pProtocol, impl.handleNewStreamWithContext(ctx))
	return impl
}

func (s *ServiceImpl) SendMessage(ctx context.Context, peer *common.Peer, m *msg.Message) {
	s.sendRequestAsync(ctx, peer.GetPeerInfo().ID, m, false)
}

func (s *ServiceImpl) SendResponse(ctx context.Context, m *msg.Message) {
	go func() {
		log.Debug("Sending response")
		stream := m.Stream()
		writer := protoio.NewDelimitedWriter(stream)
		spew.Dump(m)
		if e := writer.WriteMsg(m.Message); e != nil {
			log.Error("Response not sent", e)
			return
		}
		log.Debug("Response sent")
	}()
}

func (s *ServiceImpl) SendRequest(ctx context.Context, peer *common.Peer, req *msg.Message) (resp chan *msg.Message, err chan error) {
	return s.sendRequestAsync(ctx, peer.GetPeerInfo().ID, req, true)
}

func (s *ServiceImpl) sendRequestAsync(ctx context.Context, pid peer.ID, req *msg.Message, withResponse bool) (resp chan *msg.Message, err chan error) {
	resp = make(chan *msg.Message)
	err = make(chan error)
	go func(ctx context.Context, pid peer.ID, m *msg.Message, withResponse bool) {
		message, e := s.sendRequestSync(ctx, pid, m, withResponse)
		if e != nil {
			err <- e
			close(resp)
		} else if !withResponse {
			close(resp)
		} else {
			resp <- message
		}
	}(ctx, pid, req, withResponse)

	return resp, err
}

func (s *ServiceImpl) SendRequestToRandomPeer(ctx context.Context, req *msg.Message) (resp chan *msg.Message, err chan error) {
	connected := s.node.Host.Network().Peers()
	pid := randomSubsetOfIds(connected, 1)[0]

	return s.sendRequestAsync(ctx, pid, req, true)

}

func (s *ServiceImpl) sendRequestSync(ctx context.Context, pid peer.ID, req *msg.Message, withResponse bool) (resp *msg.Message, err error) {
	stream, e := s.node.Host.NewStream(ctx, pid, Libp2pProtocol)
	if e != nil {
		return nil, e
	}

	writer := protoio.NewDelimitedWriter(stream)
	if e := writer.WriteMsg(req.Message); e != nil {
		return nil, e
	}

	if !withResponse {
		if err := stream.Close(); err != nil {
			log.Debugf("error while closing stream %e", err)
		}
		return nil, nil
	}

	cr := ctxio.NewReader(ctx, stream)
	r := protoio.NewDelimitedReader(cr, net.MessageSizeMax) //TODO decide on msg size

	respMsg := &pb.Message{}
	if e := r.ReadMsg(respMsg); e != nil {
		_ = stream.Reset()
		return nil, e
	}
	info := s.node.Host.Peerstore().PeerInfo(stream.Conn().RemotePeer())
	p := common.CreatePeer(nil, nil, &info)
	return msg.CreateMessageFromProto(respMsg, p, nil), nil
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

		e = s.node.PubSub.Publish(ctx, HotstuffTopic, bytes)
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
				if err := stream.Reset(); err != nil {
					log.Error("error resetting stream", err)
				}
				log.Infof("error reading message from %s: %s", stream.Conn().RemotePeer(), err)
			} else {
				// Just be nice. They probably won't read this
				// but it doesn't hurt to send it.
				if err := stream.Close(); err != nil {
					log.Error("error closing stream", err)
				}
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

func (s *ServiceImpl) Bootstrap(ctx context.Context) (chan int, chan error) {
	statusChan := make(chan int)
	errChan := make(chan error)
	go func() {
		p, e := s.hotstuffListener.Subscribe(ctx)
		if e != nil {
			errChan <- e
			return
		}

		tx, e := s.txListener.Subscribe(ctx)
		if e != nil {
			errChan <- e
			return
		}

		go func() {
			s.hotstuffListener.Listen(ctx, p)
		}()
		go func() {
			s.txListener.Listen(ctx, tx)
		}()

		statusChan <- 1
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

func (l *TopicListenerImpl) Subscribe(ctx context.Context) (*pubsub.Subscription, error) {
	// Subscribe to the topic
	return l.node.PubSub.SubscribeAndProvide(ctx, l.topicName)
}

func (l *TopicListenerImpl) Listen(ctx context.Context, sub *pubsub.Subscription) {
	log.Infof("Listening topic %v", l.topicName)
	for {
		e := l.handleTopicMessage(ctx, sub)
		if e == context.Canceled {
			break
		}
		if e != nil {
			log.Error(e)
		}
	}

}

func (l *TopicListenerImpl) handleTopicMessage(ctx context.Context, sub *pubsub.Subscription) error {
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
	info := l.node.Host.Peerstore().PeerInfo(pid)
	message := msg.CreateFromSerialized(m.Data, common.CreatePeer(nil, nil, &info))
	l.dispatcher.Dispatch(message)

	return nil
}
