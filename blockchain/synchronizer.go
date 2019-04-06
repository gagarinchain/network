package blockchain

import (
	"context"
	"github.com/poslibp2p/eth/common"
	"github.com/poslibp2p/message"
	"sync"
)

type Synchronizer interface {
	//RequestBlockWithParent(header *Header)

	//Requesting blocks (low, high]
	RequestBlocks(ctx context.Context, low int32, high int32)
	RequestFork(ctx context.Context, hash common.Hash) error
}

type SynchronizerImpl struct {
	me   *message.Peer
	bsrv BlockService
	bc   *Blockchain
}

func CreateSynchronizer(me *message.Peer, bsrv BlockService, bc *Blockchain) Synchronizer {
	return &SynchronizerImpl{
		me:   me,
		bsrv: bsrv,
		bc:   bc,
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

//TODO make kind of parallel batch loading here
func (s *SynchronizerImpl) RequestBlocks(ctx context.Context, low int32, high int32) {
	wg := &sync.WaitGroup{}
	wg.Add(int(high - low))

	for i := low + 1; i <= high; i++ {
		go func(group *sync.WaitGroup, ind int32) {
			for b := range s.bsrv.RequestBlocksAtHeight(ctx, ind) {
				if err := s.bc.AddBlock(b); err != nil {
					log.Warningf("Error adding block [%v]", b.Header().Hash().Hex())
				}
			}

			group.Done()
		}(wg, i)

	}
	wg.Wait()
}

//request all unknown blocks sequentially from top height head descending
func (s *SynchronizerImpl) RequestFork(ctx context.Context, head common.Hash) error {

	return nil
}
