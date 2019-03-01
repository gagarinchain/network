package blockchain

import (
	"github.com/golang/protobuf/ptypes"
	"github.com/poslibp2p/eth/common"
	"github.com/poslibp2p/message"
	"github.com/poslibp2p/message/protobuff"
	"github.com/poslibp2p/network"
)

type Synchronizer interface {
	RequestBlock(hash common.Hash)
	Bootstrap()
	RequestBlockWithDeps(header *Header)
}


type SynchronizerImpl struct {
	bchan <-chan *Block
	me *network.Peer
	srv network.Service
	bc *Blockchain
}

//TODO: IMPORTANT think whether we MUST wait until we receive absent blocks to go on processing
func (s *SynchronizerImpl) RequestBlockWithDeps(header *Header) {
	if header != nil && !s.bc.Contains(header.hash) {
		s.RequestBlockSynchronously(header.hash)
	}

	if header.parent!= nil && !s.bc.Contains(header.parent.hash) {
		s.RequestBlockSynchronously(header.parent.hash)
	}

	//TODO think about first two conditions, they probably never should be true, because every block we receive contains qc and must be validated
	//Wrong, we can pass here qref block header, which won't contain QC itself
	if header.qc != nil && header.qc.qrefBlock != nil && !s.bc.Contains(header.qc.qrefBlock.hash) {
		s.RequestBlockSynchronously(header.qc.qrefBlock.hash)
	}

}

finish me
func (s *SynchronizerImpl) RequestBlockSynchronously(hash common.Hash) {
	payload := &pb.BlockRequestPayload{Hash:hash.Bytes()}
	any, e := ptypes.MarshalAny(payload)
	if e != nil {
		log.Error("Can't assemble message", e)
	}

	msg := message.CreateMessage(pb.Message_BLOCK_REQUEST, s.me.GetPrivateKey(), any)
	resp := s.srv.SendMessageToRandomPeerAndGetResponse(msg)
	if resp.Type != pb.Message_BLOCK_RESPONSE {
		log.Errorf("Received message of type %v, but expected %v", resp.Type.String(), pb.Message_BLOCK_RESPONSE.String())
	}

	rp := &pb.BlockResponsePayload{}
	if err := ptypes.UnmarshalAny(resp.Payload, rp); err != nil {
		log.Error("Couldn't unmarshal response")
	}

	rp.GetBlock()
}

func (s *SynchronizerImpl) Bootstrap() {
	go s.sync()
}

func (s *SynchronizerImpl) sync() {
	for {
		block := <- s.bchan
		e := s.bc.AddBlock(block)
		if e != nil {
			log.Error("Error while adding block", e)
		}
	}
}