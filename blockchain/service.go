package blockchain

import (
	"context"
	"fmt"
	gagarinchain "github.com/gagarinchain/network"
	com "github.com/gagarinchain/network/common"
	"github.com/gagarinchain/network/common/eth/common"
	"github.com/gagarinchain/network/common/message"
	"github.com/gagarinchain/network/common/protobuff"
	"github.com/gagarinchain/network/network"
	"github.com/golang/protobuf/ptypes"
	"github.com/pkg/errors"
)

type BlockService interface {
	// Requests blocks at with a specific hash from a specific peer.
	// If peer == nil, will request the same from a random peer.
	RequestBlock(ctx context.Context, hash common.Hash, peer *com.Peer) (resp chan *Block, err chan error)
	// Requests blocks at a specific height from a specific peer.
	// If peer == nil, will request the same from a random peer.
	RequestBlocksAtHeight(ctx context.Context, height int32, peer *com.Peer) (resp chan *Block, err chan error)
	// Requests block headers at (low, high] height, all in one batch, from a specific peer.
	// If peer == nil, will request the same from a random peer.
	RequestBlockHeaderBatch(ctx context.Context, low int32, high int32, peer *com.Peer) (resp chan *Header, err chan error)
	RequestFork(ctx context.Context, lowHeight int32, hash common.Hash, peer *com.Peer) (resp chan *Block, err chan error)
}

type BlockServiceImpl struct {
	srv       network.Service
	validator gagarinchain.Validator
}

func NewBlockService(srv network.Service, validator gagarinchain.Validator) *BlockServiceImpl {
	return &BlockServiceImpl{srv: srv, validator: validator}
}

func (s *BlockServiceImpl) RequestBlock(ctx context.Context, hash common.Hash, peer *com.Peer) (resp chan *Block, err chan error) {
	return s.requestBlock(ctx, hash, -1, peer)
}

func (s *BlockServiceImpl) RequestBlocksAtHeight(ctx context.Context, height int32, peer *com.Peer) (resp chan *Block, err chan error) {
	return s.requestBlock(ctx, common.Hash{}, height, peer)
}

//we can pass rather hash or block level here. if we want to omit height parameter must pass -1.
//we can pass height and header hash, it will mean that we want to get fork starting from hash block up to block at parameter height excluded
func (s *BlockServiceImpl) requestBlock(ctx context.Context, hash common.Hash, height int32, peer *com.Peer) (resp chan *Block, err chan error) {
	var payload *pb.BlockRequestPayload

	if height < 0 {
		payload = &pb.BlockRequestPayload{Hash: hash.Bytes(), Height: com.DefaultIntValue}
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
			if !s.validator.Supported(pb.Message_BLOCK_RESPONSE) {
				panic("bad block validator")
			}
			isValid, e := s.validator.IsValid(block)
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
	return s.requestBlock(ctx, hash, lowHeight, peer)
}

func (s *BlockServiceImpl) RequestBlockHeaderBatch(ctx context.Context, low int32, high int32, peer *com.Peer) (resp chan *Header, err chan error) {
	if low >= high {
		panic(fmt.Errorf("invalid low..high range: (%d, %d]", low, high))
	}

	err = make(chan error)
	resp = make(chan *Header)
	payload := &pb.BlockHeaderBatchRequestPayload{Low: low, High: high}
	any, e := ptypes.MarshalAny(payload)

	if e != nil {
		err <- e
		return resp, err
	}

	msg := message.CreateMessage(pb.Message_BLOCK_HEADER_BATCH_REQUEST, any, nil)

	var m *message.Message
	go func() {
		rc, ec := s.srv.SendRequest(ctx, peer, msg)
		select {
		case m = <-rc:
		case e := <-ec:
			err <- e
			return
		}

		if m.Type != pb.Message_BLOCK_HEADER_BATCH_RESPONSE {
			err <- errors.New(fmt.Sprintf("Received message of type %v, but expected %v",
				m.Type.String(), pb.Message_BLOCK_HEADER_BATCH_RESPONSE.String()))
			return
		}

		rp := &pb.BlockHeaderBatchResponsePayload{}
		if e := ptypes.UnmarshalAny(m.Payload, rp); e != nil {
			err <- e
			return
		}

		for _, headerM := range rp.Headers {
			header := CreateBlockHeaderFromMessage(headerM)

			log.Infof("Received new block header batch: (%v, %v]", low, high)

			// TODO: Validate headers when received. [!!!]

			resp <- header
		}

		close(resp)
	}()

	return resp, err
}

func ReadBlocksWithErrors(blockChan chan *Block, errChan chan error) (blocks []*Block, err error) {
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

	return blocks, nil
}
