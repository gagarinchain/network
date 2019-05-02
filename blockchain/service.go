package blockchain

import (
	"context"
	"fmt"
	"github.com/golang/protobuf/ptypes"
	"github.com/pkg/errors"
	com "github.com/poslibp2p/common"
	"github.com/poslibp2p/common/eth/common"
	"github.com/poslibp2p/common/message"
	"github.com/poslibp2p/common/protobuff"
	"github.com/poslibp2p/network"
)

type BlockService interface {
	RequestBlock(ctx context.Context, hash common.Hash, peer *com.Peer) (resp chan *Block, err chan error)
	RequestBlocksAtHeight(ctx context.Context, height int32, peer *com.Peer) (resp chan *Block, err chan error)
	RequestFork(ctx context.Context, lowHeight int32, hash common.Hash, peer *com.Peer) (resp chan *Block, err chan error)
}

type BlockServiceImpl struct {
	srv network.Service
}

func NewBlockService(srv network.Service) *BlockServiceImpl {
	return &BlockServiceImpl{srv: srv}
}

func (s *BlockServiceImpl) RequestBlock(ctx context.Context, hash common.Hash, peer *com.Peer) (resp chan *Block, err chan error) {
	return s.requestBlockUgly(ctx, hash, -1, nil)
}

func (s *BlockServiceImpl) RequestBlocksAtHeight(ctx context.Context, height int32, peer *com.Peer) (resp chan *Block, err chan error) {
	return s.requestBlockUgly(ctx, common.Hash{}, height, nil)
}

//we can pass rather hash or block level here. if we want to omit height parameter must pass -1.
//we can pass height and header hash, it will mean that we want to get fork starting from hash block up to block at parameter height excluded
func (s *BlockServiceImpl) requestBlockUgly(ctx context.Context, hash common.Hash, height int32, peer *com.Peer) (resp chan *Block, err chan error) {
	var payload *pb.BlockRequestPayload

	if height < 0 {
		payload = &pb.BlockRequestPayload{Hash: hash.Bytes(), Height: DefaultIntValue}
	} else if len(hash.Bytes()) == 0 {
		payload = &pb.BlockRequestPayload{Height: height}
	} else {
		payload = &pb.BlockRequestPayload{Height: height, Hash: hash.Bytes()}
	}

	any, e := ptypes.MarshalAny(payload)
	if e != nil {
		log.Error("Can't assemble message", e)
	}

	msg := message.CreateMessage(pb.Message_BLOCK_REQUEST, any, nil)

	resp = make(chan *Block)
	err = make(chan error)
	var m *message.Message
	go func() {
		if peer == nil {
			resps, errs := s.srv.SendRequestToRandomPeer(ctx, msg)
			select {
			case m = <-resps:
			case e := <-errs:
				err <- e
				close(resp)
				return
			}
		} else {
			resps, errs := s.srv.SendRequest(ctx, peer, msg)
			select {
			case m = <-resps:
			case e := <-errs:
				err <- e
				close(resp)
				return
			}
		}

		if m.Type != pb.Message_BLOCK_RESPONSE {
			err <- errors.New(fmt.Sprintf("Received message of type %v, but expected %v", m.Type.String(), pb.Message_BLOCK_RESPONSE.String()))
			close(resp)
			return
		}

		rp := &pb.BlockResponsePayload{}
		if e := ptypes.UnmarshalAny(m.Payload, rp); e != nil {
			err <- e
			close(resp)
			return
		}

		for _, blockM := range rp.GetBlocks().GetBlocks() {
			block := CreateBlockFromMessage(blockM)

			log.Infof("Received new block with hash %v", block.Header().Hash().Hex())
			isValid, e := IsValid(block)
			if e != nil {
				log.Errorf("Block %v is not  valid, %v", block.Header().Hash().Hex(), e)
				continue
			}
			if !isValid {
				log.Errorf("Block %v is not  valid", block.Header().Hash().Hex())
				continue
			}
			resp <- block
		}

		close(resp)
	}()
	return resp, err
}

func (s *BlockServiceImpl) RequestFork(ctx context.Context, lowHeight int32, hash common.Hash, peer *com.Peer) (resp chan *Block, err chan error) {
	return s.requestBlockUgly(ctx, hash, lowHeight, peer)
}

func ReadBlocksWithErrors(blockChan chan *Block, errChan chan error) (blocks []*Block, err error) {
	for blockChan != nil {
		for blockChan != nil {
			select {
			case b, ok := <-blockChan:
				if !ok {
					blockChan = nil
				} else {
					blocks = append(blocks, b)
				}
			case err := <-errChan:
				return nil, err
			}
		}
	}

	return blocks, nil
}
