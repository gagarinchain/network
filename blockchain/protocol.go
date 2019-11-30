package blockchain

import (
	"context"
	"errors"
	"fmt"
	"github.com/gagarinchain/network/common/eth/common"
	msg "github.com/gagarinchain/network/common/message"
	"github.com/gagarinchain/network/common/protobuff"
	"github.com/gagarinchain/network/network"
	"github.com/golang/protobuf/ptypes"
	"time"
)

//TODO Consider adding stream-peer cache, so we can reuse opened streams, don't forget that this scheme can produce bottlenecks
//Now we don't use same stream to send response, it means that we pass peer id to open new stream to this peer when sending response, this scheme is redundant too
type BlockProtocol struct {
	srv  network.Service
	bc   *Blockchain
	sync Synchronizer
	stop chan int
}

var Version int32 = 1

// We won't allow batches bigger than this (the server-side limit).
const RequestBlockHeaderBatchSizeLimit = 100

func CreateBlockProtocol(srv network.Service, bc *Blockchain, sync Synchronizer) *BlockProtocol {
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
		if err := p.sync.RequestBlocks(ctx, p.bc.GetTopHeight(), h.GetTopBlockHeight(), resp.Source()); err != nil {
			return err
		}
	}

	return nil
}

func (p *BlockProtocol) OnBlockRequest(ctx context.Context, req *msg.Message) error {
	payload := req.GetPayload()
	br := &pb.BlockRequestPayload{}
	if err := ptypes.UnmarshalAny(payload, br); err != nil {
		return err
	}

	var blocks []*Block
	if br.GetHeight() != -1 && br.GetHash() != nil {
		blocks = p.bc.GetFork(br.GetHeight(), common.BytesToHash(br.GetHash()))
	} else if br.GetHash() != nil { //requesting exact
		hash := common.BytesToHash(br.GetHash())
		blocks = append(blocks, p.bc.GetBlockByHash(hash))
	} else if br.GetHeight() != -1 { //requesting by height
		blocks = append(blocks, p.bc.GetBlockByHeight(br.GetHeight())...)
	}

	resp, e := createBlockResponse(blocks)
	if e != nil {
		return e
	}

	any, e := ptypes.MarshalAny(resp)
	if e != nil {
		return e
	}
	block := msg.CreateMessage(pb.Message_BLOCK_RESPONSE, any, nil)
	block.SetStream(req.Stream())
	p.srv.SendResponse(ctx, block)
	return nil
}

func createBlockResponse(blocks []*Block) (*pb.BlockResponsePayload, error) {
	var bs []*pb.Block
	if len(blocks) == 0 {
		e := &pb.Error{Code: pb.Error_NOT_FOUND, Desc: "Not found"}
		return &pb.BlockResponsePayload{Response: &pb.BlockResponsePayload_ErrorCode{ErrorCode: e}}, nil
	}

	for _, b := range blocks {
		bs = append(bs, b.GetMessage())
	}
	i := &pb.Blocks{Blocks: bs}
	p := &pb.BlockResponsePayload_Blocks{Blocks: i}
	return &pb.BlockResponsePayload{Response: p}, nil
}

func (p *BlockProtocol) OnBlockHeaderBatchRequest(ctx context.Context, req *msg.Message) error {
	payload := req.GetPayload()
	reqPayload := &pb.BlockHeaderBatchRequestPayload{}
	if err := ptypes.UnmarshalAny(payload, reqPayload); err != nil {
		return err
	}

	low := reqPayload.Low
	high := reqPayload.High
	batchSize := low - high

	if batchSize < 1 {
		return fmt.Errorf("invalid low..high batch range: (%d, %d]", low, high)
	} else if batchSize > RequestBlockHeaderBatchSizeLimit {
		return fmt.Errorf("invalid low..high batch range: (%d, %d], "+
			"maximum batch size is %d", low, high, RequestBlockHeaderBatchSizeLimit)
	}

	headers := make([]*pb.BlockHeader, 0, batchSize) // Preallocate a slice of batchSize size.

	for h := low + 1; h <= high; h++ {
		blocks := p.bc.GetBlockByHeight(h)
		for _, block := range blocks {
			headers = append(headers, block.header.GetMessage())
		}
	}

	resPayload := &pb.BlockHeaderBatchResponsePayload{Headers: headers}

	any, e := ptypes.MarshalAny(resPayload)
	if e != nil {
		return e
	}

	resMessage := msg.CreateMessage(pb.Message_BLOCK_HEADER_BATCH_RESPONSE, any, nil)
	resMessage.SetStream(req.Stream())
	p.srv.SendResponse(ctx, resMessage)

	return nil
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
	case pb.Message_BLOCK_REQUEST:
		return p.OnBlockRequest(ctx, m)
	case pb.Message_BLOCK_HEADER_BATCH_REQUEST:
		return p.OnBlockHeaderBatchRequest(ctx, m)
	}
	return nil
}
