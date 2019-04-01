package blockchain

import (
	"github.com/golang/protobuf/ptypes"
	"github.com/poslibp2p/eth/common"
	msg "github.com/poslibp2p/message"
	"github.com/poslibp2p/message/protobuff"
	"github.com/poslibp2p/network"
	"time"
)

type BlockProtocol struct {
	srv  network.Service
	bc   *Blockchain
	sync Synchronizer
}

var Version int32 = 1

func CreateBlockProtocol(srv network.Service, bc *Blockchain, sync Synchronizer) *BlockProtocol {
	return &BlockProtocol{srv: srv, bc: bc, sync: sync}
}

func (p *BlockProtocol) Bootstrap() {
	rq := &pb.Message{Type: pb.Message_HELLO_REQUEST}
	m := &msg.Message{Message: rq}
	resp := <-p.srv.SendRequestToRandomPeer(m)

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
		p.sync.RequestBlocks(p.bc.GetTopHeight(), h.GetTopBlockHeight())
	}
}

func (p *BlockProtocol) OnBlockRequest(req *msg.Message) {
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
	p.srv.SendMessage(req.Source(), msg.CreateMessage(pb.Message_BLOCK_RESPONSE, any, nil))

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

func (p *BlockProtocol) OnHello(m *msg.Message) {
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
	p.srv.SendMessage(m.Source(), resp)
}
