package blockchain

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/golang/protobuf/ptypes"
	"github.com/poslibp2p/eth/common"
	"github.com/poslibp2p/message"
	"github.com/poslibp2p/message/protobuff"
	"github.com/poslibp2p/network"
)

type Synchronizer interface {
	RequestBlock(hash common.Hash, respChan chan<- *Block)
	Bootstrap()
	RequestBlockWithParent(header *Header)
}

type SynchronizerImpl struct {
	bchan <-chan *Block
	me    *network.Peer
	srv   network.Service
	bc    *Blockchain
}

func CreateSynchronizer(bchan <-chan *Block, me *network.Peer, srv network.Service, bc *Blockchain) Synchronizer {
	return &SynchronizerImpl{bchan: bchan, me: me, srv: srv, bc: bc}
}

//IMPORTANT: think whether we MUST wait until we receive absent blocks to go on processing
//I think we must, if we have unknown block in the 3-chain we can't push protocol forward
func (s *SynchronizerImpl) RequestBlockWithParent(header *Header) {
	var headChan chan *Block
	var parentChan chan *Block

	if header != nil && !s.bc.Contains(header.hash) {
		headChan = make(chan *Block)
		go s.RequestBlock(header.hash, headChan)
	}

	if !s.bc.Contains(header.parent) {
		parentChan = make(chan *Block)
		go s.RequestBlock(header.parent, parentChan)
	}

	for headChan != nil || parentChan != nil {
		select {
		case headBlock, ok := <-headChan:
			if !ok {
				headChan = nil
			} else if e := s.bc.AddBlock(headBlock); e != nil {
				log.Error(e)
			}
		case parentBlock, ok := <-parentChan:
			if !ok {
				parentChan = nil
			} else if e := s.bc.AddBlock(parentBlock); e != nil {
				log.Error(e)
			}
		}
	}

}

func (s *SynchronizerImpl) RequestBlock(hash common.Hash, respChan chan<- *Block) {
	payload := &pb.BlockRequestPayload{Hash: hash.Bytes()}
	any, e := ptypes.MarshalAny(payload)
	if e != nil {
		log.Error("Can't assemble message", e)
	}

	msg := message.CreateMessage(pb.Message_BLOCK_REQUEST, s.me.GetPrivateKey(), any)

	resp := s.srv.SendMessageToRandomPeer(msg)

	if resp.Type != pb.Message_BLOCK_RESPONSE {
		log.Errorf("Received message of type %v, but expected %v", resp.Type.String(), pb.Message_BLOCK_RESPONSE.String())
	}

	rp := &pb.Block{}
	if err := ptypes.UnmarshalAny(resp.Payload, rp); err != nil {
		log.Error("Couldn't unmarshal response", err)
	}

	block := CreateBlockFromMessage(rp)

	log.Info("Received new block")
	spew.Dump(block)
	//TODO validate block
	respChan <- block
	close(respChan)
}

func (s *SynchronizerImpl) Bootstrap() {
	go s.sync()
}

func (s *SynchronizerImpl) sync() {
	for {
		block := <-s.bchan
		e := s.bc.AddBlock(block)
		if e != nil {
			log.Error("Error while adding block", e)
		}
	}
}
