package network

import (
	"context"
	"github.com/davecgh/go-spew/spew"
	"github.com/gagarinchain/network/common"
	cmn "github.com/gagarinchain/network/common/eth/common"
	msg "github.com/gagarinchain/network/common/message"
	pb "github.com/gagarinchain/network/common/protobuff"
	protoio "github.com/gogo/protobuf/io"
	"github.com/golang/protobuf/ptypes"
	ctxio "github.com/jbenet/go-context/io"
	"github.com/libp2p/go-libp2p-core/network"
	"io"
)

type GagarinEventBus struct {
	stream network.Stream
	events chan *common.Event
}

func NewGagarinEventBus(events chan *common.Event) *GagarinEventBus {
	return &GagarinEventBus{events: events}
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

	n.stream = stream

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
			n.stream = nil
			return
		}

		info := s.node.Host.Peerstore().PeerInfo(stream.Conn().RemotePeer())
		p := common.CreatePeer(nil, nil, &info)
		fromProto := msg.CreateMessageFromProto(m, p, stream)
		spew.Dump(fromProto)
	}
}

func (n *GagarinEventBus) Run(ctx context.Context) {
	for {
		select {
		case e := <-n.events:
			if err := n.notify(ctx, e); err != nil {
				log.Error("error while notify", err)
				continue
			}
		case <-ctx.Done():
			log.Info("Stopped EventBus")
		}
	}
}

func (n *GagarinEventBus) notify(ctx context.Context, req *common.Event) error {
	if n.stream == nil {
		return nil
	}
	writer := protoio.NewDelimitedWriter(n.stream)

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
	case common.ChangedView:
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
	}

	if e := writer.WriteMsg(event); e != nil {
		return e
	}
	return nil
}
