package network

import (
	"context"
	"github.com/gagarinchain/common"
	msg "github.com/gagarinchain/common/message"
	"github.com/gagarinchain/common/protobuff"
	protoio "github.com/gagarinchain/common/protobuff/io"
	"github.com/golang/protobuf/proto"
	"github.com/jbenet/go-context/io"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/libp2p/go-libp2p-pubsub"
	"io"
	"math/rand"
	"time"
)

type Service interface {
	//Send message to particular peer

	SendMessageTriggered(ctx context.Context, peer *common.Peer, msg *msg.Message, trigger chan interface{})

	SendMessage(ctx context.Context, peer *common.Peer, msg *msg.Message)

	SendResponse(ctx context.Context, msg *msg.Message)

	SendRequest(ctx context.Context, peer *common.Peer, msg *msg.Message) (resp chan *msg.Message, err chan error)

	//Send message to a random peer
	//TODO return peer that was chosen
	SendRequestToRandomPeer(ctx context.Context, req *msg.Message) (resp chan *msg.Message, err chan error)

	//Broadcast message to all peers
	Broadcast(ctx context.Context, msg *msg.Message)
	BroadcastTransaction(ctx context.Context, msg *msg.Message)

	Bootstrap(ctx context.Context, cfg *BootstrapConfig) (chan int, chan error)
}

type TopicListener interface {
	Listen(ctx context.Context, sub *pubsub.Subscription)
	Subscribe(ctx context.Context) (*pubsub.Subscription, error)
	Provide(ctx context.Context) error
}

const BlockchainProtocol protocol.ID = "/gagarin/bc/1.0.0"
const GagarinProtocol protocol.ID = "/gagarin/ext/1.0.0"
const TransactionProtocol protocol.ID = "/gagarin/tx/1.0.0"
const HotstuffTopic string = "/hotstuff"
const TransactionTopic string = "/tx"

//TODO find out whether we have to cache streams and synchronize access to them
//TODO handle contexts correctly
type ServiceImpl struct {
	node              *Node
	dispatcher        msg.Dispatcher
	hotstuffListener  TopicListener
	txListener        TopicListener
	bus               *GagarinEventBus
	connectionTimeout time.Duration
}

type TopicListenerImpl struct {
	node       *Node
	topicName  string
	dispatcher msg.Dispatcher
}

func NewTopicListenerImpl(node *Node, topicName string, dispatcher msg.Dispatcher) TopicListener {
	return &TopicListenerImpl{node: node, topicName: topicName, dispatcher: dispatcher}
}

func CreateService(ctx context.Context, node *Node, dispatcher msg.Dispatcher, txDispatcher msg.Dispatcher, bus *GagarinEventBus) Service {
	listener := NewTopicListenerImpl(node, HotstuffTopic, dispatcher)
	txListener := NewTopicListenerImpl(node, TransactionTopic, txDispatcher)

	impl := &ServiceImpl{
		node:             node,
		hotstuffListener: listener,
		txListener:       txListener,
		dispatcher:       dispatcher,
		bus:              bus,
	}

	impl.node.Host.SetStreamHandler(BlockchainProtocol, impl.handleNewStreamWithContext(ctx, dispatcher))
	impl.node.Host.SetStreamHandler(TransactionProtocol, impl.handleNewStreamWithContext(ctx, txDispatcher))
	impl.node.Host.SetStreamHandler(GagarinProtocol, impl.handleGagarinWithContext(ctx))
	return impl
}

func (s *ServiceImpl) SendMessage(ctx context.Context, peer *common.Peer, m *msg.Message) {
	resp, err := s.sendRequestAsync(ctx, peer.GetPeerInfo().ID, m, BlockchainProtocol, false)
	select {
	case <-resp:
		log.Debugf("sent successfully to %v", peer.GetAddress().Hex())
	case e := <-err:
		log.Errorf("error (%v) sending message %v to %v", e.Error(), m.Type.String(), peer.GetAddress().Hex())
	}
}

func (s *ServiceImpl) SendResponse(ctx context.Context, m *msg.Message) {
	timeout, _ := context.WithTimeout(ctx, s.connectionTimeout)
	go func(ctx context.Context) {
		log.Debug("Sending response")
		stream := m.Stream()
		wr := ctxio.NewWriter(timeout, stream)

		writer := protoio.NewDelimitedWriter(wr)
		if e := writer.WriteMsg(m.Message); e != nil {
			log.Error("Response not sent", e)
			return
		}
		log.Debug("Response sent")
	}(timeout)
}

func (s *ServiceImpl) SendRequest(ctx context.Context, peer *common.Peer, req *msg.Message) (resp chan *msg.Message, err chan error) {
	return s.sendRequestAsync(ctx, peer.GetPeerInfo().ID, req, BlockchainProtocol, true)
}

func (s *ServiceImpl) sendRequestAsync(ctx context.Context, pid peer.ID, req *msg.Message, protocol protocol.ID, withResponse bool) (resp chan *msg.Message, err chan error) {
	resp = make(chan *msg.Message)
	err = make(chan error)

	//we should handle loop messages in a special way, since self dialing is not allowed now, but will be implemented in the future
	//https://github.com/libp2p/go-libp2p/issues/328#issuecomment-465264415
	if s.node.GetPeerInfo().ID == pid {
		log.Debug("Sending message to self")
		go func() {
			s.dispatcher.Dispatch(req)
			close(resp)
		}()
		return resp, err
	}

	timeout, _ := context.WithTimeout(ctx, s.connectionTimeout)
	go func(ctx context.Context, pid peer.ID, m *msg.Message, withResponse bool) {
		message, e := s.sendRequestSync(ctx, pid, m, protocol, withResponse)
		if e != nil {
			err <- e
		} else if !withResponse {
			close(resp)
		} else {
			resp <- message
			close(resp)
		}
	}(timeout, pid, req, withResponse)

	return resp, err
}

