package blockchain

import (
	"bytes"
	"context"
	comm "github.com/gagarinchain/network/common"
	"github.com/gagarinchain/network/common/api"
	"github.com/gagarinchain/network/common/eth/common"
	"github.com/pkg/errors"
	"sort"
	"sync"
	"time"
)

type Synchronizer interface {
	//Requesting blocks (low, high]
	LoadBlocks(ctx context.Context, low int32, high int32, peer *comm.Peer) error
	LoadFork(ctx context.Context, headHeight int32, head common.Hash, peer *comm.Peer) error
}

type SynchronizerImpl struct {
	me                  *comm.Peer
	bsrv                BlockService
	bc                  api.Blockchain
	timeout             time.Duration
	headersLimit        int32
	depthLimit          int32
	headersAttemptCount int32
	blocksParallelLimit int32
	maxForkLength       int32
}

const DefaultTimeout = 50 * time.Millisecond
const DepthLimitConst = 4

var (
	DepthLimitExceeded    = errors.New("depth headersLimit exceeded")
	BatchLoadError        = errors.New("failed to load batch")
	HeadNotFoundError     = errors.New("failed to load head")
	LowHeightHeadersError = errors.New("the lowest height block has higher height")
)

//creates synchronizer
//to ommit depthLimit set -1
//to ommit timeout set -1
func CreateSynchronizer(me *comm.Peer, bsrv BlockService, bc api.Blockchain, timeout time.Duration, headersLimit int32,
	headersAttemptCount int32, blocksParallelLimit int32, depthLimit int32, maxForkLength int32) Synchronizer {
	if depthLimit == -1 {
		depthLimit = DepthLimitConst
	}
	if timeout == -1 {
		timeout = DefaultTimeout
	}
	return &SynchronizerImpl{
		me:                  me,
		bsrv:                bsrv,
		bc:                  bc,
		timeout:             timeout,
		depthLimit:          depthLimit,
		headersLimit:        headersLimit,
		headersAttemptCount: headersAttemptCount,
		blocksParallelLimit: blocksParallelLimit,
		maxForkLength:       maxForkLength,
	}
}

//Loads blocks from bottom up excluding lowest height and including highest (low;high]
//We filter and omit orphans, so we add only chains starting from common blockchain block upto highest loaded not guaranteed to be #high
func (s *SynchronizerImpl) LoadBlocks(ctx context.Context, low int32, high int32, peer *comm.Peer) error {
	return s.loadBlocks(ctx, low, high, -1, nil, peer)
}

// Request blocks (low, high]
func (s *SynchronizerImpl) loadBlocks(ctx context.Context, low int32, high int32, headHeight int32, head *common.Hash, peer *comm.Peer) error {
	log.Debugf("Requesting blocks (%v:%v]", low, high)
	//cut requested interval into headersLimit chunks
	for i := low; i < high; {
		chunkLow := i
		chunkHigh := chunkLow + s.headersLimit
		if high < chunkHigh {
			chunkHigh = high
		}

		success := false
		var headers []api.Header
		//try headersAttemptCount times to load headers batch and all blocks, then add them to bc
		for j := 0; j < int(s.headersAttemptCount); j++ {
			log.Debugf("Attempt %v", j)
			h, e := s.requestHeadersBatch(ctx, chunkLow, chunkHigh, headHeight, head, 0, peer)
			headers = h
			if e != nil {
				log.Error(e)
				continue
			}

			sort.Sort(HeadersByHeight(headers))
			var filteredHeaders []api.Header
			for k := 0; k < len(headers); k++ {
				if !s.bc.Contains(headers[k].Hash()) {
					filteredHeaders = append(filteredHeaders, headers[k])
				}
			}
			for k := 0; k < len(filteredHeaders); k += int(s.blocksParallelLimit) {
				l := k
				h := k + int(s.blocksParallelLimit)
				if h >= len(filteredHeaders) {
					h = len(filteredHeaders)
				}
				blocks, e := s.requestBlocks(ctx, filteredHeaders[l:h], peer)
				if e != nil {
					goto END
				}
				e = s.addBlocksTransactional(blocks)
				if e != nil {
					goto END
				}
			}
			success = true
			break
		END:
			continue
		}

		if !success || headers == nil {
			return BatchLoadError
		}

		topHeightLoaded := headers[len(headers)-1].Height()
		i = topHeightLoaded
	}
	return nil
}

