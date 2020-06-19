package blockchain

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/emirpasic/gods/utils"
	cmn "github.com/gagarinchain/common"
	"github.com/gagarinchain/common/api"
	"github.com/gagarinchain/common/eth/common"
	"github.com/gagarinchain/common/eth/crypto"
	"github.com/gagarinchain/network/blockchain/state"
	"github.com/gagarinchain/network/blockchain/tx"
	"github.com/gagarinchain/network/storage"
	"github.com/op/go-logging"
	"sync"
	"time"
)
import "github.com/emirpasic/gods/maps/treemap"

const TxLimit = 50

var (
	InvalidStateHashError = errors.New("invalid block state hash")
)

type BlockchainPersister struct {
	Storage storage.Storage
}

func (bp *BlockchainPersister) PutCurrentTopHeight(currentTopHeight int32) error {
	return bp.Storage.Put(storage.CurrentTopHeight, nil, storage.Int32ToByte(currentTopHeight))
}

func (bp *BlockchainPersister) GetCurrentTopHeight() (val int32, err error) {
	value, err := bp.Storage.Get(storage.CurrentTopHeight, nil)
	if err != nil {
		return storage.DefaultIntValue, nil
	}
	return storage.ByteToInt32(value)
}

func (bp *BlockchainPersister) PutTopCommittedHeight(currentTopHeight int32) error {
	return bp.Storage.Put(storage.TopCommittedHeight, nil, storage.Int32ToByte(currentTopHeight))
}

func (bp *BlockchainPersister) GetTopCommittedHeight() (val int32, err error) {
	value, err := bp.Storage.Get(storage.TopCommittedHeight, nil)
	if err != nil {
		return storage.DefaultIntValue, nil
	}
	return storage.ByteToInt32(value)
}

func (bp *BlockchainPersister) PutHeightIndexRecord(b api.Block) error {
	key := make([]byte, binary.MaxVarintLen64)
	binary.PutVarint(key, int64(b.Height()))

	found := bp.Storage.Contains(storage.HeightIndex, key)
	var indexValue []byte

	if found {
		value, err := bp.Storage.Get(storage.HeightIndex, key)
		if err != nil {
			return err
		}
		indexValue = value
	}

	indexValue = append(indexValue, b.Header().Hash().Bytes()...)
	return bp.Storage.Put(storage.HeightIndex, key, indexValue)
}

func (bp *BlockchainPersister) GetHeightIndexRecord(height int32) (hashes []common.Hash, err error) {
	key := make([]byte, binary.MaxVarintLen64)
	binary.PutVarint(key, int64(height))

	val, err := bp.Storage.Get(storage.HeightIndex, key)
	if err != nil {
		return nil, err
	}

	hashes = make([]common.Hash, len(val)/common.HashLength)

	for i := 0; i < len(val)/common.HashLength; i++ {
		hashes[i] = common.BytesToHash(val[i*common.HashLength : (i+1)*common.HashLength])
	}

	return hashes, nil
}

type BlockchainImpl struct {
	indexGuard *sync.RWMutex
	//index for storing blocks by their hash
	blocksByHash map[common.Hash]api.Block
	//indexes for storing blocks according to their height,
	// <int32, *Block>
	committedChainByHeight *treemap.Map
	//indexes for storing block arrays according to their height, while not committed head may contain forks,
	//<int32, []*Block>
	uncommittedTreeByHeight *treemap.Map
	txPool                  tx.TransactionPool
	stateDB                 state.DB
	proposerGetter          api.ProposerForHeight
	blockPersister          *BlockPersister
	chainPersister          *BlockchainPersister
	delta                   time.Duration
	onNewBlockCreated       api.OnNewBlockCreated
	bus                     cmn.EventBus
}

func (bc *BlockchainImpl) SetProposerGetter(proposerGetter api.ProposerForHeight) {
	bc.proposerGetter = proposerGetter
}

var log = logging.MustGetLogger("blockchain")

