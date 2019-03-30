package blockchain

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/golang/protobuf/ptypes"
	"github.com/poslibp2p/eth/common"
	"github.com/poslibp2p/message"
	"github.com/poslibp2p/message/protobuff"
	"github.com/poslibp2p/network"
)

type BlockService interface {
	RequestBlock(hash common.Hash) <-chan *Block
	RequestBlocksAtHeight(height int32) <-chan *Block
}

type BlockServiceImpl struct {
	srv network.Service
}

func NewBlockService(srv network.Service) *BlockServiceImpl {
	return &BlockServiceImpl{srv: srv}
}

func (s *BlockServiceImpl) RequestBlock(hash common.Hash) <-chan *Block {
	return s.requestBlockUgly(hash, -1)
}

func (s *BlockServiceImpl) RequestBlocksAtHeight(height int32) <-chan *Block {
	return s.requestBlockUgly(common.Hash{}, height)
}

//we can pass rather hash or block level here. if we want to omit height parameter must pass -1
func (s *BlockServiceImpl) requestBlockUgly(hash common.Hash, height int32) <-chan *Block {
	var payload *pb.BlockRequestPayload

	if height < 0 {
		payload = &pb.BlockRequestPayload{Hash: hash.Bytes()}
	} else {
		payload = &pb.BlockRequestPayload{Height: height}
	}

	any, e := ptypes.MarshalAny(payload)
	if e != nil {
		log.Error("Can't assemble message", e)
	}

	msg := message.CreateMessage(pb.Message_BLOCK_REQUEST, any)

	resultChan := make(chan *Block)
	go func() {
		resp := <-s.srv.SendRequestToRandomPeer(msg)
		if resp.Type != pb.Message_BLOCK_RESPONSE {
			log.Errorf("Received message of type %v, but expected %v", resp.Type.String(), pb.Message_BLOCK_RESPONSE.String())
			close(resultChan)
			return
		}

		rp := &pb.BlockResponsePayload{}
		if err := ptypes.UnmarshalAny(resp.Payload, rp); err != nil {
			log.Error("Couldn't unmarshal response", err)
			close(resultChan)
			return
		}

		for _, blockM := range rp.GetBlocks().GetBlocks() {
			block := CreateBlockFromMessage(blockM)

			log.Info("Received new block")
			spew.Dump(block)
			//TODO validate block
			resultChan <- block
		}

		close(resultChan)
	}()
	return resultChan
}
