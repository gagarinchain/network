package blockchain

import (
	"bytes"
	"context"
	"errors"
	com "github.com/gagarinchain/common"
	"github.com/gagarinchain/common/api"
	"github.com/gagarinchain/common/eth/common"
	"github.com/gagarinchain/common/message"
	"github.com/gagarinchain/common/protobuff"
	"github.com/gagarinchain/network/network"
	"github.com/golang/protobuf/ptypes"
)

var (
	NotFound          = errors.New("not found")
	MalformedResponse = errors.New("malformed response")
	NotValid          = errors.New("not valid")
)

type BlockService interface {
	RequestHeaders(ctx context.Context, low int32, high int32, peer *com.Peer) (resp []api.Header, e error)
	RequestBlock(ctx context.Context, hash common.Hash, peer *com.Peer) (resp api.Block, e error)
}

type BlockServiceImpl struct {
	srv             network.Service
	validator       api.Validator
	headerValidator api.Validator
	threshold       int
}

func NewBlockService(srv network.Service, validator api.Validator, headerValidator api.Validator) *BlockServiceImpl {
	return &BlockServiceImpl{srv: srv, validator: validator, headerValidator: headerValidator}
}

// return headers of blocks for boundaries (low, high]
func (s *BlockServiceImpl) RequestHeaders(ctx context.Context, low int32, high int32, peer *com.Peer) (resp []api.Header, e error) {
	headers, e := s.requestHeaders(ctx, low, high, peer)

	if e == nil {
		return headers, nil
	} else {
		log.Error("Error loading headers", e)
	}

	for i := 1; i < s.threshold; i++ {
		headers, e := s.requestHeaders(ctx, low, high, nil)

		if e == nil {
			return headers, nil
		} else {
			log.Error("Error loading headers", e)
		}

	}

	log.Error("No headers found for ")
	return nil, NotFound
}

func (s *BlockServiceImpl) RequestBlock(ctx context.Context, hash common.Hash, peer *com.Peer) (resp api.Block, e error) {
	block, e := s.requestBlock(ctx, hash, peer)
	if e == nil {
		return block, nil
	}

	for i := 1; i < s.threshold; i++ {
		block, e := s.requestBlock(ctx, hash, nil)
		//todo handle errors with ban score
		if e == nil && block != nil && bytes.Equal(hash.Bytes(), block.Header().Hash().Bytes()) {
			return block, nil
		}
	}

	log.Error("Block with hash %v can't be found", hash.Hex())
	return nil, NotFound
}

// Synchronously query peers for block with exact hash threshold times until we get block or reach the threshold
// We know nothing about block structures and make simple block validations
//TODO add ban scores for specific error handling
func (s *BlockServiceImpl) requestBlock(ctx context.Context, hash common.Hash, peer *com.Peer) (resp api.Block, e error) {
	payload := &pb.BlockRequestPayload{Hash: hash.Bytes()}

	any, e := ptypes.MarshalAny(payload)
	if e != nil {
		return nil, e
	}

	msg := message.CreateMessage(pb.Message_BLOCK_REQUEST, any, nil)

	var m *message.Message
	if peer == nil {
		resps, errs := s.srv.SendRequestToRandomPeer(ctx, msg)
		select {
		case m = <-resps:
		case e := <-errs:
			return nil, e
		}
	} else {
		resps, errs := s.srv.SendRequest(ctx, peer, msg)
		select {
		case m = <-resps:
		case e := <-errs:
			return nil, e
		}
	}

	if m.Type != pb.Message_BLOCK_RESPONSE {
		log.Errorf("received message of type %v, but expected %v", m.Type.String(),
			pb.Message_BLOCK_RESPONSE.String())
		return nil, MalformedResponse
	}

	rp := &pb.BlockResponsePayload{}
	if e := ptypes.UnmarshalAny(m.Payload, rp); e != nil {
		log.Error(e)
		return nil, MalformedResponse
	}

	if rp.GetErrorCode() != nil {
		switch rp.GetErrorCode().Code {
		case pb.Error_NOT_FOUND:
			return nil, NotFound
		}
	}

	if pbBlock := rp.GetBlock(); pbBlock != nil {
		resp = CreateBlockFromMessage(pbBlock)

		log.Infof("Received new block with hash %v", resp.Header().Hash().Hex())
		if !s.validator.Supported(pb.Message_BLOCK_RESPONSE) {
			panic("bad block validator")
		}
		isValid, e := s.validator.IsValid(resp)
		if e != nil {
			log.Errorf("Block %v is not  valid, %v", resp.Header().Hash().Hex(), e)
			return nil, NotValid
		}
		if !isValid {
			log.Errorf("Block %v is not  valid", resp.Header().Hash().Hex())
			return nil, NotValid
		}
	}
	return resp, nil
}

func (s *BlockServiceImpl) requestHeaders(ctx context.Context, low int32, high int32, peer *com.Peer) (resp []api.Header, e error) {
	payload := &pb.HeadersRequest{Low: low, High: high}

	any, e := ptypes.MarshalAny(payload)
	if e != nil {
		return nil, e
	}

	msg := message.CreateMessage(pb.Message_HEADERS_REQUEST, any, nil)
	var m *message.Message
	if peer == nil {
		resps, errs := s.srv.SendRequestToRandomPeer(ctx, msg)
		select {
		case m = <-resps:
		case e := <-errs:
			return nil, e
		}
	} else {
		resps, errs := s.srv.SendRequest(ctx, peer, msg)
		select {
		case m = <-resps:
		case e := <-errs:
			return nil, e
		}
	}

	if m.Type != pb.Message_HEADERS_RESPONSE {
		log.Errorf("received message of type %v, but expected %v", m.Type.String(),
			pb.Message_HEADERS_RESPONSE.String())
		return nil, MalformedResponse
	}

	rp := &pb.HeadersResponse{}
	if e := ptypes.UnmarshalAny(m.Payload, rp); e != nil {
		log.Error(e)
		return nil, MalformedResponse
	}

	if rp.GetErrorCode() != nil || rp.GetHeaders() == nil || len(rp.GetHeaders().Headers) == 0 {
		switch rp.GetErrorCode().Code {
		case pb.Error_NOT_FOUND:
			return nil, NotFound
		}
	}

	for _, headerM := range rp.GetHeaders().Headers {
		header := CreateBlockHeaderFromMessage(headerM)

		log.Infof("Received new header with hash %v", header.Hash().Hex())
		if !s.headerValidator.Supported(pb.Message_HEADERS_RESPONSE) {
			panic("bad header validator")
		}
		isValid, e := s.headerValidator.IsValid(header)
		if e != nil {
			log.Errorf("Header %v is not  valid, %v", header.Hash().Hex(), e)
			return nil, NotValid
		}
		if !isValid {
			log.Errorf("Block %v is not  valid", header.Hash().Hex())
			return nil, NotValid
		}

		resp = append(resp, header)
	}
	return resp, nil
}
