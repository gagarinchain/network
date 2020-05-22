package blockchain

import (
	"context"
	"errors"
	"fmt"
	"github.com/gagarinchain/common/api"
	"github.com/gagarinchain/common/eth/common"
	msg "github.com/gagarinchain/common/message"
	"github.com/gagarinchain/common/protobuff"
	"github.com/gagarinchain/network/network"
	"github.com/golang/protobuf/ptypes"
	"time"
)

//TODO Consider adding stream-peer cache, so we can reuse opened streams, don't forget that this scheme can produce bottlenecks
//Now we don't use same stream to send response, it means that we pass peer id to open new stream to this peer when sending response, this scheme is redundant too
type BlockProtocol struct {
	srv  network.Service
	bc   api.Blockchain
	sync Synchronizer
	stop chan int
}

var (
	Version     int32 = 1
	HeaderLimit       = 30
)

func CreateBlockProtocol(srv network.Service, bc api.Blockchain, sync Synchronizer) *BlockProtocol {
	return &BlockProtocol{srv: srv, bc: bc, sync: sync, stop: make(chan int)}
}

func (p *BlockProtocol) Bootstrap(ctx context.Context) (respChan chan int, errChan chan error) {
	respChan = make(chan int)
	errChan = make(chan error)
	//todo move to config
	t := time.NewTicker(5 * time.Second)
	go func() {
		for {
			select {
			case <-t.C:
				if err := p.SendHello(ctx); err != nil {
					log.Debug(err)
					errChan <- err
				} else {
					t.Stop()
					respChan <- 0
				}
			case <-ctx.Done():
				t.Stop()
				return
			}
		}
	}()
	go func() {
		if err := p.SendHello(ctx); err != nil {
			log.Debug(err)
			errChan <- err
			return
		}
		t.Stop()
		respChan <- 0
	}()
	return respChan, errChan
}

func (p *BlockProtocol) SendHello(ctx context.Context) error {
	log.Debug("Sending Hello message")
	rq := &pb.Message{Type: pb.Message_HELLO_REQUEST}
	m := &msg.Message{Message: rq}
	mChan, err := p.srv.SendRequestToRandomPeer(ctx, m)
	var resp *msg.Message
	select {
	case b, ok := <-mChan:
		if !ok {
			return errors.New("error while requesting hello, channel is closed")
		} else {
			resp = b
		}
	case err := <-err:
		return err
	}

	if resp.Type != pb.Message_HELLO_RESPONSE {
		return errors.New(fmt.Sprintf("not expected msg type %v response to Hello", resp.Type))
	}

	h := &pb.HelloPayload{}
	if err := ptypes.UnmarshalAny(resp.Payload, h); err != nil {
		return err
	}
	//TODO check here different equivocations, such as very high block heights etc
	if h.GetVersion() != Version {
		return errors.New("wrong version")
	}

	if h.GetTopBlockHeight() > p.bc.GetTopHeight() {
		//log.Info("loading absent blocks from %v to %v, peer %v", p.bc.GetTopHeight(), h.GetTopBlockHeight(), resp.Source().GetPeerInfo().ID.Pretty())
		if err := p.sync.LoadBlocks(ctx, p.bc.GetTopHeight(), h.GetTopBlockHeight(), resp.Source()); err != nil {
			return err
		}
	}

	return nil
}

func (p *BlockProtocol) OnHeaderRequest(ctx context.Context, req *msg.Message) error {
	log.Debug("Got headers request from peer %v", req.Source().GetPeerInfo().ID.Pretty())
	payload := req.GetPayload()
	br := &pb.HeadersRequest{}
	if err := ptypes.UnmarshalAny(payload, br); err != nil {
		return err
	}

	low := br.GetLow()
	high := br.GetHigh()

	if low < 0 || high < 0 {
		return errors.New("negative boundaries")
	}

	if high <= low {
		return errors.New("invalid boundaries")
	}

	var headers []api.Header
	for height := low + 1; height <= high; height++ {
		res := p.bc.GetBlockByHeight(height)
		if len(res) == 0 || len(headers)+len(res) > HeaderLimit {
			break
		}

		for _, b := range res {
			headers = append(headers, b.Header())
		}
	}

	var resp *pb.HeadersResponse
	if len(headers) == 0 {
		resp = createHeadersError(headers)
	} else {
		var blockHeaders []*pb.BlockHeader
		for _, h := range headers {
			pbHeader := h.GetMessage()
			blockHeaders = append(blockHeaders, pbHeader)
		}
		pbHeaders := &pb.Headers{Headers: blockHeaders}
		resp = &pb.HeadersResponse{Response: &pb.HeadersResponse_Headers{Headers: pbHeaders}}
	}

	any, e := ptypes.MarshalAny(resp)
	if e != nil {
		return e
	}
	b := msg.CreateMessage(pb.Message_HEADERS_RESPONSE, any, nil)
	b.SetStream(req.Stream())
	p.srv.SendResponse(ctx, b)
	return nil
}

