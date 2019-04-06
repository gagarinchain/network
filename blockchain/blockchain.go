package blockchain

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"github.com/emirpasic/gods/utils"
	"github.com/op/go-logging"
	"github.com/poslibp2p/eth/common"
	"github.com/poslibp2p/eth/crypto"
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

func (bc *Blockchain) GetBlockByHashOrLoad(hash common.Hash) (b *Block, loaded bool) {
	loaded = !bc.Contains(hash)
	if loaded {
		b = bc.LoadBlock(hash)
	} else {
		b = bc.GetBlockByHash(hash)
	}

	return b, loaded
}

func (bc *Blockchain) LoadBlock(hash common.Hash) *Block {
	log.Infof("Loading block with hash [%v]", hash.Hex())
	ch := bc.blockService.RequestBlock(context.Background(), hash)
	block := <-ch
	if err := bc.AddBlock(block); err != nil {
		log.Error("Can't add loaded block", err)
	}
	return block
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
		return errors.New(fmt.Sprintf("block with hash [%v] already exist", block.Header().Hash().Hex()))
	}

	value, _ := bc.uncommittedHeadByHeight.Get(block.Header().Height())
	if value == nil {
		value = make([]*Block, 0)
		if err := bc.storage.PutCurrentTopHeight(block.Header().Height()); err != nil {
			return err
		}
	}

	value = append(value.([]*Block), block)
	bc.blocksByHash[block.Header().Hash()] = block

	bc.uncommittedHeadByHeight.Put(block.Header().Height(), value)
	if err := bc.storage.PutBlock(block); err != nil {
		return err
	}

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

	parent, _ := bc.GetBlockByHashOrLoad(sibling.parent)

	if parent.Header().IsGenesisBlock() || parent.header.height < ancestor.height {
		return false
	}

	if parent.Header().Hash() == ancestor.Hash() {
		return true
	}

	return bc.IsSibling(parent.Header(), ancestor)
}

//func (bc *Blockchain) GetMessageForHeader(h *Header) (msg *pb.BlockHeader) {
//	parent := bc.GetBlockByHashOrLoad(h.Parent())
//	return &pb.BlockHeader{ParentHash: parent.Header().Hash().Bytes(), DataHash: h.Hash().Bytes(), Height: h.Height(), Timestamp: h.Timestamp().Unix()}
//}
//
//func (bc *Blockchain) GetMessageForBlock(b *Block) (msg *pb.Block) {
//	pdata := &pb.BlockData{Data: b.Data()}
//	qc := b.QC()
//	var pqrefHeader = bc.GetMessageForHeader(qc.QrefBlock())
//	pcert := &pb.QuorumCertificate{Header: pqrefHeader, SignatureAggregate: qc.SignatureAggregate()}
//
//	return &pb.Block{Header: bc.GetMessageForHeader(b.Header()), Data: pdata, Cert: pcert}
//}
//
//func (bc *Blockchain) SerializeHeader(h *Header) []byte {
//	msg := bc.GetMessageForHeader(h)
//
//	bytes, e := proto.Marshal(msg)
//	if e != nil {
//		log.Error("Can't marshall message", e)
//	}
//
//	return bytes
//}

//func (bc *Blockchain) SerializeBlock(b *Block) []byte {
//	msg := bc.GetMessageForBlock(b)
//
//	bytes, e := proto.Marshal(msg)
//	if e != nil {
//		log.Error("Can't marshall message", e)
//	}
//
//	return bytes
//}

func (bc *Blockchain) NewBlock(parent *Block, qc *QuorumCertificate, data []byte) *Block {
	header := createHeader(parent.Header().Height()+1, common.Hash{}, qc.GetHash(), crypto.Keccak256Hash(data),
		parent.Header().Hash(), time.Now().Round(time.Second))
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

func (bc *Blockchain) GetBlockByHeight(height int32) (res []*Block) {
	block, ok := bc.committedTailByHeight.Get(height)
	if !ok {
		blocks, ok := bc.uncommittedHeadByHeight.Get(height)
		if ok {
			res = append(res, blocks.([]*Block)...)
		}
	} else {
		res = append(res, block.(*Block))
	}
	return res
}

func (bc *Blockchain) GetGenesisBlockSignedHash(key *ecdsa.PrivateKey) []byte {
	sig, e := crypto.Sign(bc.GetGenesisBlock().Header().Hash().Bytes(), key)
	if e != nil {
		log.Fatal("Can't sign genesis block")
	}
	return sig

}
func (bc *Blockchain) ValidateGenesisBlockSignature(signature []byte, address common.Address) bool {
	pub, e := crypto.SigToPub(bc.GetGenesisBlock().Header().Hash().Bytes(), signature)
	if e != nil {
		log.Error("bad epoch signature")
		return false
	}
	a := common.BytesToAddress(crypto.FromECDSAPub(pub))

	return a == address
}

func (bc *Blockchain) SetGenesisCertificate(signature []byte) {
	bc.GetGenesisBlock().qc = CreateQuorumCertificate(signature, bc.GetGenesisBlock().Header())
}
