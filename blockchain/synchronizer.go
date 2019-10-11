package blockchain

import (
	"context"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	comm "github.com/gagarinchain/network/common"
	"github.com/gagarinchain/network/common/eth/common"
	"github.com/pkg/errors"
	"sort"
	"sync"
)

type Synchronizer interface {
	//Requesting blocks (low, high]
	RequestBlocks(ctx context.Context, low int32, high int32, peer *comm.Peer) error
	RequestFork(ctx context.Context, hash common.Hash, peer *comm.Peer) error
}

type SynchronizerImpl struct {
	me   *comm.Peer
	bsrv BlockService
	bc   *Blockchain
}

func CreateSynchronizer(me *comm.Peer, bsrv BlockService, bc *Blockchain) Synchronizer {
	return &SynchronizerImpl{
		me:   me,
		bsrv: bsrv,
		bc:   bc,
	}
}

//TODO make kind of parallel batch loading here
func (s *SynchronizerImpl) RequestBlocks(ctx context.Context, low int32, high int32, peer *comm.Peer) error {
	wg := &sync.WaitGroup{}
	wg.Add(int(high - low))
	log.Info("Requesting %v blocks", int(high-low))

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
	log.Info("Received %v blocks", len(exec.blocks))
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

//Request all blocks starting at top committed block (not included), which all replicas must have (not sure that all, but 2 *f + 1)
//and all forks must include and ending with block with given hash
func (s *SynchronizerImpl) RequestFork(ctx context.Context, hash common.Hash, peer *comm.Peer) error {
	//we never request genesis block here, cause it will break block validity
	requestedHeight := max(s.bc.GetTopCommittedBlock().Header().Height()+1, 1)
	resp, err := s.bsrv.RequestFork(ctx, requestedHeight, hash, peer)
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
			for _, v := range blocks {
				log.Debugf(spew.Sdump(v.Header()))
			}

			log.Debugf("Blocks parent %v, parent hash %v", b.Header().Parent().Hex(), parent.header.hash.Hex())
			log.Debugf("Height difference %d", b.Header().Height()-parent.Header().Height())
			return fmt.Errorf("fork integrity violation for fork block [%v] %d in fork", b.Header().Hash().Hex(), i)
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

func max(a int32, b int32) int32 {
	if a > b {
		return a
	} else {
		return b
	}
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