func (s *ServiceImpl) SendRequestToRandomPeer(ctx context.Context, req *msg.Message) (resp chan *msg.Message, err chan error) {
	connected := s.node.Host.Network().Peers()
	pid := randomSubsetOfIds(connected, 1)[0]

	return s.sendRequestAsync(ctx, pid, req, BlockchainProtocol, true)

}

func (s *ServiceImpl) sendRequestSync(ctx context.Context, pid peer.ID, req *msg.Message, protocol protocol.ID, withResponse bool) (resp *msg.Message, err error) {
	stream, e := s.node.Host.NewStream(ctx, pid, protocol)
	if e != nil {
		return nil, e
	}
	wr := ctxio.NewWriter(ctx, stream)

	writer := protoio.NewDelimitedWriter(wr)
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
	r := protoio.NewDelimitedReader(cr, network.MessageSizeMax) //TODO decide on msg size

	respMsg := &pb.Message{}
	if e := r.ReadMsg(respMsg); e != nil {
		_ = stream.Reset()
		return nil, e
	}
	_ = stream.Close()

	info := s.node.Host.Peerstore().PeerInfo(stream.Conn().RemotePeer())
	p := common.CreatePeer(nil, nil, &info)
	return msg.CreateMessageFromProto(respMsg, p, nil), nil
}

func (s *ServiceImpl) SendMessageTriggered(ctx context.Context, peer *common.Peer, msg *msg.Message, trigger chan interface{}) {
	<-trigger
	s.SendMessage(ctx, peer, msg)
}

func (s *ServiceImpl) Broadcast(ctx context.Context, msg *msg.Message) {
	s.broadcast(ctx, HotstuffTopic, msg)
}

func (s *ServiceImpl) BroadcastTransaction(ctx context.Context, msg *msg.Message) {
	connected := s.node.Host.Network().Peers()
	size := 0
	if len(connected) > 3 {
		size = 3
	} else {
		size = len(connected)
	}

	peers := randomSubsetOfIds(connected, size)

	for _, p := range peers {
		_, err := s.sendRequestSync(ctx, p, msg, TransactionProtocol, false)
		if err != nil {
			log.Errorf("Can't send message to %v peer", p.Pretty(), err)
		}
	}
}

func (s *ServiceImpl) broadcast(ctx context.Context, topic string, msg *msg.Message) {
	timeout, _ := context.WithTimeout(ctx, s.connectionTimeout)
	bytes, e := proto.Marshal(msg.Message)
	if e != nil {
		log.Error("Can't marshall message", e)
	}

	e = s.node.PubSub.Publish(timeout, topic, bytes)
	if e != nil {
		log.Error("Can't broadcast message", e)
	}
}

func (s *ServiceImpl) handleNewMessage(ctx context.Context, stream network.Stream, d msg.Dispatcher) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("panic occurred: ", r)
			if e := stream.Reset(); e != nil {
				log.Error("failed reset stream ")
			}
		}
	}()
	cr := ctxio.NewReader(ctx, stream)
	r := protoio.NewDelimitedReader(cr, network.MessageSizeMax)
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
		log.Infof("Received Direct [%v] message from [%v]", m.Type.String(), p.GetPeerInfo().ID.Pretty())
		d.Dispatch(msg.CreateMessageFromProto(m, p, stream))
	}
}

func (s *ServiceImpl) handleNewStreamWithContext(ctx context.Context, d msg.Dispatcher) network.StreamHandler {
	return func(stream network.Stream) {
		s.handleNewMessage(ctx, stream, d)
	}
}
func (s *ServiceImpl) handleGagarinWithContext(ctx context.Context) network.StreamHandler {
	return func(stream network.Stream) {
		s.bus.handleNewMessage(ctx, s, stream)
	}
}

func (s *ServiceImpl) Bootstrap(ctx context.Context, cfg *BootstrapConfig) (chan int, chan error) {
	s.connectionTimeout = cfg.ConnectionTimeout
	statusChan := make(chan int)
	errChan := make(chan error)
	go func() {
		p, e := s.hotstuffListener.Subscribe(ctx)
		if e != nil {
			errChan <- e
			return
		}

		e = s.txListener.Provide(ctx)
		if e != nil {
			errChan <- e
			return
		}

		go func() {
			s.hotstuffListener.Listen(ctx, p)
		}()

		//currently comment tx topic listening
		//go func() {
		//	s.txListener.Listen(ctx, tx)
		//}()

		statusChan <- 1
	}()

	return statusChan, errChan
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

func (l *TopicListenerImpl) Subscribe(ctx context.Context) (*pubsub.Subscription, error) {
	// Subscribe to the topic
	return l.node.PubSub.SubscribeAndProvide(ctx, l.topicName)
}

func (l *TopicListenerImpl) Provide(ctx context.Context) error {
	id, e := NewTopicCid(l.topicName).CID()
	if e != nil {
		return e
	}

	return l.node.PubSub.Routing.Provide(ctx, *id, true)
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

	//We do several very easy checks here and give control to dispatcher
	info := l.node.Host.Peerstore().PeerInfo(pid)
	message := msg.CreateFromSerialized(m.Data, common.CreatePeer(nil, nil, &info))
	log.Infof("Received Pubsub message [%v] from [%v]\n", message.Type.String(), pid.Pretty())
	l.dispatcher.Dispatch(message)

	return nil
}
