package blockchain

import (
	"github.com/poslibp2p/message"
	"sync"
)

type Synchronizer interface {
	Bootstrap()
	//RequestBlockWithParent(header *Header)

	//Requesting blocks (low, high]
	RequestBlocks(low int32, high int32)
}

type SynchronizerImpl struct {
	bchan <-chan *Block
	me    *message.Peer
	bsrv  BlockService
	bc    *Blockchain
}

func CreateSynchronizer(bchan <-chan *Block, me *message.Peer, bsrv BlockService, bc *Blockchain) Synchronizer {
	return &SynchronizerImpl{
		bchan: bchan,
		me:    me,
		bsrv:  bsrv,
		bc:    bc,
	}
}

//IMPORTANT: think whether we MUST wait until we receive absent blocks to go on processing
//I think we must, if we have unknown block in the 3-chain we can't push protocol forward
//func (s *SynchronizerImpl) RequestBlockWithParent(header *Header) {
//	var headChan <-chan *Block
//	var parentChan <-chan *Block
//
//	if header != nil && !s.bc.Contains(header.hash) {
//		headChan = s.RequestBlock(header.hash)
//	}
//
//	if !s.bc.Contains(header.parent) {
//		parentChan = s.RequestBlock(header.parent)
//	}
//
//	for headChan != nil || parentChan != nil {
//		select {
//		case headBlock, ok := <-headChan:
//			if !ok {
//				headChan = nil
//			} else if e := s.bc.AddBlock(headBlock); e != nil {
//				log.Error(e)
//			}
//		case parentBlock, ok := <-parentChan:
//			if !ok {
//				parentChan = nil
//			} else if e := s.bc.AddBlock(parentBlock); e != nil {
//				log.Error(e)
//			}
//		}
//	}
//
//}

func (s *SynchronizerImpl) Bootstrap() {
	go s.sync()
}

//TODO make kind of parallel batch loading here
func (s *SynchronizerImpl) RequestBlocks(low int32, high int32) {
	wg := &sync.WaitGroup{}
	wg.Add(int(high - low))

	for i := low + 1; i <= high; i++ {
		go func(group *sync.WaitGroup, ind int32) {
			for b := range s.bsrv.RequestBlocksAtHeight(ind) {
				if err := s.bc.AddBlock(b); err != nil {
					log.Warningf("Error adding block [%v]", b.Header().Hash().Hex())
				}
			}

			group.Done()
		}(wg, i)

	}
	wg.Wait()
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