func (s *SynchronizerImpl) requestHeadersBatch(ctx context.Context, low int32, high int32, headHeight int32, head *common.Hash, depth int32, peer *comm.Peer) ([]api.Header, error) {
	log.Debugf("Requesting headers for %v:%v, attempt %v", low, high, depth)
	if depth > s.depthLimit || low < 0 {
		return nil, DepthLimitExceeded
	}

	timeout, _ := context.WithTimeout(ctx, s.timeout)
	headers, e := s.bsrv.RequestHeaders(timeout, low, high, peer)
	if e != nil {
		return nil, e
	}

	sort.Sort(HeadersByHeight(headers))
	root := headers[0]

	log.Debugf("Root at height %v", root.Height())
	parent := s.bc.GetBlockByHash(root.Parent())
	//find if we have parent of lowest block in the blockchain
	if parent == nil { //we loaded chunk with parent not loaded previously, reload lower fork headers and add them to this chunk
		previousHead, e := s.requestHeadersBatch(ctx, low-s.headersLimit, low, headHeight, head, depth+1, peer)
		if e != nil {
			return nil, e
		}

		headers = append(headers, previousHead...)
		sort.Sort(HeadersByHeight(headers))
		root := headers[0]
		parent = s.bc.GetBlockByHash(root.Parent())

	}

	parentSiblingMapping := make(map[common.Hash]api.Header)
	for _, h := range headers {
		parentSiblingMapping[h.Parent()] = h
	}

	//determine chain heads
	var heads []api.Header
	for _, h := range headers {
		if h.Height() == headers[0].Height() {
			heads = append(heads, h)
		}
	}

	//filter siblings
	var filtered []api.Header
	for _, head := range heads {
		current := head
		found := true
		for found {
			filtered = append(filtered, current)
			current, found = parentSiblingMapping[current.Hash()]
		}
	}
	if filtered == nil {
		return nil, BatchLoadError
	}

	//try to find head of fork in loaded chain
	sort.Sort(HeadersByHeight(filtered))
	headers = filtered
	headFound := true
	if head != nil && headers[0].Height() <= headHeight && headers[len(headers)-1].Height() >= headHeight {
		headFound = false
		for _, h := range headers {
			if bytes.Equal(h.Hash().Bytes(), head.Bytes()) {
				headFound = true
				break
			}
		}
	}
	if !headFound {
		return nil, HeadNotFoundError
	}

	if headers[0].Height() > low+1 {
		return nil, LowHeightHeadersError
	}

	return headers, e
}

func (s *SynchronizerImpl) requestBlocks(ctx context.Context, headers []api.Header, peer *comm.Peer) ([]api.Block, error) {
	wg := &sync.WaitGroup{}
	amount := len(headers)
	wg.Add(amount)
	log.Infof("Requesting %v blocks", amount)

	type Exec struct {
		blocks []api.Block
		errors []error
		lock   *sync.Mutex
	}

	exec := &Exec{
		blocks: nil,
		errors: nil,
		lock:   &sync.Mutex{},
	}

	for i := 0; i < amount; i++ {
		go func(group *sync.WaitGroup, ind int, exec *Exec) {
			timeout, _ := context.WithTimeout(ctx, s.timeout)
			log.Debugf("Requesting block at %v", headers[ind].Height())
			b, e := s.bsrv.RequestBlock(timeout, headers[ind].Hash(), peer)
			if e != nil {
				log.Error("Can't read block", e)
				exec.lock.Lock()
				exec.errors = append(exec.errors, e)
				exec.lock.Unlock()
			} else {
				exec.lock.Lock()
				exec.blocks = append(exec.blocks, b)
				exec.lock.Unlock()
			}

			group.Done()
		}(wg, i, exec)

	}
	wg.Wait()

	if len(exec.errors) > 0 {
		log.Errorf("Received %v errors", len(exec.errors))
		return nil, exec.errors[0]
	}

	log.Infof("Received %v blocks", len(exec.blocks))
	return exec.blocks, nil
}

//Request all blocks starting at top committed block (not included) in blockchain, which all replicas must have (not sure that all, but 2 *f + 1)
//We filter and omit orphans, so we add only chains starting from common blockchain block upto highest loaded not guaranteed to be #high
func (s *SynchronizerImpl) LoadFork(ctx context.Context, headHeight int32, head common.Hash, peer *comm.Peer) error {
	topCommited := s.bc.GetTopCommittedBlock()
	topCommitedHeight := topCommited.Height()

	low := int32(0)
	if topCommitedHeight < headHeight-s.maxForkLength {
		low = headHeight - s.maxForkLength
	} else {
		low = topCommitedHeight
	}

	return s.loadBlocks(ctx, low, headHeight, headHeight, &head, peer)
}

func (s *SynchronizerImpl) addBlocksTransactional(blocks []api.Block) error {
	errorIndex := -1
	var err error
	sort.Sort(ByHeight(blocks))
	for i, b := range blocks {
		log.Debugf("Adding block %v", b.Height())
		if err = s.bc.AddBlock(b); err != nil {
			errorIndex = i
			log.Infof("Error adding block [%v], %v", b.Header().Hash().Hex(), err.Error())
			break
		}
	}
	//compensation logic, should clean up previous additions
	if errorIndex > -1 {
		toCleanUp := blocks[:errorIndex+1]
		sort.Sort(sort.Reverse(ByHeight(toCleanUp)))
		for _, b := range toCleanUp {
			log.Debugf("Removing block %v", b.Height())
			if err := s.bc.RemoveBlock(b); err != nil {
				log.Fatal(err)
				panic("Can't delete added previously block, reload blockchain")
			}
		}
		return err
	}
	return nil
}
