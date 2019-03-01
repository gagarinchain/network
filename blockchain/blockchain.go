package blockchain

import (
	"errors"
	"fmt"
	"github.com/emirpasic/gods/utils"
	"github.com/op/go-logging"
	"github.com/poslibp2p/eth/common"
	"sync"
)
import "github.com/emirpasic/gods/maps/treemap"

type Blockchain struct {
	indexGuard            *sync.RWMutex
	blocksIndexedByHash   map[common.Hash]*Block
	blocksIndexedByHeight *treemap.Map

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

	log.Info(common.Bytes2Hex(hash.Bytes()))

	for k := range bc.blocksIndexedByHash {
		log.Info(common.Bytes2Hex(k.Bytes()))

	}

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
func (bc *Blockchain) GetThreeChainForHead(hash common.Hash) (zero *Header, one *Header, two *Header) {

	head := bc.GetBlockByHash(hash)
	if head == nil {
		return nil, nil, nil
	}

	two = head.Header().QRef()
	if two == nil {
		return nil, nil, nil
	}

	one = two.QRef()
	if one == nil {
		return nil, nil, two
	}

	zero = one.QRef()
	if one == nil {
		return nil, one, two
	}

	return zero, one, two
}

// Returns three certified blocks (Bzero, Bone, Btwo) from 3-chain
// B|zero <-- B|one <-- B|two <--...--  B|head
func (bc *Blockchain) GetThreeChainForTwo(hash common.Hash) (zero *Header, one *Header, two *Header) {
	head := bc.GetBlockByHash(hash)
	if head == nil {
		return nil, nil, nil
	}

	two = head.header
	one = two.QRef()
	if one == nil {
		return nil, nil, two
	}

	zero = one.QRef()
	if one == nil {
		return nil, one, two
	}

	return zero, one, two
}

func (bc *Blockchain) Execute(h *Header) error {
	log.Infof("Applying transactions to state for block [%s]", h.Hash)

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
