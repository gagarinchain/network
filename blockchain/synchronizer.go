package blockchain

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/poslibp2p/eth/common"
	"github.com/poslibp2p/message"
	"sort"
	"sync"
)

type Synchronizer interface {
	//Requesting blocks (low, high]
	RequestBlocks(ctx context.Context, low int32, high int32, peer *message.Peer) error
	RequestFork(ctx context.Context, hash common.Hash, peer *message.Peer) error
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

//TODO make kind of parallel batch loading here
func (s *SynchronizerImpl) RequestBlocks(ctx context.Context, low int32, high int32, peer *message.Peer) error {
	wg := &sync.WaitGroup{}
	wg.Add(int(high - low))

	type Exec struct {
		blocks []*Block
		lock   *sync.Mutex
	}

	exec := &Exec{
		blocks: nil,
		lock:   &sync.Mutex{},
	}

	for i := low + 1; i <= high; i++ {
		go func(group *sync.WaitGroup, ind int32, exec *Exec) {
			blocks, e := ReadBlocksWithErrors(s.bsrv.RequestBlocksAtHeight(ctx, ind, peer))
			if e != nil {
				log.Error("Can't read blocks", e)
			}

			for _, b := range blocks {
				exec.lock.Lock()
				exec.blocks = append(exec.blocks, b)
				exec.lock.Unlock()
			}

			group.Done()
		}(wg, i, exec)

	}
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return err
	}

	if len(exec.blocks) == 0 {
		return errors.New("No blocks received")
	}

	sort.Sort(ByHeight(exec.blocks))

	if exec.blocks[0].Height() != low+1 {
		return errors.New("Tail of fork is not expected")
	}
	if exec.blocks[len(exec.blocks)-1].Height() != high {
		return errors.New("Head of fork is not expected")
	}

	parent := exec.blocks[0]
	for i := 1; i < len(exec.blocks); i++ {
		block := exec.blocks[i]
		if block.Height()-parent.Height() > 1 {
			return errors.New("fork integrity violation for fork block " + string(i))
		}
	}

	return s.addBlocksTransactional(exec.blocks)
}

//Request all blocks starting at top committed block, which all replicas must have and all forks must include and ending with block with given hash
func (s *SynchronizerImpl) RequestFork(ctx context.Context, hash common.Hash, peer *message.Peer) error {
	resp, err := s.bsrv.RequestFork(ctx, s.bc.GetTopCommittedBlock().Header().Height(), hash, peer)
	blocks, e := ReadBlocksWithErrors(resp, err)
	if e != nil {
		return e
	}
	if len(blocks) == 0 {
		return errors.New("No blocks received")
	}
	sort.Sort(ByHeight(blocks))

	//Check fork integrity
	parent := s.bc.GetTopCommittedBlock()
	for i, b := range blocks {
		if b.Header().Parent() != parent.Header().Hash() || b.Header().Height()-parent.Header().Height() != 1 {
			return errors.New(fmt.Sprintf("fork integrity violation for fork block [%v] %d in fork", b.Header().Hash().Hex(), i))
		}
		parent = b
	}

	//Check blockchain head is what we requested
	log.Debug(blocks)
	if blocks[len(blocks)-1].Header().Hash() != hash {
		return errors.New("Head of fork is not expected")
	}

	return s.addBlocksTransactional(blocks)
}

func (s *SynchronizerImpl) addBlocksTransactional(blocks []*Block) error {
	errorIndex := -1
	var err error
	for i, b := range blocks {
		if err = s.bc.AddBlock(b); err != nil {
			errorIndex = i
			log.Infof("Error adding block [%v]", b.Header().Hash().Hex())
			break
		}
	}
	//compensation logic, should clean up previous additions
	if errorIndex > -1 {
		toCleanUp := blocks[:errorIndex+1]
		sort.Sort(sort.Reverse(ByHeight(toCleanUp)))
		for _, b := range toCleanUp {
			if err := s.bc.RemoveBlock(b); err != nil {
				log.Fatal(err)
				panic("Can't delete added previously block, reload blockchain")
			}
		}
		return err
	}
	return nil
}
