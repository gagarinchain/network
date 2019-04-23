package blockchain

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"github.com/emirpasic/gods/utils"
	"github.com/op/go-logging"
	"github.com/poslibp2p/common/eth/common"
	"github.com/poslibp2p/common/eth/crypto"
	"sync"
	"time"
)
import "github.com/emirpasic/gods/maps/treemap"

type Blockchain struct {
	indexGuard *sync.RWMutex
	//index for storing blocks by their hash
	blocksByHash map[common.Hash]*Block

	//indexes for storing blocks according to their height,
	// <int32, *Block>
	committedTailByHeight *treemap.Map
	//indexes for storing block arrays according to their height, while not committed head may contain forks,
	//<int32, []*Block>
	uncommittedHeadByHeight *treemap.Map
	storage                 Storage
	blockService            BlockService
}

func (bc *Blockchain) BlockService(blockService BlockService) {
	bc.blockService = blockService
}

func (bc *Blockchain) SetStorage(storage Storage) {
	bc.storage = storage
}

var log = logging.MustGetLogger("blockchain")

func CreateBlockchainFromGenesisBlock(storage Storage, blockService BlockService) *Blockchain {
	zero := CreateGenesisBlock()
	blockchain := &Blockchain{
		blocksByHash:            make(map[common.Hash]*Block),
		committedTailByHeight:   treemap.NewWith(utils.Int32Comparator),
		uncommittedHeadByHeight: treemap.NewWith(utils.Int32Comparator),
		indexGuard:              &sync.RWMutex{},
		storage:                 storage,
		blockService:            blockService,
	}

	if err := blockchain.AddBlock(zero); err != nil {
		panic("can't add genesis block")
	}

	return blockchain
}