func CreateBlockchainFromGenesisBlock(cfg *BlockchainConfig) *BlockchainImpl {
	zero := CreateGenesisBlock()
	if cfg.EventBus == nil {
		cfg.EventBus = &cmn.NullBus{}
	}
	var onBlock api.OnNewBlockCreated
	if cfg.OnNewBlockCreated == nil {
		onBlock = &api.NullOnNewBlockCreated{}
	} else {
		onBlock = cfg.OnNewBlockCreated
	}
	blockchain := &BlockchainImpl{
		blocksByHash:            make(map[common.Hash]api.Block),
		committedChainByHeight:  treemap.NewWith(utils.Int32Comparator),
		uncommittedTreeByHeight: treemap.NewWith(utils.Int32Comparator),
		indexGuard:              &sync.RWMutex{},
		txPool:                  cfg.Pool,
		stateDB:                 cfg.Db,
		proposerGetter:          cfg.ProposerGetter,
		blockPersister:          cfg.BlockPerister,
		chainPersister:          cfg.ChainPersister,
		delta:                   cfg.Delta,
		bus:                     cfg.EventBus,
		onNewBlockCreated:       onBlock,
	}

	var s *state.Snapshot
	if cfg.Seed != nil {
		s = state.NewSnapshotWithAccounts(zero.Header().Hash(), common.Address{}, cfg.Seed)
	}
	if e := cfg.Db.Init(zero.Header().Hash(), s); e != nil {
		panic("can't init state DB")
	}
	if _, err := blockchain.AddBlock(zero); err != nil {
		log.Error(err)
		panic("can't add genesis block")
	}
	return blockchain
}

func CreateBlockchainFromStorage(cfg *BlockchainConfig) *BlockchainImpl {
	pblock := &BlockPersister{Storage: cfg.Storage}
	pchain := &BlockchainPersister{Storage: cfg.Storage}

	topHeight, err := pchain.GetCurrentTopHeight()
	log.Debugf("loaded topheight %v", topHeight)

	if err != nil || topHeight < 0 {
		return CreateBlockchainFromGenesisBlock(cfg)
	}

	if cfg.EventBus == nil {
		cfg.EventBus = &cmn.NullBus{}
	}
	var onBlock api.OnNewBlockCreated
	if cfg.OnNewBlockCreated == nil {
		onBlock = &api.NullOnNewBlockCreated{}
	} else {
		onBlock = cfg.OnNewBlockCreated
	}
	blockchain := &BlockchainImpl{
		blocksByHash:            make(map[common.Hash]api.Block),
		committedChainByHeight:  treemap.NewWith(utils.Int32Comparator),
		uncommittedTreeByHeight: treemap.NewWith(utils.Int32Comparator),
		indexGuard:              &sync.RWMutex{},
		txPool:                  cfg.Pool,
		stateDB:                 cfg.Db,
		blockPersister:          pblock,
		chainPersister:          pchain,
		delta:                   cfg.Delta,
		bus:                     cfg.EventBus,
		onNewBlockCreated:       onBlock,
	}

	topCommittedHeight, err := pchain.GetTopCommittedHeight()
	blockchain.indexGuard.RLock()
	defer blockchain.indexGuard.RUnlock()

	//TODO this place should be heavily optimized, i think we should preload everything except transactions for committed part of bc, headers and signatures will take 1k per block amortised
	for i := topHeight; i >= 0; i-- {
		hashes, err := pchain.GetHeightIndexRecord(i)
		if err != nil {
			log.Fatal("Can't load index", err)
		}
		for _, h := range hashes {
			b, err := pblock.Load(h)
			if b == nil {
				log.Warningf("Block with hash %v is found in index, but is not stored in DB itself", h.Hex())
			}
			if err != nil {
				log.Fatal(err)
			}
			blockchain.blocksByHash[h] = b

			if i > topCommittedHeight {
				if err := blockchain.addUncommittedBlock(b); err != nil {
					log.Fatal(err)
				}
			} else {
				blockchain.committedChainByHeight.Put(b.Height(), b)
			}
		}
	}

	return blockchain
}

func (bc *BlockchainImpl) GetBlockByHash(hash common.Hash) (block api.Block) {
	bc.indexGuard.RLock()
	defer bc.indexGuard.RUnlock()

	return bc.getBlockByHash(hash)
}

func (bc *BlockchainImpl) getBlockByHash(hash common.Hash) api.Block {
	block := bc.blocksByHash[hash]

	if block == nil {
		block, er := bc.blockPersister.Load(hash)
		if er != nil {
			log.Error(er)
		}
		return block
	}

	return block
}

