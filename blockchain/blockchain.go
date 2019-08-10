package blockchain

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/emirpasic/gods/utils"
	net "github.com/gagarinchain/network"
	"github.com/gagarinchain/network/blockchain/state"
	cmn "github.com/gagarinchain/network/common"
	"github.com/gagarinchain/network/common/eth/common"
	"github.com/gagarinchain/network/common/eth/crypto"
	"github.com/gagarinchain/network/common/trie"
	"github.com/gagarinchain/network/common/tx"
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
	Storage net.Storage
}

func (bp *BlockchainPersister) PutCurrentTopHeight(currentTopHeight int32) error {
	return bp.Storage.Put(net.CurrentTopHeight, nil, cmn.Int32ToByte(currentTopHeight))
}

func (bp *BlockchainPersister) GetCurrentTopHeight() (val int32, err error) {
	value, err := bp.Storage.Get(net.CurrentTopHeight, nil)
	if err != nil {
		return cmn.DefaultIntValue, nil
	}
	return cmn.ByteToInt32(value)
}

func (bp *BlockchainPersister) PutTopCommittedHeight(currentTopHeight int32) error {
	return bp.Storage.Put(net.TopCommittedHeight, nil, cmn.Int32ToByte(currentTopHeight))
}

func (bp *BlockchainPersister) GetTopCommittedHeight() (val int32, err error) {
	value, err := bp.Storage.Get(net.TopCommittedHeight, nil)
	if err != nil {
		return cmn.DefaultIntValue, nil
	}
	return cmn.ByteToInt32(value)
}

func (bp *BlockchainPersister) PutHeightIndexRecord(b *Block) error {
	key := make([]byte, binary.MaxVarintLen64)
	binary.PutVarint(key, int64(b.Height()))

	found := bp.Storage.Contains(net.HeightIndex, key)
	var indexValue []byte

	if found {
		value, err := bp.Storage.Get(net.HeightIndex, key)
		if err != nil {
			return err
		}
		indexValue = value
	}

	indexValue = append(indexValue, b.Header().Hash().Bytes()...)
	return bp.Storage.Put(net.HeightIndex, key, indexValue)
}

func (bp *BlockchainPersister) GetHeightIndexRecord(height int32) (hashes []common.Hash, err error) {
	key := make([]byte, binary.MaxVarintLen64)
	binary.PutVarint(key, int64(height))

	val, err := bp.Storage.Get(net.HeightIndex, key)
	if err != nil {
		return nil, err
	}

	hashes = make([]common.Hash, len(val)/common.HashLength)

	for i := 0; i < len(val)/common.HashLength; i++ {
		hashes[i] = common.BytesToHash(val[i*common.HashLength : (i+1)*common.HashLength])
	}

	return hashes, nil
}

type Blockchain struct {
	indexGuard *sync.RWMutex
	//index for storing blocks by their hash
	blocksByHash map[common.Hash]*Block
	//indexes for storing blocks according to their height,
	// <int32, *Block>
	committedChainByHeight *treemap.Map
	//indexes for storing block arrays according to their height, while not committed head may contain forks,
	//<int32, []*Block>
	uncommittedTreeByHeight *treemap.Map
	blockService            BlockService
	txPool                  TransactionPool
	stateDB                 state.DB
	blockPersister          *BlockPersister
	chainPersister          *BlockchainPersister
}

func (bc *Blockchain) BlockService(blockService BlockService) {
	bc.blockService = blockService
}

var log = logging.MustGetLogger("blockchain")

