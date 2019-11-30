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

// Note that this can't be higher than protocol.go#RequestBlockHeaderBatchSizeLimit - the server-side limit.
const RequestBlocksBatchSize = 20
const RequestBlocksMaxParallelBatches = 3
const RequestBlocksMaxParallelBlocksPerBatch = 10
const RequestBlocksMaxRetries = 5 //TODO: We should probably do at least (f+1) retries with different peers.

type Synchronizer interface {
	//Requests blocks at (low, high] height one by one from the specified peer.
	RequestBlocks(ctx context.Context, low int32, high int32, peer *comm.Peer) error
	//Requests blocks at (low, high] height in batches, concurrently from multiple (random) peers.
	RequestBlocksParallel(ctx context.Context, low int32, high int32) error
	RequestFork(ctx context.Context, hash common.Hash, peer *comm.Peer) error
}

type SynchronizerImpl struct {
	me   *comm.Peer
	bsrv BlockService
	bc   *Blockchain
}

type blockBatch struct {
	low       int32
	high      int32
	done      chan bool
	prevBatch *blockBatch
}

func CreateSynchronizer(me *comm.Peer, bsrv BlockService, bc *Blockchain) Synchronizer {
	return &SynchronizerImpl{
		me:   me,
		bsrv: bsrv,
		bc:   bc,
	}
}

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

func (s *SynchronizerImpl) RequestBlocksParallel(ctx context.Context, low int32, high int32) error {
	// The (low, high] range can be potentially huge (e.g. a million of blocks).
	// We're going to split it into small batches, e.g. 100 blocks per batch, and will process the batches sequentially,
	// i.e. we'll first load blocks 1-100, validate and store them in the local blockchain, and then will proceed with
	// processing blocks 101-200, to avoid using too much memory. If the process is interrupted in the middle, we can
	// continue from where we stopped the last batch.
	// To optimize the network bandwidth usage and deal with the "long tail" problem (loaded 99 blocks fast, but 1 block
	// is stuck), we'll start loading batch 2 in parallel while batch 1 is still loading, but won't start validating and
	// processing it until batch 1 is processed.
	// We can load all block headers in a batch with one BlockHeaderBatchRequest, and then load the blocks themselves
	// one by one. It doesn't make sense to batch the block loading as every block will typically be pretty big, ~1 MB.

	if low >= high {
		panic(fmt.Errorf("invalid low..high range: (%d, %d]", low, high))
	}

	quitChan := make(chan bool)
	errorChan := make(chan error, RequestBlocksMaxParallelBatches)
	batchQueue := make(chan *blockBatch, RequestBlocksMaxParallelBatches)

	// Start batch generator:

	go func() {
		defer close(batchQueue)

		var pb *blockBatch = nil

		for lo := low; lo < high; lo += RequestBlocksBatchSize {
			// Create batches. For example, for lo = 5, high = 57, RequestBlocksBatchSize = 20:
			// 		 (5, 20]
			// 		(20, 40]
			// 		(40, 57]

			hi := min(high, lo+RequestBlocksBatchSize)

			b := &blockBatch{
				low:       lo,
				high:      hi,
				done:      make(chan bool),
				prevBatch: pb,
			}

			pb = b

			select {
			case batchQueue <- b:
				log.Debugf("created batch (%d, %d]", lo, hi)
			case <-quitChan:
				log.Debug("batch generator stopping due to an error")
				return
			}
		}
	}()

	// Start up to RequestBlocksMaxParallelBatches batch processors:

	batchProcessorDone := make(chan bool, RequestBlocksMaxParallelBatches)

	for i := 0; i < RequestBlocksMaxParallelBatches; i++ {
		go func(index int) {
			for {
				select {
				case <-quitChan:
					log.Debugf("batch processor #%v stopping due to an error", index)
					return // Abort early because of an error in another thread.
				case b, open := <-batchQueue:
					if !open {
						log.Debugf("batch processor #%v processed all batches", index)
						batchProcessorDone <- true
						return // Processed all batches successfully.
					}

					if e := s.loadBlockBatch(ctx, b, quitChan); e != nil {
						log.Warningf("batch processor #%v failed to process batch %v: %v", index, b, e)
						errorChan <- e
						return // Error in this thread.
					}

					log.Debugf("batch processor #%v processed batch %v", index, b)
					close(b.done) // Signal that this batch is done, so that the next one could start processing.
				}
			}
		}(i)
	}

	for cnt := 0; cnt < RequestBlocksMaxParallelBatches; {
		select {
		case e := <-errorChan:
			close(quitChan)
			return e
		case <-batchProcessorDone:
			cnt++
		}
	}

	return nil
}