func (bc *BlockchainImpl) GetBlockByHeight(height int32) (res []api.Block) {
	block, ok := bc.committedChainByHeight.Get(height)
	if !ok {
		blocks, ok := bc.uncommittedTreeByHeight.Get(height)
		if ok {
			res = append(res, blocks.([]api.Block)...)
		} else { //TODO it seems like a hack, we should preload blockchain structure from index to memory
			hashes, err := bc.chainPersister.GetHeightIndexRecord(height)
			if err == nil {
				for _, hash := range hashes {
					block, err := bc.blockPersister.Load(hash)
					if err == nil {
						res = append(res, block)
					}
				}
			}
		}
	} else {
		res = append(res, block.(api.Block))
	}
	return res
}

//Returns fork starting at block with headHash and including block at height
func (bc *BlockchainImpl) GetFork(height int32, headHash common.Hash) (res []api.Block) {
	bc.indexGuard.RLock()
	defer bc.indexGuard.RUnlock()

	head := bc.blocksByHash[headHash]
	hash := headHash
	if head.Height()-height+1 < 0 {
		log.Error("len out of range")
		return nil
	}
	res = make([]api.Block, head.Height()-height+1)
	for i := 0; i < len(res); i++ {
		head = bc.blocksByHash[hash]
		res[i] = head
		hash = head.Header().Parent()
	}

	return res
}

func (bc *BlockchainImpl) Contains(hash common.Hash) bool {
	return bc.GetBlockByHash(hash) != nil
}

// Returns three certified blocks (Bzero, Bone, Btwo) from 3-chain
// B|zero <-- B|one <-- B|two <--...--  B|head
func (bc *BlockchainImpl) GetThreeChain(twoHash common.Hash) (zero api.Block, one api.Block, two api.Block) {
	two = bc.GetBlockByHash(twoHash)
	if two == nil {
		return nil, nil, nil
	}

	one = bc.GetBlockByHash(two.QRef().Hash())
	if one == nil {
		return nil, nil, two
	}

	zero = bc.GetBlockByHash(one.QRef().Hash())
	if zero == nil {
		return nil, one, two
	}

	return zero, one, two
}

//move uncommitted chain with b as head to committed and analyze rejected forks
//toCommit is ordered by height ascending
//TODO should we delete orphans from BlocksByHash index? they hang there now
func (bc *BlockchainImpl) OnCommit(b api.Block) (toCommit []api.Block, orphans *treemap.Map, err error) {
	orphans = treemap.NewWith(utils.Int32Comparator)

	low, _ := bc.uncommittedTreeByHeight.Min()
	if low == nil {
		return nil, nil, errors.New("empty commit index")
	}

	//lets collect all predecessors of block to commit, and make orphans map
	phash := b.Header().Hash()
	for height := b.Height(); height >= low.(int32); height-- {
		val, found := bc.uncommittedTreeByHeight.Get(height)
		currentBag := val.([]api.Block)
		if !found {
			return nil, nil, errors.New("bad index structure")
		}
		for i, el := range currentBag {
			if el.Header().Hash() == phash {
				toCommit = append(toCommit, el)
				orph := append(currentBag[:i], currentBag[i+1:]...)
				if len(orph) != 0 {
					orphans.Put(height, orph)
				}
				bc.uncommittedTreeByHeight.Remove(height)
				phash = el.Header().Parent()
				break
			}
		}
	}

	//lets filter orphan blocks
	hashes := map[common.Hash]bool{b.Header().Hash(): true}
	max, _ := bc.uncommittedTreeByHeight.Max()
	for height := b.Height() + 1; height <= max.(int32); height++ {
		val, found := bc.uncommittedTreeByHeight.Get(height)
		currentBag := val.([]api.Block)
		if !found {
			return nil, nil, errors.New("bad index structure")
		}
		var o []api.Block
		var c []api.Block
		for _, el := range currentBag {
			_, ok := hashes[el.Header().Parent()]
			if ok {
				c = append(c, el)
				hashes[el.Header().Hash()] = true
			} else {
				o = append(o, el)
			}
		}
		orphans.Put(height, o)
		bc.uncommittedTreeByHeight.Put(height, c)
	}

	//reverse toCommit
	for left, right := 0, len(toCommit)-1; left < right; left, right = left+1, right-1 {
		toCommit[left], toCommit[right] = toCommit[right], toCommit[left]
	}

	//add blocks marked to commit to committed index
	for _, k := range toCommit {
		bc.indexGuard.Lock()
		bc.committedChainByHeight.Put(k.Header().Height(), k)
		bc.indexGuard.Unlock()

		err := bc.chainPersister.PutTopCommittedHeight(k.Height())
		if err != nil {
			log.Error(err)
		}
	}

	for _, k := range toCommit {
		bc.bus.FireEvent(&cmn.Event{
			Payload: k.Header().Hash(),
			T:       cmn.Committed,
		})
	}

	//remove committed transactions from the pool
	for _, b := range toCommit {
		it := b.Txs()
		for next := it.Next(); next != nil; next = it.Next() {
			bc.txPool.Remove(next)
		}
	}

	//release orphan states
	_, v := orphans.Min()
	lowestHeightOrphans := v.([]api.Block)
	for _, o := range lowestHeightOrphans {
		if e := bc.stateDB.Release(o.Header().Hash()); e != nil {
			log.Error(e)
		}
	}

	return toCommit, orphans, nil
}

