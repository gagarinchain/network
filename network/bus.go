package network

import (
	"context"
	"github.com/gagarinchain/common"
	cmn "github.com/gagarinchain/common/eth/common"
	pb "github.com/gagarinchain/common/protobuff"
	protoio "github.com/gagarinchain/common/protobuff/io"
	"github.com/golang/protobuf/ptypes"
	ctxio "github.com/jbenet/go-context/io"
	"github.com/libp2p/go-libp2p-core/network"
	"io"
	"sync"
)

type GagarinEventBus struct {
	streams map[string]io.Writer
	sl      sync.RWMutex

	events   chan *common.Event
	handlers map[pb.Request_RequestType]Handler
}

type Handler func(req *pb.Request) *pb.Event

func NewGagarinEventBus(events chan *common.Event) *GagarinEventBus {
	return &GagarinEventBus{
		events:   events,
		sl:       sync.RWMutex{},
		streams:  make(map[string]io.Writer),
		handlers: make(map[pb.Request_RequestType]Handler),
	}
}
func (n *GagarinEventBus) AddHandler(t pb.Request_RequestType, h Handler) {
	n.handlers[t] = h
}

func (n *GagarinEventBus) FireEvent(event *common.Event) {
	go func() {
		n.events <- event
	}()
}

func (n *GagarinEventBus) Subscribe(id string, writer io.Writer) {
	n.sl.Lock()
	if _, f := n.streams[id]; !f {
		n.streams[id] = writer
	}
	n.sl.Unlock()
}
func (n *GagarinEventBus) Unsubscribe(id string) {
	n.sl.Lock()
	delete(n.streams, id)
	n.sl.Unlock()
}

func (n *GagarinEventBus) GetStreamsCopy() map[string]io.Writer {
	n.sl.RLock()
	scopy := make(map[string]io.Writer)
	for id, s := range n.streams {
		scopy[id] = s
	}
	n.sl.RUnlock()
	return scopy
}

func (n *GagarinEventBus) handleNewMessage(ctx context.Context, s *ServiceImpl, stream network.Stream) {
	log.Debug("opened new gagarin stream")
	cr := ctxio.NewReader(ctx, stream)
	r := protoio.NewDelimitedReader(cr, network.MessageSizeMax)

	n.Subscribe(stream.Conn().RemotePeer().Pretty(), stream)

	for {
		log.Debug("reading")
		req := &pb.Request{}
		err := r.ReadMsg(req)
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
			n.Unsubscribe(stream.Conn().RemotePeer().Pretty())
			return
		}

		//info := s.node.Host.Peerstore().PeerInfo(stream.Conn().RemotePeer())
		//p := common.CreatePeer(nil, nil, &info)
		log.Debug(req)
		n.dispatch(req, stream)
	}
}

func (n *GagarinEventBus) dispatch(req *pb.Request, stream io.Writer) {
	h, f := n.handlers[req.Type]

	if !f {
		log.Errorf("No handler is found for %v", req.Type)
		return
	}

	resp := h(req)
	log.Debug(resp)
	writer := protoio.NewDelimitedWriter(stream)
	if e := writer.WriteMsg(resp); e != nil {
		//TODO mb close stream here
		log.Error("Error while notify", e)
	}

}

func (n *GagarinEventBus) Run(ctx context.Context) {
	log.Info("Running event buss")
	errorChan := make(chan string)
	for {
		select {
		case e := <-n.events:
			n.notify(ctx, e, errorChan)
		case id := <-errorChan:
			n.Unsubscribe(id)
		case <-ctx.Done():
			log.Info("Stopped EventBus")
			return
		}
	}
}

func (n *GagarinEventBus) notify(ctx context.Context, req *common.Event, errorChan chan string) {
	if n.streams == nil {
		return
	}

	streams := n.GetStreamsCopy()

	for id, s := range streams {
		go func(id string, s io.Writer) {
			e := n.notifyPeer(req, s)
			if e != nil {
				log.Error(e)
				errorChan <- id
			}
		}(id, s)
	}

	return
}

func (n *GagarinEventBus) notifyPeer(req *common.Event, stream io.Writer) error {
	writer := protoio.NewDelimitedWriter(stream)
	var event *pb.Event
	switch req.T {
	case common.BlockAdded:
		block := req.Payload.(*pb.Block)
		any, e := ptypes.MarshalAny(block)
		if e != nil {
			log.Error(e)
		}
		event = &pb.Event{
			Type:    pb.Event_BLOCK_ADDED,
			Payload: any,
		}
	case common.EpochStarted:
		payload := req.Payload.(*pb.EpochStartedPayload)
		any, e := ptypes.MarshalAny(payload)
		if e != nil {
			log.Error(e)
		}
		event = &pb.Event{
			Type:    pb.Event_EPOCH_STARTED,
			Payload: any,
		}
	case common.ViewChanged:
		payload := req.Payload.(*pb.ViewChangedPayload)
		any, e := ptypes.MarshalAny(payload)
		if e != nil {
			log.Error(e)
		}
		event = &pb.Event{
			Type:    pb.Event_VIEW_CHANGED,
			Payload: any,
		}
	case common.Committed:
		payload := req.Payload.(cmn.Hash)
		any, e := ptypes.MarshalAny(&pb.CommittedPayload{Hash: payload.Bytes()})
		if e != nil {
			log.Error(e)
		}
		event = &pb.Event{
			Type:    pb.Event_COMMITTED,
			Payload: any,
		}
	case common.BalanceUpdated:
		payload := req.Payload.(*pb.AccountUpdatedPayload)
		any, e := ptypes.MarshalAny(payload)
		if e != nil {
			log.Error(e)
		}
		event = &pb.Event{
			Type:    pb.Event_ACCOUNT,
			Payload: any,
		}

	}

	if e := writer.WriteMsg(event); e != nil {
		//TODO mb close stream here
		return e
	}
	return nil
}
