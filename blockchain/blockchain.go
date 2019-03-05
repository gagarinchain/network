package blockchain

import (
	"errors"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/emirpasic/gods/utils"
	"github.com/gogo/protobuf/proto"
	"github.com/op/go-logging"
	"github.com/poslibp2p/eth/common"
	"github.com/poslibp2p/eth/crypto"
	"github.com/poslibp2p/message/protobuff"
	"sync"
)
import "github.com/emirpasic/gods/maps/treemap"

type Blockchain struct {
	indexGuard            *sync.RWMutex
	blocksIndexedByHash   map[common.Hash]*Block
	blocksIndexedByHeight *treemap.Map

	synchronizer Synchronizer

	genesisCert *QuorumCertificate
}

var log = logging.MustGetLogger("blockchain")

func CreateBlockchainFromGenesisBlock() *Blockchain {
	zero, one, two, certToHead := CreateGenesisTriChain()
	blockchain := &Blockchain{blocksIndexedByHash: make(map[common.Hash]*Block),
		blocksIndexedByHeight: treemap.NewWith(utils.Int32Comparator), indexGuard: &sync.RWMutex{}}

	blockchain.AddBlock(zero)
	blockchain.AddBlock(one)
	blockchain.AddBlock(two)

	blockchain.genesisCert = certToHead

	return blockchain
}

func (bc *Blockchain) GetBlockByHash(hash common.Hash) *Block {
	bc.indexGuard.RLock()
	defer bc.indexGuard.RUnlock()

	block := bc.blocksIndexedByHash[hash]

	//if block == nil {
	//	panic(fmt.Sprintf("Can't find expected block for [%hash]", hash))
	//}
	//
	return block
}

func (bc *Blockchain) Contains(hash common.Hash) bool {
	return bc.GetBlockByHash(hash) != nil
}

// Returns three certified blocks (Bzero, Bone, Btwo) from 3-chain
// B|zero <-- B|one <-- B|two <--...--  B|head
func (bc *Blockchain) GetThreeChainForHead(hash common.Hash) (zero *Block, one *Block, two *Block) {

	head := bc.GetBlockByHash(hash)
	if head == nil {
		return nil, nil, nil
	}

	two = bc.GetBlockByHash(head.QRef().Hash())
	if two == nil {
		return nil, nil, nil
	}

	one = bc.GetBlockByHash(two.QRef().Hash())
	if one == nil {
		return nil, nil, two
	}

	zero = bc.GetBlockByHash(one.QRef().Hash())
	if one == nil {
		return nil, one, two
	}

	return zero, one, two
}

// Returns three certified blocks (Bzero, Bone, Btwo) from 3-chain
// B|zero <-- B|one <-- B|two <--...--  B|head
func (bc *Blockchain) GetThreeChainForTwo(twoHash common.Hash) (zero *Block, one *Block, two *Block) {
	spew.Dump(bc.blocksIndexedByHash)

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

func (bc *Blockchain) Execute(b *Block) error {
	log.Infof("Applying transactions to state for block [%s]", b.Header().Hash())

	return nil
}
func (bc *Blockchain) GetHead() *Block {
	bc.indexGuard.RLock()
	defer bc.indexGuard.RUnlock()

	//TODO find out can we have several heads by design? what fork choice rule we have
	_, block := bc.blocksIndexedByHeight.Max()
	var b = block.(*Block)
	return b
}

func (bc *Blockchain) AddBlock(block *Block) error {
	bc.indexGuard.Lock()
	defer bc.indexGuard.Unlock()

	if bc.blocksIndexedByHash[block.Header().Hash()] != nil {
		return errors.New(fmt.Sprintf("block with hash [%v] already exist", block.Header().Hash()))
	}

	bc.blocksIndexedByHeight.Put(block.Header().Height(), block)
	bc.blocksIndexedByHash[block.Header().Hash()] = block

	return nil
}

func (bc *Blockchain) GetGenesisBlock() *Block {
	bc.indexGuard.RLock()
	defer bc.indexGuard.RUnlock()

	_, value := bc.blocksIndexedByHeight.Min()

	return value.(*Block)
}

func (bc *Blockchain) GetGenesisCert() *QuorumCertificate {
	return bc.genesisCert
}

func (bc *Blockchain) Parent(hash common.Hash) *Block {
	block := bc.GetBlockByHash(hash)

	if block == nil {
		resp := make(chan *Block)
		go bc.synchronizer.RequestBlock(hash, resp)
		block = <-resp
	}
	return block
}

func (bc Blockchain) IsSibling(sibling *Header, ancestor *Header) bool {
	//Genesis block is ancestor of every block
	if ancestor.IsGenesisBlock() {
		return true
	}

	parent := bc.Parent(sibling.Hash())
	if parent.Header().Hash() == ancestor.Hash() && parent.Header().IsGenesisBlock() {
		return true
	}

	return bc.IsSibling(parent.Header(), ancestor)
}

func (bc *Blockchain) GetMessageForHeader(h *Header) (msg *pb.BlockHeader) {
	parent := bc.Parent(h.Parent())
	return &pb.BlockHeader{ParentHash: parent.Header().Hash().Bytes(), DataHash: h.Hash().Bytes(), Height: h.Height(), Timestamp: h.Timestamp().Unix()}
}

func (bc *Blockchain) GetMessageForBlock(b *Block) (msg *pb.Block) {
	pdata := &pb.BlockData{Data: b.Data()}
	qc := b.QC()
	var pqrefHeader = bc.GetMessageForHeader(qc.QrefBlock())
	pcert := &pb.QuorumCertificate{Header: pqrefHeader, SignatureAggregate: qc.SignatureAggregate()}

	return &pb.Block{Header: bc.GetMessageForHeader(b.Header()), Data: pdata, Cert: pcert}
}

func (bc *Blockchain) SerializeHeader(h *Header) []byte {
	msg := bc.GetMessageForHeader(h)

	bytes, e := proto.Marshal(msg)
	if e != nil {
		log.Error("Can't marshall message", e)
	}

	return bytes
}

func (bc *Blockchain) SerializeBlock(b *Block) []byte {
	msg := bc.GetMessageForBlock(b)

	bytes, e := proto.Marshal(msg)
	if e != nil {
		log.Error("Can't marshall message", e)
	}

	return bytes
}

func (bc *Blockchain) CreateBlockAndSetHash(header *Header, data []byte) *Block {
	b := &Block{header: header, data: data}
	bc.SetHash(b)
	return b
}

func (bc *Blockchain) SetHash(b *Block) {
	msg := bc.GetMessageForBlock(b)
	bytes, e := proto.Marshal(msg)
	if e == nil {
		b.header.hash = crypto.Keccak256Hash(bytes)
	}
}

func (bc *Blockchain) NewBlock(parent *Block, qc *QuorumCertificate, data []byte) *Block {
	hash := common.BytesToHash([]byte(""))
	header := &Header{height: parent.Header().Height() + 1, hash: hash, parent: parent.Header().Hash()}
	block := &Block{header: header, data: data, qc: qc}

	bc.SetHash(block)
	return block
}