func (bc *BlockchainImpl) GetHead() api.Block {
	bc.indexGuard.RLock()
	defer bc.indexGuard.RUnlock()

	//TODO probably it is not correct^ have to return head of PREF block branch here
	_, blocks := bc.uncommittedTreeByHeight.Max()
	var b = blocks.([]api.Block)
	return b[0]
}

func (bc *BlockchainImpl) GetHeadRecord() api.Record {
	r, f := bc.stateDB.Get(bc.GetHead().Header().Hash())
	if !f {
		log.Error("Can't find head snapshot")
	}
	return r
}

func (bc *BlockchainImpl) GetTopHeight() int32 {
	return bc.GetHead().Header().Height()
}
func (bc *BlockchainImpl) GetTopHeightBlocks() []api.Block {
	bc.indexGuard.RLock()
	defer bc.indexGuard.RUnlock()

	_, blocks := bc.uncommittedTreeByHeight.Max()
	var b = blocks.([]api.Block)
	return b
}

func (bc *BlockchainImpl) AddBlock(block api.Block) ([]api.Receipt, error) {
	log.Infof("Adding block with hash [%v]", block.Header().Hash().Hex())
	//spew.Dump(block)

	bc.indexGuard.Lock()
	defer bc.indexGuard.Unlock()

	if bc.blocksByHash[block.Header().Hash()] != nil || bc.blockPersister.Contains(block.Header().Hash()) {
		log.Warningf("Block with hash [%v] already exists", block.Header().Hash().Hex())
		//TODO mb store receipts and return produced previously
		return block.Receipts(), nil
	}

	if bc.blocksByHash[block.Header().Parent()] == nil && !bc.blockPersister.Contains(block.Header().Parent()) && block.Height() != 0 {
		return nil, errors.New(fmt.Sprintf("block with hash [%v] don't have parent loaded to blockchain", block.Header().Hash().Hex()))
	}

	if block.QC() != nil {
		bc.updateBlockSignature(block)
	}

	var receipts []api.Receipt
	//todo remove this hack by handling genesis in special way
	//probably handle already executed blocks here with && block.Receipts() == nil
	if block.Height() != 0 {
		var err error
		if receipts, err = bc.applyTransactionsAndValidate(block); err != nil {
			return nil, err
		}
		log.Debugf("%v receipts produced", len(receipts))
		block.SetReceipts(receipts)
	}

	if err := bc.addUncommittedBlock(block); err != nil {
		return nil, err
	}

	if err := bc.blockPersister.Persist(block); err != nil {
		return nil, err
	}
	if err := bc.chainPersister.PutHeightIndexRecord(block); err != nil {
		return nil, err
	}

	bc.bus.FireEvent(&cmn.Event{
		T:       cmn.BlockAdded,
		Payload: block.GetMessage(),
	})

	return receipts, nil
}

func (bc *BlockchainImpl) updateBlockSignature(b api.Block) {
	if b.Height() == 0 {
		return
	}
	qrefHash := b.QC().QrefBlock().Hash()
	qrefBlock := bc.getBlockByHash(qrefHash)
	if qrefBlock.Height() == 0 {
		return
	}

	qrefBlock.SetSignature(b.QC().SignatureAggregate())

	if err := bc.blockPersister.Update(qrefBlock); err != nil {
		log.Error("error while block signature update", err)
	}

}

