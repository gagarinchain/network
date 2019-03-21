package blockchain

import (
	"errors"
	"fmt"
	"github.com/davecgh/go-spew/spew"
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

	synchronizer Synchronizer

	genesisCert *QuorumCertificate
}

func (bc *Blockchain) SetSynchronizer(synchronizer Synchronizer) {
	bc.synchronizer = synchronizer
}

var log = logging.MustGetLogger("blockchain")

func CreateBlockchainFromGenesisBlock() *Blockchain {
	zero, one, two, certToHead := CreateGenesisTriChain()
	blockchain := &Blockchain{
		blocksByHash:            make(map[common.Hash]*Block),
		committedTailByHeight:   treemap.NewWith(utils.Int32Comparator),
		uncommittedHeadByHeight: treemap.NewWith(utils.Int32Comparator),
		indexGuard:              &sync.RWMutex{}}

	blockchain.AddBlock(zero)
	blockchain.AddBlock(one)
	blockchain.AddBlock(two)

	blockchain.genesisCert = certToHead

	return blockchain
}

func (bc *Blockchain) GetBlockByHash(hash common.Hash) *Block {
	bc.indexGuard.RLock()
	defer bc.indexGuard.RUnlock()

	block := bc.blocksByHash[hash]

	//if block == nil {
	//	panic(fmt.Sprintf("Can't find expected block for [%hash]", hash))
	//}
	//
	return block
}

func (bc *Blockchain) GetBlockByHashOrLoad(hash common.Hash) *Block {
	if block := bc.GetBlockByHash(hash); block != nil {
		return block
	}
	block := bc.LoadBlock(hash)
	return block
}

func (bc *Blockchain) LoadBlock(hash common.Hash) *Block {
	log.Infof("Loading block with hash [%v]", hash.Hex())
	ch := bc.synchronizer.RequestBlock(hash)
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
	spew.Dump(bc.blocksByHash)

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

	_, blocks := bc.uncommittedHeadByHeight.Max()
	var b = blocks.([]*Block)
	return b[0]
}

func (bc *Blockchain) AddBlock(block *Block) error {
	log.Infof("Adding block with hash [%v]", block.header.Hash().Hex())
	//spew.Dump(block)

	bc.indexGuard.Lock()
	defer bc.indexGuard.Unlock()

	if bc.blocksByHash[block.Header().Hash()] != nil {
		return errors.New(fmt.Sprintf("block with hash [%v] already exist", block.Header().Hash()))
	}

	value, _ := bc.uncommittedHeadByHeight.Get(block.Header().Height())
	if value == nil {
		value = make([]*Block, 0)
	}

	value = append(value.([]*Block), block)
	bc.blocksByHash[block.Header().Hash()] = block
	bc.uncommittedHeadByHeight.Put(block.Header().Height(), value)

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
	return bc.genesisCert
}

func (bc Blockchain) IsSibling(sibling *Header, ancestor *Header) bool {
	//Genesis block is ancestor of every block
	if ancestor.IsGenesisBlock() {
		return true
	}

	parent := bc.GetBlockByHashOrLoad(sibling.parent)

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

func (bc *Blockchain) PadEmptyBlock() {
	block := bc.NewBlock(bc.GetHead(), bc.GetHead().QC(), []byte(""))

	if e := bc.AddBlock(block); e != nil {
		log.Error("Can't add empty block")
	}
}