func CreateBlockchainFromStorage(storage Storage, blockService BlockService) *Blockchain {
	topHeight, err := storage.GetCurrentTopHeight()
	if err != nil || topHeight < 0 {
		return CreateBlockchainFromGenesisBlock(storage, blockService)
	}

	blockchain := &Blockchain{
		blocksByHash:            make(map[common.Hash]*Block),
		committedTailByHeight:   treemap.NewWith(utils.Int32Comparator),
		uncommittedHeadByHeight: treemap.NewWith(utils.Int32Comparator),
		indexGuard:              &sync.RWMutex{},
		storage:                 storage,
		blockService:            blockService,
	}

	topCommittedHeight, err := storage.GetTopCommittedHeight()
	blockchain.indexGuard.RLock()
	defer blockchain.indexGuard.RUnlock()

	//TODO this place should be heavily optimized, i think we should preload everything except transactions for committed part of bc, headers and signatures will take 1k per block amortised
	for i := topHeight; i >= 0; i-- {
		hashes, err := storage.GetHeightIndexRecord(i)
		if err != nil {
			log.Fatal(err)
		}
		for _, h := range hashes {
			b, err := storage.GetBlock(h)
			if err != nil {
				log.Fatal(err)
			}
			blockchain.blocksByHash[h] = b

			if i > topCommittedHeight {
				if err := blockchain.addUncommittedBlock(b); err != nil {
					log.Fatal(err)
				}
			} else {
				blockchain.committedTailByHeight.Put(b.Height(), b)
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
		block, er := bc.storage.GetBlock(hash)
		if er != nil {
			log.Error(er)
		}
		return block
	}

	return block
}

func (bc *Blockchain) GetBlockByHeight(height int32) (res []*Block) {
	block, ok := bc.committedTailByHeight.Get(height)
	if !ok {
		blocks, ok := bc.uncommittedHeadByHeight.Get(height)
		if ok {
			res = append(res, blocks.([]*Block)...)
		} else { //TODO it seems like a hack, we should preload blockchain structure from index to memory
			hashes, err := bc.storage.GetHeightIndexRecord(height)
			if err == nil {
				for _, hash := range hashes {
					block, err := bc.storage.GetBlock(hash)
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

func (bc *Blockchain) GetBlockByHashOrLoad(hash common.Hash) (b *Block, loaded bool) {
	loaded = !bc.Contains(hash)
	if loaded {
		b = bc.LoadBlock(hash)
	} else {
		b = bc.GetBlockByHash(hash)
	}

	return b, loaded
}

//todo pass loading errors higher
func (bc *Blockchain) LoadBlock(hash common.Hash) *Block {
	log.Infof("Loading block with hash [%v]", hash.Hex())
	blocks, err := ReadBlocksWithErrors(bc.blockService.RequestBlock(context.Background(), hash, nil))
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
func (bc *Blockchain) OnCommit(b *Block) {
	bc.indexGuard.RLock()
	uncommittedTail := bc.uncommittedHeadByHeight.Select(func(key interface{}, value interface{}) bool {
		return key.(int32) <= b.Header().height
	})
	bc.indexGuard.RUnlock()

	for i := 0; i < uncommittedTail.Size(); i++ {
		bc.indexGuard.Lock()
		bc.committedTailByHeight.Put(b.Header().Height(), b)
		bc.uncommittedHeadByHeight.Remove(b.Header().Height())
		bc.indexGuard.Unlock()

		err := bc.storage.PutTopCommittedHeight(b.Height())
		if err != nil {
			log.Error(err)
		}
		b = bc.GetBlockByHash(b.Header().Hash())
	}

	analyze(uncommittedTail)
}

//TODO analyze forks and find where peers equivocated, than possibly slash them
func analyze(uncommittedTail *treemap.Map) {
	log.Infof("Analyzing")
}

func (bc *Blockchain) Execute(b *Block) error {

	log.Infof("Applying transactions to state for block [%s]", b.Header().Hash())

	return nil
}
func (bc *Blockchain) GetHead() *Block {
	bc.indexGuard.RLock()
	defer bc.indexGuard.RUnlock()

	//TODO probably it is not correct^ have to return head of PREF block branch here
	_, blocks := bc.uncommittedHeadByHeight.Max()
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

	if bc.blocksByHash[block.Header().Hash()] != nil || bc.storage.Contains(block.Header().Hash()) {
		log.Debugf("Block with hash [%v] already exists, updating", block.header.Hash().Hex())
		return bc.storage.PutBlock(block)
	}

	if bc.blocksByHash[block.Header().Parent()] == nil && !bc.storage.Contains(block.Header().Parent()) && block.Height() != 0 {
		return errors.New(fmt.Sprintf("block with hash [%v] don't have parent loaded to blockchain", block.Header().Hash().Hex()))
	}

	value, _ := bc.uncommittedHeadByHeight.Get(block.Header().Height())
	if value == nil {
		if err := bc.storage.PutCurrentTopHeight(block.Header().Height()); err != nil {
			return err
		}
	}

	if err := bc.addUncommittedBlock(block); err != nil {
		return err
	}

	if err := bc.storage.PutBlock(block); err != nil {
		return err
	}

	return nil
}

func (bc *Blockchain) addUncommittedBlock(block *Block) error {
	value, _ := bc.uncommittedHeadByHeight.Get(block.Header().Height())
	if value == nil {
		value = make([]*Block, 0)
	}

	value = append(value.([]*Block), block)
	bc.blocksByHash[block.Header().Hash()] = block
	bc.uncommittedHeadByHeight.Put(block.Header().Height(), value)

	return nil
}

func (bc *Blockchain) RemoveBlock(block *Block) error {
	log.Infof("Removing block with hash [%v]", block.header.Hash().Hex())
	//spew.Dump(block)

	bc.indexGuard.Lock()
	defer bc.indexGuard.Unlock()

	if bc.blocksByHash[block.Header().Hash()] == nil && !bc.storage.Contains(block.Header().Hash()) {
		log.Warningf("Block with hash [%v] is absent", block.header.Hash().Hex())
		return nil
	}

	//we can delete only leafs, so check whether this block is parent of any
	value, found := bc.uncommittedHeadByHeight.Get(block.Height() + 1)
	if found {
		nextHeight := value.([]*Block)
		for _, next := range nextHeight {
			if block.Header().Parent().Big() == next.Header().Hash().Big() {
				return errors.New(fmt.Sprintf("block [%v] has sibling [%v] on next level", block.header.Hash().Hex(), next.Header().Hash().Hex()))
			}
		}
	}

	delete(bc.blocksByHash, block.Header().Hash())
	bc.uncommittedHeadByHeight.Remove(block.Header().Hash())

	return nil
}

func (bc *Blockchain) GetGenesisBlock() *Block {
	bc.indexGuard.RLock()
	defer bc.indexGuard.RUnlock()

	value, ok := bc.committedTailByHeight.Get(int32(0))
	if !ok {
		if h, okk := bc.uncommittedHeadByHeight.Get(int32(0)); okk {
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

	//todo should load blocks earlier
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
	header := createHeader(parent.Header().Height()+1, common.Hash{}, qc.GetHash(), crypto.Keccak256Hash(data),
		parent.Header().Hash(), time.Now().UTC().Round(time.Second))
	header.SetHash()
	block := &Block{header: header, data: data, qc: qc}

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
	_, block := bc.committedTailByHeight.Max()
	//todo think about committing genesis at making QC
	if block == nil {
		return bc.GetGenesisBlock()
	}
	return block.(*Block)
}

func (bc *Blockchain) UpdateGenesisBlockQC(certificate *QuorumCertificate) {
	bc.GetGenesisBlock().qc = certificate
	//we can simply put new block and replace existing. in fact ignoring height index is not a problem for us since Genesis is hashed without it's QC
	if err := bc.storage.PutBlock(bc.GetGenesisBlock()); err != nil {
		log.Error(err)
		return
	}

}