func (bc *BlockchainImpl) addUncommittedBlock(block api.Block) error {
	value, found := bc.uncommittedTreeByHeight.Get(block.Header().Height())
	if !found {
		value = make([]api.Block, 0)
		if err := bc.chainPersister.PutCurrentTopHeight(block.Header().Height()); err != nil {
			return err
		}
	}

	value = append(value.([]api.Block), block)
	bc.blocksByHash[block.Header().Hash()] = block
	bc.uncommittedTreeByHeight.Put(block.Header().Height(), value)

	return nil
}

func (bc *BlockchainImpl) RemoveBlock(block api.Block) error {
	log.Infof("Removing block with hash [%v]", block.Header().Hash().Hex())
	//spew.Dump(block)

	bc.indexGuard.Lock()
	defer bc.indexGuard.Unlock()

	if bc.blocksByHash[block.Header().Hash()] == nil && !bc.blockPersister.Contains(block.Header().Hash()) {
		log.Warningf("Block with hash [%v] is absent", block.Header().Hash().Hex())
		return nil
	}

	//we can delete only leafs, so check whether this block is not parent of any
	value, found := bc.uncommittedTreeByHeight.Get(block.Height() + 1)
	if found {
		nextHeight := value.([]api.Block)
		for _, next := range nextHeight {
			if block.Header().Parent().Big() == next.Header().Hash().Big() {
				return errors.New(fmt.Sprintf("block [%v] has sibling [%v] on next level",
					block.Header().Hash().Hex(), next.Header().Hash().Hex()))
			}
		}
	}

	delete(bc.blocksByHash, block.Header().Hash())

	onHeight, found := bc.uncommittedTreeByHeight.Get(block.Height())
	if found {
		thisHeight := onHeight.([]api.Block)
		for i, b := range thisHeight {
			if b.Header().Hash().Big() == block.Header().Hash().Big() {
				thisHeight = append(thisHeight[:i], thisHeight[i+1:]...)
				bc.uncommittedTreeByHeight.Put(block.Height(), thisHeight)
				break
			}
		}
	}

	return nil
}

func (bc *BlockchainImpl) GetGenesisBlock() api.Block {
	bc.indexGuard.RLock()
	defer bc.indexGuard.RUnlock()

	return bc.getGenesisBlock()
}

func (bc *BlockchainImpl) getGenesisBlock() api.Block {
	value, ok := bc.committedChainByHeight.Get(int32(0))
	if !ok {
		if h, okk := bc.uncommittedTreeByHeight.Get(int32(0)); okk {
			value = h.([]api.Block)[0]
		}
	}

	return value.(api.Block)
}

func (bc *BlockchainImpl) GetGenesisCert() api.QuorumCertificate {
	return bc.GetGenesisBlock().QC()
}

func (bc BlockchainImpl) IsSibling(sibling api.Header, ancestor api.Header) bool {
	//Genesis block is ancestor of every block
	if ancestor.IsGenesisBlock() {
		return true
	}

	//todo should load blocks earlier, remove this call
	parent := bc.GetBlockByHash(sibling.Parent())

	if parent.Header().IsGenesisBlock() || parent.Header().Height() < ancestor.Height() {
		return false
	}

	if bytes.Equal(parent.Header().Hash().Bytes(), ancestor.Hash().Bytes()) {
		return true
	}

	return bc.IsSibling(parent.Header(), ancestor)
}

func (bc *BlockchainImpl) NewBlock(parent api.Block, qc api.QuorumCertificate, data []byte) api.Block {
	return bc.newBlock(parent, qc, data, true)
}

func (bc *BlockchainImpl) newBlock(parent api.Block, qc api.QuorumCertificate, data []byte, withTransactions bool) api.Block {
	proposer := bc.proposerGetter.ProposerForHeight(parent.Header().Height() + 1).GetAddress() //this block will be the block of next height
	r, e := bc.stateDB.Create(parent.Header().Hash(), proposer)
	if e != nil {
		log.Error("Can't create new block", e)
		return nil
	}

	var txs []api.Transaction
	var receipts []api.Receipt
	if withTransactions {
		txs, receipts = bc.collectTransactions(r)
	}

	header := createHeader(
		parent.Header().Height()+1,
		common.Hash{},
		qc.GetHash(),
		BuildProof(txs),
		r.RootProof(),
		crypto.Keccak256Hash(data),
		parent.Header().Hash(),
		time.Now().UTC().Round(time.Second))

	builder := NewBlockBuilderImpl()
	builder.SetHeader(header).SetQC(qc).SetData(data).SetTxs(txs)
	created, e := bc.onNewBlockCreated.OnNewBlockCreated(context.Background(), builder, receipts)
	if e != nil {
		log.Error(e)
		return nil
	}

	var block api.Block
	if created != nil {
		block = created
	} else {
		block = builder.Build()
	}
	_, err := bc.stateDB.Commit(parent.Header().Hash(), header.Hash())

	if err != nil {
		log.Error("Can't create new block", err)
		return nil
	}

	block.SetReceipts(receipts)
	return block
}

