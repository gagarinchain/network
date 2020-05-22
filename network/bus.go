package network

import (
	"context"
	"github.com/gagarinchain/common"
	cmn "github.com/gagarinchain/common/eth/common"
	pb "github.com/gagarinchain/common/protobuff"
	protoio "github.com/gogo/protobuf/io"
	"github.com/golang/protobuf/ptypes"
	ctxio "github.com/jbenet/go-context/io"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"io"
	"sync"
)

type GagarinEventBus struct {
	streams map[peer.ID]network.Stream
	sl      sync.RWMutex

	events   chan *common.Event
	handlers map[pb.Request_RequestType]Handler
}

type Handler func(req *pb.Request) *pb.Event

func NewGagarinEventBus(events chan *common.Event) *GagarinEventBus {
	return &GagarinEventBus{
		events:   events,
		sl:       sync.RWMutex{},
		streams:  make(map[peer.ID]network.Stream),
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

func (n *GagarinEventBus) handleNewMessage(ctx context.Context, s *ServiceImpl, stream network.Stream) {
	log.Debug("opened new gagarin stream")
	cr := ctxio.NewReader(ctx, stream)
	r := protoio.NewDelimitedReader(cr, network.MessageSizeMax)

	n.sl.Lock()
	if _, f := n.streams[stream.Conn().RemotePeer()]; !f {
		n.streams[stream.Conn().RemotePeer()] = stream
	}
	n.sl.Unlock()

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
			n.sl.Lock()
			delete(n.streams, stream.Conn().RemotePeer())
			n.sl.Unlock()

			return
		}

		//info := s.node.Host.Peerstore().PeerInfo(stream.Conn().RemotePeer())
		//p := common.CreatePeer(nil, nil, &info)
		log.Debug(req)
		n.dispatch(req, stream)
	}
}

func (n *GagarinEventBus) dispatch(req *pb.Request, stream network.Stream) {
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
	for {
		select {
		case e := <-n.events:
			if errs := n.notify(ctx, e); errs != nil && len(errs) > 0 {
				n.sl.Lock()
				for p, e := range errs {
					delete(n.streams, p)
					log.Error("error while notify", e)
					continue

				}
				n.sl.Unlock()
			}
		case <-ctx.Done():
			log.Info("Stopped EventBus")
		}
	}
}

func (n *GagarinEventBus) notify(ctx context.Context, req *common.Event) map[peer.ID]error {
	if n.streams == nil {
		return nil
	}
	errors := make(map[peer.ID]error)
	n.sl.RLock()
	for _, v := range n.streams {
		e := n.notifyPeer(req, v)
		if e != nil {
			errors[v.Conn().RemotePeer()] = e
		}
	}
	n.sl.RUnlock()

	return errors
}

func (n *GagarinEventBus) notifyPeer(req *common.Event, stream network.Stream) error {
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
