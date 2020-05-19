package blockchain

import (
	"github.com/gagarinchain/network/blockchain/state"
	"github.com/gagarinchain/network/common/eth/common"
	pb "github.com/gagarinchain/network/common/protobuff"
	"github.com/golang/protobuf/ptypes"
)

type RequestHandler struct {
	bc Blockchain
	db state.DB
}

func NewRequestHandler(bc Blockchain, db state.DB) *RequestHandler {
	return &RequestHandler{bc: bc, db: db}
}

func (d *RequestHandler) HandleAccountRequest(req *pb.Request) *pb.Event {
	payload := &pb.AccountRequestPayload{}
	if err := ptypes.UnmarshalAny(req.Payload, payload); err != nil {
		log.Error("can't parse", err)
		return &pb.Event{Id: req.Id, Type: pb.Event_ACCOUNT}
	}
	var hash common.Hash
	if payload.Block == nil || len(payload.Block) == 0 {
		hash = d.bc.GetTopCommittedBlock().Header().Hash()
	} else {
		hash = common.BytesToHash(payload.Block)
	}
	r, f := d.db.Get(hash)
	if !f {
		return &pb.Event{Id: req.Id, Type: pb.Event_ACCOUNT}
	}

	acc, found := r.Get(common.BytesToAddress(payload.Address))
	if !found {
		log.Error("not found")
		return &pb.Event{Id: req.Id, Type: pb.Event_ACCOUNT}
	}
	//proof := r.Proof(common2.BytesToAddress(payload.Address))
	any, e := ptypes.MarshalAny(&pb.AccountResponsePayload{
		Account: &pb.AccountE{
			Address: payload.Address,
			Block:   hash.Bytes(),
			Nonce:   acc.Nonce(),
			Value:   acc.Balance().Uint64(),
		},
	})
	if e != nil {
		log.Error("can't parse", e)
		return &pb.Event{Id: req.Id, Type: pb.Event_ACCOUNT}
	}
	return &pb.Event{
		Id:      req.Id,
		Type:    pb.Event_ACCOUNT,
		Payload: any,
	}
}

func (d *RequestHandler) HandleBlockRequest(req *pb.Request) *pb.Event {
	payload := &pb.BlockRequestPayload{}
	if err := ptypes.UnmarshalAny(req.Payload, payload); err != nil {
		log.Error("can't parse", err)
		return &pb.Event{Id: req.Id, Type: pb.Event_BLOCK}
	}
	block := d.bc.GetBlockByHash(common.BytesToHash(payload.Hash))
	if block == nil {
		return &pb.Event{Id: req.Id, Type: pb.Event_BLOCK}
	}
	pbblock := block.GetMessage()
	any, e := ptypes.MarshalAny(pbblock)
	if e != nil {
		log.Error("can't parse", e)
		return &pb.Event{Id: req.Id, Type: pb.Event_BLOCK}
	}
	return &pb.Event{
		Type:    pb.Event_BLOCK,
		Id:      req.Id,
		Payload: any,
	}
}