func (s *SynchronizerImpl) loadBlockBatch(ctx context.Context, b *blockBatch, quitChan chan bool) error {
	// Start loading blocks into memory, but don't process them until the previous batch is done.

	// We don't know the exact number of blocks in the batch (could be more than 1 at a specific height),
	// so we'll buffer 2x batch size blocks just in case:
	blocksInBatchBufferSize := (b.high - b.low) * 2

	// Load all block headers for this batch with one batch request.
	// This needs to be synchronous as we'll need these headers to validate the actual blocks.

	headers := make([]*Header, 0, blocksInBatchBufferSize)

tryLoop:
	for try := 1; try <= RequestBlocksMaxRetries; try++ {
		respHeaderChan, respErrChan := s.bsrv.RequestBlockHeaderBatch(ctx, b.low, b.high, nil)
		headers = headers[:0]

	readLoop:
		for {
			select {
			case <-quitChan:
				return fmt.Errorf("header loader stopping due to an error in another batch")
			case header, open := <-respHeaderChan:
				if !open {
					sort.Sort(ByHeaderHeight(headers))
					e := validateBatchHeaders(headers)

					if e != nil {
						if try >= RequestBlocksMaxRetries {
							return e
						} else {
							log.Debugf("batch %v failed to validate headers (but will re-try): %v", b, e)
							break readLoop // Break the header reading loop but continue the try loop.
						}
					}

					break tryLoop // Read all headers from the response.
				} else {
					headers = append(headers, header)
				}
			case e := <-respErrChan:
				if try >= RequestBlocksMaxRetries {
					return e
				} else {
					log.Debugf("batch %v header loader failed (but will re-try): %v", b, e)
					break readLoop // Break the header reading loop but continue the try loop.
				}
			}
		}
	}

	blockRequestQueue := make(chan int32, RequestBlocksMaxParallelBlocksPerBatch)

	go func() {
		defer close(blockRequestQueue)

		for h := b.low + 1; h <= b.high; h++ {
			select {
			case blockRequestQueue <- h:
			case <-quitChan:
				log.Debugf("batch %v processing cancelled due to an error in another batch", b)
				return
			}
		}
	}()

	// Start up to RequestBlocksMaxParallelBlocksPerBatch parallel block loaders:

	blockResults := make(chan *Block, blocksInBatchBufferSize)
	blockLoaderDone := make(chan bool, RequestBlocksMaxParallelBlocksPerBatch)
	errorChan := make(chan error, RequestBlocksMaxParallelBlocksPerBatch)

	for i := 0; i < RequestBlocksMaxParallelBlocksPerBatch; i++ {
		go func(index int) {
			for {
				select {
				case <-quitChan:
					log.Debugf("batch %v block loader #%v stopping due to an error in another batch", b, index)
					return
				case height, open := <-blockRequestQueue:
					if !open {
						blockLoaderDone <- true
						return // No more blocks to process.
					}

					blocks := make([]*Block, 0, 1)

				tryLoop:
					for try := 1; try <= RequestBlocksMaxRetries; try++ {
						respBlockChan, respErrChan := s.bsrv.RequestBlocksAtHeight(ctx, height, nil)
						blocks = blocks[:0]

					readLoop:
						for {
							select {
							case <-quitChan:
								log.Debugf("batch %v block loader #%v "+
									"stopping due to an error in another batch", b, index)
								return
							case block, open := <-respBlockChan:
								if !open {
									break tryLoop // Read all blocks from the response.
								} else {
									e := s.validateBlockInBatch(headers, block, height)

									if e == nil {
										blocks = append(blocks, block)
									} else {
										if try >= RequestBlocksMaxRetries {
											errorChan <- e
											return
										} else {
											log.Debugf("batch %v block loader #%v "+
												"failed to validate block @%v (but will re-try): %v", b, index, height, e)
											break readLoop // Break the block reading loop but continue the try loop.
										}
									}
								}
							case e := <-respErrChan:
								if try >= RequestBlocksMaxRetries {
									errorChan <- e
									return
								} else {
									log.Debugf("batch %v block loader #%v "+
										"failed to load block @%v (but will re-try): %v", b, index, height, e)
									break readLoop // Break the block reading loop but continue the try loop.
								}
							}
						}
					}

					for _, b := range blocks {
						blockResults <- b
					}
				}
			}
		}(i)
	}

	// Wait for the previous batch to complete (if needed) before processing this one:

	if b.prevBatch != nil {
		log.Debugf("batch %v waiting for batch %v to complete...", b, b.prevBatch)

		select {
		case <-quitChan:
			return fmt.Errorf("loading cancelled due to an error in another batch")
		case e := <-errorChan:
			return e
		case <-b.prevBatch.done:
			break
		}
	}

	// Wait for all the block loaders to finish:

	for cnt := 0; cnt < RequestBlocksMaxParallelBlocksPerBatch; {
		select {
		case <-quitChan:
			return fmt.Errorf("loading cancelled due to an error in another batch")
		case e := <-errorChan:
			return e
		case <-blockLoaderDone:
			cnt++
		}
	}

	close(blockResults)

	// Process blocks:

	blocks := make([]*Block, 0, len(headers))

	for block := range blockResults {
		blocks = append(blocks, block)
	}

	sort.Sort(ByHeight(blocks))

	// Some sanity checks. Shouldn't happen as all these blocks should've been already validated during load.

	if len(blocks) == 0 {
		panic("no blocks received")
	}
	if blocks[0].Height() != b.low+1 {
		panic("unexpected fork tail")
	}
	if blocks[len(blocks)-1].Height() != b.high {
		panic("unexpected fork head")
	}

	return s.addBlocksTransactional(blocks)
}

// Validates all headers in a (low, high] batch of blocks, without the actual block data.
func (s *SynchronizerImpl) validateBatchHeaders(h []*Header) error {
	//TODO: Implement batch header validation. [!!!]
	return nil
}

// Given all headers for a (low, high] batch of blocks, checks if the block b belongs to the batch and validates it.
//
// All blocks at height [0, low] have to be already added to the blockchain.
// Blocks at height (low, b.height) do not have to be added to the blockchain,
// the block b is validated based on the header information only.
func (s *SynchronizerImpl) validateBlockInBatch(batchHeaders []*Header, b *Block, expectedHeight int32) error {
	//TODO: Implement single block validation. [!!!]

	if b.Height() != expectedHeight {
		return fmt.Errorf("unexpected block height %v, expected %v", b.Height(), expectedHeight)
	}

	return nil
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

func min(a int32, b int32) int32 {
	if a < b {
		return a
	} else {
		return b
	}
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

func (b blockBatch) String() string {
	return fmt.Sprintf("(%v, %v]", b.low, b.high)
}
