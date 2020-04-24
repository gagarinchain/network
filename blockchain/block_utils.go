package blockchain

import (
	"github.com/gagarinchain/network/common/eth/common"
	"time"
)

func CreateBlockWithParentH(parent *Header) *Block {
	header := &Header{
		height:    parent.height + 1,
		hash:      common.Hash{},
		txHash:    common.Hash{},
		stateHash: common.Hash{},
		dataHash:  common.Hash{},
		qcHash:    common.Hash{},
		parent:    parent.Hash(),
		timestamp: time.Now(),
	}
	header.SetHash()
	return &Block{
		header:    header,
		qc:        nil,
		signature: nil,
		txs:       nil,
		data:      nil,
	}
}
func CreateBlockWithParent(parent *Block) *Block {
	header := &Header{
		height:    parent.Height() + 1,
		hash:      common.Hash{},
		txHash:    common.Hash{},
		stateHash: common.Hash{},
		dataHash:  common.Hash{},
		qcHash:    common.Hash{},
		parent:    parent.Header().Hash(),
		timestamp: time.Now(),
	}
	header.SetHash()
	return &Block{
		header:    header,
		qc:        parent.QC(),
		signature: nil,
		txs:       nil,
		data:      nil,
	}
}