func CreateBlockchainFromGenesisBlock(cfg *BlockchainConfig) *Blockchain {
	zero := CreateGenesisBlock()
	blockchain := &Blockchain{
		blocksByHash:            make(map[common.Hash]*Block),
		committedChainByHeight:  treemap.NewWith(utils.Int32Comparator),
		uncommittedTreeByHeight: treemap.NewWith(utils.Int32Comparator),
		indexGuard:              &sync.RWMutex{},
		blockService:            cfg.BlockService,
		txPool:                  cfg.Pool,
		stateDB:                 cfg.Db,
		blockPersister:          cfg.BlockPerister,
		chainPersister:          cfg.ChainPersister,
	}

	var s *state.Snapshot
	if cfg.Seed != nil {
		s = state.NewSnapshotWithAccounts(zero.Header().Hash(), cfg.Seed)
	}
	if e := cfg.Db.Init(zero.Header().Hash(), s); e != nil {
		panic("can't init state DB")
	}
	if err := blockchain.AddBlock(zero); err != nil {
		log.Error(err)
		panic("can't add genesis block")
	}
	return blockchain
}

func CreateBlockchainFromStorage(storage net.Storage, blockService BlockService, pool TransactionPool, db state.DB) *Blockchain {
	pblock := &BlockPersister{Storage: storage}
	pchain := &BlockchainPersister{Storage: storage}

	topHeight, err := pchain.GetCurrentTopHeight()
	log.Debugf("loaded topheight %v", topHeight)

	if err != nil || topHeight < 0 {
		return CreateBlockchainFromGenesisBlock(&BlockchainConfig{
			Seed:           nil,
			ChainPersister: pchain,
			BlockPerister:  pblock,
			BlockService:   blockService,
			Pool:           pool,
			Db:             db,
		})
	}

	blockchain := &Blockchain{
		blocksByHash:            make(map[common.Hash]*Block),
		committedChainByHeight:  treemap.NewWith(utils.Int32Comparator),
		uncommittedTreeByHeight: treemap.NewWith(utils.Int32Comparator),
		indexGuard:              &sync.RWMutex{},
		blockService:            blockService,
		txPool:                  pool,
		stateDB:                 db,
		blockPersister:          pblock,
		chainPersister:          pchain,
	}

	topCommittedHeight, err := pchain.GetTopCommittedHeight()
	blockchain.indexGuard.RLock()
	defer blockchain.indexGuard.RUnlock()

	//TODO this place should be heavily optimized, i think we should preload everything except transactions for committed part of bc, headers and signatures will take 1k per block amortised
	for i := topHeight; i >= 0; i-- {
		hashes, err := pchain.GetHeightIndexRecord(i)
		if err != nil {
			log.Fatal(err)
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

func (bc *Blockchain) GetBlockByHash(hash common.Hash) (block *Block) {
	bc.indexGuard.RLock()
	defer bc.indexGuard.RUnlock()

	block = bc.blocksByHash[hash]

	if block == nil {
		block, er := bc.blockPersister.Load(hash)
		if er != nil {
			log.Error(er)
		}
		return block
	}

	return block
}

func (bc *Blockchain) GetBlockByHeight(height int32) (res []*Block) {
	block, ok := bc.committedChainByHeight.Get(height)
	if !ok {
		blocks, ok := bc.uncommittedTreeByHeight.Get(height)
		if ok {
			res = append(res, blocks.([]*Block)...)
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
		res = append(res, block.(*Block))
	}
	return res
}

func (bc *Blockchain) GetFork(height int32, headHash common.Hash) (res []*Block) {
	head := bc.blocksByHash[headHash]
	hash := headHash
	res = make([]*Block, head.Height()-height+1)
	for i := 0; i < len(res); i++ {
		head = bc.blocksByHash[hash]
		res[i] = head
		hash = head.Header().Parent()
	}

	return res
}

//todo remove this method
func (bc *Blockchain) GetBlockByHashOrLoad(ctx context.Context, hash common.Hash) (b *Block, loaded bool) {
	loaded = !bc.Contains(hash)
	if loaded {
		b = bc.LoadBlock(ctx, hash)
	} else {
		b = bc.GetBlockByHash(hash)
	}

	return b, loaded
}

func (bc *Blockchain) LoadBlock(ctx context.Context, hash common.Hash) *Block {
	log.Infof("Loading block with hash [%v]", hash.Hex())
	blocks, err := ReadBlocksWithErrors(bc.blockService.RequestBlock(ctx, hash, nil))
	if err != nil {
		log.Error("Can't load block", err)
		return nil
	}
	if err := bc.AddBlock(blocks[0]); err != nil {
		log.Error("Can't add loaded block", err)
		return nil
	}
	return blocks[0]
}

func (bc *Blockchain) Contains(hash common.Hash) bool {
	return bc.GetBlockByHash(hash) != nil
}

// Returns three certified blocks (Bzero, Bone, Btwo) from 3-chain
// B|zero <-- B|one <-- B|two <--...--  B|head
func (bc *Blockchain) GetThreeChain(twoHash common.Hash) (zero *Block, one *Block, two *Block) {
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
func (bc *Blockchain) OnCommit(b *Block) (toCommit []*Block, orphans *treemap.Map, err error) {
	orphans = treemap.NewWith(utils.Int32Comparator)

	low, _ := bc.uncommittedTreeByHeight.Min()
	if low == nil {
		return nil, nil, errors.New("empty commit index")
	}

	//lets collect all predecessors of block to commit, and make orphans map
	phash := b.header.hash
	for height := b.Height(); height >= low.(int32); height-- {
		val, found := bc.uncommittedTreeByHeight.Get(height)
		currentBag := val.([]*Block)
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
	hashes := map[common.Hash]bool{b.header.hash: true}
	max, _ := bc.uncommittedTreeByHeight.Max()
	for height := b.Height() + 1; height <= max.(int32); height++ {
		val, found := bc.uncommittedTreeByHeight.Get(height)
		currentBag := val.([]*Block)
		if !found {
			return nil, nil, errors.New("bad index structure")
		}
		var o []*Block
		var c []*Block
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

	//remove committed transactions from the pool
	for _, b := range toCommit {
		it := b.Txs()
		for next := it.Next(); next != nil; next = it.Next() {
			bc.txPool.Remove(next)
		}
	}

	//release orphan states
	_, v := orphans.Min()
	lowestHeightOrphans := v.([]*Block)
	for _, o := range lowestHeightOrphans {
		if e := bc.stateDB.Release(o.Header().hash); e != nil {
			log.Error(e)
		}
	}

	return toCommit, orphans, nil
}

func (bc *Blockchain) Execute(b *Block) error {

	log.Infof("Applying transactions to state for block [%s]", b.Header().Hash())

	return nil
}
func (bc *Blockchain) GetHead() *Block {
	bc.indexGuard.RLock()
	defer bc.indexGuard.RUnlock()

	//TODO probably it is not correct^ have to return head of PREF block branch here
	_, blocks := bc.uncommittedTreeByHeight.Max()
	var b = blocks.([]*Block)
	return b[0]
}

func (bc *Blockchain) GetTopHeight() int32 {
	return bc.GetHead().Header().Height()
}

func (bc *Blockchain) AddBlock(block *Block) error {
	log.Infof("Adding block with hash [%v]", block.header.Hash().Hex())
	//spew.Dump(block)

	bc.indexGuard.Lock()
	defer bc.indexGuard.Unlock()

	if bc.blocksByHash[block.Header().Hash()] != nil || bc.blockPersister.Contains(block.Header().Hash()) {
		log.Debugf("Block with hash [%v] already exists, updating", block.header.Hash().Hex())
		return bc.blockPersister.Persist(block)
	}

	if bc.blocksByHash[block.Header().Parent()] == nil && !bc.blockPersister.Contains(block.Header().Parent()) && block.Height() != 0 {
		return errors.New(fmt.Sprintf("block with hash [%v] don't have parent loaded to blockchain", block.Header().Hash().Hex()))
	}

	//todo remove this hack by handling genesis in special way
	if block.Height() != 0 {
		if err := bc.applyTransactionsAndValidateProof(block); err != nil {
			return err
		}
	}

	value, _ := bc.uncommittedTreeByHeight.Get(block.Header().Height())
	if value == nil {
		if err := bc.chainPersister.PutCurrentTopHeight(block.Header().Height()); err != nil {
			return err
		}
	}

	if err := bc.addUncommittedBlock(block); err != nil {
		return err
	}

	if err := bc.blockPersister.Persist(block); err != nil {
		return err
	}
	if err := bc.chainPersister.PutHeightIndexRecord(block); err != nil {
		return err
	}

	return nil
}

func (bc *Blockchain) addUncommittedBlock(block *Block) error {
	value, _ := bc.uncommittedTreeByHeight.Get(block.Header().Height())
	if value == nil {
		value = make([]*Block, 0)
	}

	value = append(value.([]*Block), block)
	bc.blocksByHash[block.Header().Hash()] = block
	bc.uncommittedTreeByHeight.Put(block.Header().Height(), value)

	return nil
}

func (bc *Blockchain) RemoveBlock(block *Block) error {
	log.Infof("Removing block with hash [%v]", block.header.Hash().Hex())
	//spew.Dump(block)

	bc.indexGuard.Lock()
	defer bc.indexGuard.Unlock()

	if bc.blocksByHash[block.Header().Hash()] == nil && !bc.blockPersister.Contains(block.Header().Hash()) {
		log.Warningf("Block with hash [%v] is absent", block.header.Hash().Hex())
		return nil
	}

	//we can delete only leafs, so check whether this block is not parent of any
	value, found := bc.uncommittedTreeByHeight.Get(block.Height() + 1)
	if found {
		nextHeight := value.([]*Block)
		for _, next := range nextHeight {
			if block.Header().Parent().Big() == next.Header().Hash().Big() {
				return errors.New(fmt.Sprintf("block [%v] has sibling [%v] on next level", block.header.Hash().Hex(), next.Header().Hash().Hex()))
			}
		}
	}

	delete(bc.blocksByHash, block.Header().Hash())
	bc.uncommittedTreeByHeight.Remove(block.Header().Hash())

	return nil
}

func (bc *Blockchain) GetGenesisBlock() *Block {
	bc.indexGuard.RLock()
	defer bc.indexGuard.RUnlock()

	return bc.getGenesisBlock()
}

func (bc *Blockchain) getGenesisBlock() *Block {
	value, ok := bc.committedChainByHeight.Get(int32(0))
	if !ok {
		if h, okk := bc.uncommittedTreeByHeight.Get(int32(0)); okk {
			value = h.([]*Block)[0]
		}
	}

	return value.(*Block)
}

func (bc *Blockchain) GetGenesisCert() *QuorumCertificate {
	return bc.GetGenesisBlock().QC()
}

func (bc Blockchain) IsSibling(sibling *Header, ancestor *Header) bool {
	//Genesis block is ancestor of every block
	if ancestor.IsGenesisBlock() {
		return true
	}

	//todo should load blocks earlier, remove this call
	parent := bc.GetBlockByHash(sibling.parent)

	if parent.Header().IsGenesisBlock() || parent.header.height < ancestor.height {
		return false
	}

	if parent.Header().Hash() == ancestor.Hash() {
		return true
	}

	return bc.IsSibling(parent.Header(), ancestor)
}

func (bc *Blockchain) NewBlock(parent *Block, qc *QuorumCertificate, data []byte) *Block {
	s, e := bc.stateDB.Create(parent.Header().Hash())
	if e != nil {
		log.Error("Can't create new block", e)
		return nil
	}

	it := bc.txPool.Iterator()
	txs := trie.New()
	var txs_arr []*tx.Transaction
	//todo make optimizer, to collect transactions due to fee, size, single account, etc
	for next, count := it.Next(), 0; next != nil && count < TxLimit; next, count = it.Next(), count+1 {
		err := s.ApplyTransaction(next)
		switch err {
		case state.FutureTransactionError:
			log.Error(err)
			//ignore this transaction
			continue
		case state.InsufficientFundsError:
			log.Error(err)
			//ignore this transaction
			continue
		case state.ExpiredTransactionError: //this is pretty normal to see stale transactions in the pool, possibly blocks containing this txs were not yet committed
			log.Debug(err)
			continue
		}
		txs.InsertOrUpdate([]byte(next.HashKey().Hex()), next.Serialized())
		txs_arr = append(txs_arr, next)
		spew.Dump(next)
	}

	header := createHeader(
		parent.Header().Height()+1,
		common.Hash{},
		qc.GetHash(),
		txs.Proof(),
		s.Proof(),
		crypto.Keccak256Hash(data),
		parent.Header().Hash(),
		time.Now().UTC().Round(time.Second))
	header.SetHash()

	_, err := bc.stateDB.Commit(parent.Header().Hash(), header.Hash())
	//bc.txPool.RemoveAll(txs_arr...)

	if err != nil {
		log.Error("Can't create new block", err)
		return nil
	}

	block := &Block{header: header, data: data, qc: qc, txs: txs}

	return block
}

func (bc *Blockchain) PadEmptyBlock(head *Block) *Block {
	block := bc.NewBlock(head, head.QC(), []byte(""))

	if e := bc.AddBlock(block); e != nil {
		log.Error("Can't add empty block")
		return nil
	}

	return block
}

func (bc *Blockchain) GetGenesisBlockSignedHash(key *ecdsa.PrivateKey) []byte {
	hash := bc.GetGenesisBlock().Header().Hash()
	sig, e := crypto.Sign(hash.Bytes(), key)
	if e != nil {
		log.Fatal("Can't sign genesis block")
	}
	return sig

}
func (bc *Blockchain) ValidateGenesisBlockSignature(signature []byte, address common.Address) bool {
	hash := bc.GetGenesisBlock().Header().Hash()
	pub, e := crypto.SigToPub(hash.Bytes(), signature)
	if e != nil {
		log.Error("bad epoch signature")
		return false
	}
	a := crypto.PubkeyToAddress(*pub)

	return a == address
}

func (bc *Blockchain) GetTopCommittedBlock() *Block {
	bc.indexGuard.RLock()
	defer bc.indexGuard.RUnlock()
	_, block := bc.committedChainByHeight.Max()
	//todo think about committing genesis at making QC
	if block == nil {
		return bc.GetGenesisBlock()
	}
	return block.(*Block)
}

func (bc *Blockchain) UpdateGenesisBlockQC(certificate *QuorumCertificate) {
	bc.GetGenesisBlock().qc = certificate
	//we can simply put new block and replace existing. in fact ignoring height index is not a problem for us since Genesis is hashed without it's QC
	if err := bc.blockPersister.Persist(bc.GetGenesisBlock()); err != nil {
		log.Error(err)
		return
	}
}

func (bc *Blockchain) applyTransactionsAndValidateProof(block *Block) error {
	s, e := bc.stateDB.Create(block.Header().Parent())
	if e != nil {
		return e
	}

	iterator := block.Txs()
	for next := iterator.Next(); next != nil; next = iterator.Next() {
		if err := s.ApplyTransaction(next); err != nil {
			return err
		}
	}

	if !bytes.Equal(s.Proof().Bytes(), block.Header().stateHash.Bytes()) {
		log.Debugf("Not equal state hash: expected %v, calculated %v", s.Proof().Hex(), block.Header().stateHash.Hex())
		return InvalidStateHashError
	}
	_, err := bc.stateDB.Commit(block.Header().Parent(), block.Header().Hash())
	if err != nil {
		return err
	}

	return nil
}