func createHeadersError(headers []api.Header) *pb.HeadersResponse {
	e := &pb.Error{Code: pb.Error_NOT_FOUND, Desc: "Not found"}
	return &pb.HeadersResponse{Response: &pb.HeadersResponse_ErrorCode{ErrorCode: e}}
}

func (p *BlockProtocol) OnBlockRequest(ctx context.Context, req *msg.Message) error {
	log.Debug("Got block request from peer %v", req.Source().GetPeerInfo().ID.Pretty())
	payload := req.GetPayload()
	br := &pb.BlockRequestPayload{}
	if err := ptypes.UnmarshalAny(payload, br); err != nil {
		return err
	}

	hash := common.BytesToHash(br.GetHash())
	block := p.bc.GetBlockByHash(hash)

	resp := createBlockResponse(block)

	any, e := ptypes.MarshalAny(resp)
	if e != nil {
		return e
	}
	b := msg.CreateMessage(pb.Message_BLOCK_RESPONSE, any, nil)
	b.SetStream(req.Stream())
	p.srv.SendResponse(ctx, b)
	return nil
}

func createBlockResponse(block api.Block) *pb.BlockResponsePayload {
	if block == nil {
		e := &pb.Error{Code: pb.Error_NOT_FOUND, Desc: "Not found"}
		return &pb.BlockResponsePayload{Response: &pb.BlockResponsePayload_ErrorCode{ErrorCode: e}}
	}
	pblock := block.GetMessage()
	p := &pb.BlockResponsePayload_Block{Block: pblock}
	return &pb.BlockResponsePayload{Response: p}
}

func (p *BlockProtocol) OnHello(ctx context.Context, m *msg.Message) error {
	log.Debug("processing hello message")
	if m.GetType() != pb.Message_HELLO_REQUEST {
		return errors.New(fmt.Sprintf("wrong message type, expected %v", pb.Message_HELLO_REQUEST.String()))
	}

	payload := &pb.HelloPayload{}
	payload.Time = time.Now().UnixNano()
	payload.Version = Version
	payload.TopBlockHeight = p.bc.GetHead().Header().Height()

	any, e := ptypes.MarshalAny(payload)
	if e != nil {
		return e
	}
	resp := msg.CreateMessage(pb.Message_HELLO_RESPONSE, any, nil)
	resp.SetStream(m.Stream())
	p.srv.SendResponse(ctx, resp)

	return nil
}

func (p *BlockProtocol) Run(ctx context.Context, protocolChan chan *msg.Message) {
	for {
		select {
		case m := <-protocolChan:
			log.Debug("received message")
			e := p.handleMessage(ctx, m)
			if e != nil {
				log.Error(e)
			}
		case <-ctx.Done():
			log.Error(ctx.Err())
			return
		case <-p.stop:
			log.Info("Block protocol stopped")
			return
		}
	}
}

func (p *BlockProtocol) Stop() {
	go func() {
		p.stop <- 0
	}()
}

func (p *BlockProtocol) handleMessage(ctx context.Context, m *msg.Message) error {
	defer func() {
		if r := recover(); r != nil {
			log.Error("panic occurred: ", r)
		}
	}()
	switch m.GetType() {
	case pb.Message_HELLO_REQUEST:
		return p.OnHello(ctx, m)
	case pb.Message_HEADERS_REQUEST:
		return p.OnHeaderRequest(ctx, m)
	case pb.Message_BLOCK_REQUEST:
		return p.OnBlockRequest(ctx, m)
	}
	return nil
}
