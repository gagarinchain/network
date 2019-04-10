package blockchain

import (
	"context"
	"github.com/golang/protobuf/ptypes"
	"github.com/poslibp2p/common/eth/common"
	msg "github.com/poslibp2p/common/message"
	"github.com/poslibp2p/common/protobuff"
	"github.com/poslibp2p/network"
	"time"
)

//TODO Consider adding stream-peer cache, so we can reuse opened streams, don't forget that this scheme can produce bottlenecks
//Now we don't use same stream to send response, it means that we pass peer id to open new stream to this peer when sending response, this scheme is redundant too
type BlockProtocol struct {
	srv  network.Service
	bc   *Blockchain
	sync Synchronizer
}

var Version int32 = 1

func CreateBlockProtocol(srv network.Service, bc *Blockchain, sync Synchronizer) *BlockProtocol {
	return &BlockProtocol{srv: srv, bc: bc, sync: sync}
}

func (p *BlockProtocol) Bootstrap(ctx context.Context) {
	rq := &pb.Message{Type: pb.Message_HELLO_REQUEST}
	m := &msg.Message{Message: rq}
	mChan, err := p.srv.SendRequestToRandomPeer(ctx, m)
	var resp *msg.Message
	select {
	case b, ok := <-mChan:
		if !ok {
			log.Fatal("Error while requesting hello, channel is closed")
			return
		} else {
			resp = b
		}
	case err := <-err:
		log.Fatal("Error while requesting hello", err)
	}

	if resp.Type != pb.Message_HELLO_RESPONSE {
		log.Errorf("Not expected msg type %v response to Hello", resp.Type)
	}

	h := &pb.HelloPayload{}
	if err := ptypes.UnmarshalAny(resp.Payload, h); err != nil {
		log.Error("Couldn't unmarshal response", err)
	}
	//TODO check here different equivocations, such as very high block heights etc
	if h.GetVersion() != Version {
		log.Fatal("Wrong version")
		return
	}

	if h.GetTopBlockHeight() > p.bc.GetTopHeight() {
		if err := p.sync.RequestBlocks(ctx, p.bc.GetTopHeight(), h.GetTopBlockHeight(), nil); err != nil {
			log.Fatal("Error while loading blocks", err)
		}
	}
}

func (p *BlockProtocol) OnBlockRequest(ctx context.Context, req *msg.Message) {
	payload := req.GetPayload()
	br := &pb.BlockRequestPayload{}
	if err := ptypes.UnmarshalAny(payload, br); err != nil {
		log.Error("Can't unmarshal block request payload", err)
		return
	}

	var blocks []*Block
	if br.GetHash() != nil {
		hash := common.BytesToHash(br.GetHash())
		blocks = append(blocks, p.bc.GetBlockByHash(hash))
	}
	if br.GetHeight() != -1 {
		blocks = append(blocks, p.bc.GetBlockByHeight(br.GetHeight())...)
	}

	resp, e := createBlockResponse(blocks)
	if e != nil {
		log.Error("Can't create response", e)
		return
	}

	any, e := ptypes.MarshalAny(resp)
	if e != nil {
		log.Error("Can't create response", e)
		return
	}
	block := msg.CreateMessage(pb.Message_BLOCK_RESPONSE, any, nil)
	p.srv.SendMessage(ctx, req.Source(), block)

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

func (p *BlockProtocol) OnHello(ctx context.Context, m *msg.Message) {
	if m.GetType() != pb.Message_HELLO_REQUEST {
		log.Error("wrong message type, expected ", pb.Message_HELLO_REQUEST.String())
		return
	}

	payload := &pb.HelloPayload{}
	payload.Time = time.Now().UnixNano()
	payload.Version = Version
	payload.TopBlockHeight = p.bc.GetHead().Header().Height()

	any, e := ptypes.MarshalAny(payload)
	if e != nil {
		log.Error("Can't marshal block request payload", e)
		return
	}
	resp := msg.CreateMessage(pb.Message_HELLO_RESPONSE, any, nil)
	p.srv.SendMessage(ctx, m.Source(), resp)
}