func (bc *BlockchainImpl) collectTransactions(s api.Record) (txs []api.Transaction, receipts []api.Receipt) {
	c := context.Background()
	timeout, _ := context.WithTimeout(c, bc.delta)
	chunks := bc.txPool.Drain(timeout)
	i := 0
	for chunk := range chunks {
		for _, t := range chunk {
			log.Debugf("tx hash %v", t.Hash().Hex())
			rs, err := s.ApplyTransaction(t)
			if err != nil {
				continue
			}
			//txs.InsertOrUpdate([]byte(t.Hash().Hex()), t.Serialized())
			txs = append(txs, t)
			receipts = append(receipts, rs...)
			i++

			if i >= TxLimit {
				return
			}
		}
	}

	log.Debugf("Collected %v txs", i)
	return txs, receipts
}

func (bc *BlockchainImpl) PadEmptyBlock(head api.Block, qc api.QuorumCertificate) api.Block {
	block := bc.newBlock(head, qc, []byte(""), false)

	if _, e := bc.AddBlock(block); e != nil {
		log.Error("Can't add empty block")
		return nil
	}

	return block
}

func (bc *BlockchainImpl) GetGenesisBlockSignedHash(key *crypto.PrivateKey) *crypto.Signature {
	hash := bc.GetGenesisBlock().Header().Hash()
	sig := crypto.Sign(hash.Bytes(), key)
	if sig == nil {
		log.Fatal("Can't sign genesis block")
	}
	return sig

}
func (bc *BlockchainImpl) ValidateGenesisBlockSignature(signature *crypto.Signature, address common.Address) bool {
	hash := bc.GetGenesisBlock().Header().Hash()
	signatureAddress := crypto.PubkeyToAddress(crypto.NewPublicKey(signature.Pub()))
	return crypto.Verify(hash.Bytes(), signature) && bytes.Equal(address.Bytes(), signatureAddress.Bytes())
}

func (bc *BlockchainImpl) GetTopCommittedBlock() api.Block {
	bc.indexGuard.RLock()
	defer bc.indexGuard.RUnlock()
	_, block := bc.committedChainByHeight.Max()
	//todo think about committing genesis at making QC
	if block == nil {
		return bc.GetGenesisBlock()
	}
	return block.(api.Block)
}

func (bc *BlockchainImpl) UpdateGenesisBlockQC(certificate api.QuorumCertificate) {
	bc.GetGenesisBlock().SetQC(certificate)
	//we can simply put new block and replace existing. in fact ignoring height index is not a problem for us since Genesis is hashed without it's QC
	if err := bc.blockPersister.Persist(bc.GetGenesisBlock()); err != nil {
		log.Error(err)
		return
	}
}

func (bc *BlockchainImpl) applyTransactionsAndValidate(block api.Block) (receipts []api.Receipt, err error) {
	_, f := bc.stateDB.Get(block.Header().Hash())
	if f {
		log.Debug("Found block that was created by us and already processed, skip this step")
		//TODO think about storing receipts and returning them here
		return block.Receipts(), nil
	}

	r, e := bc.stateDB.Create(block.Header().Parent(), bc.proposerGetter.ProposerForHeight(block.Height()).GetAddress())
	if e != nil {
		return receipts, e
	}

	iterator := block.Txs()
	for iterator.HasNext() {
		next := iterator.Next()
		if r, err := r.ApplyTransaction(next); err != nil {
			return r, err
		} else {
			receipts = r
		}
	}

	if !bytes.Equal(r.RootProof().Bytes(), block.Header().StateHash().Bytes()) {
		log.Debugf("Not equal state hash: expected %v, calculated %v", block.Header().StateHash().Hex(), r.RootProof().Hex())
		return nil, InvalidStateHashError
	}
	_, err = bc.stateDB.Commit(block.Header().Parent(), block.Header().Hash())
	if err != nil {
		return nil, err
	}

	return receipts, nil
}
