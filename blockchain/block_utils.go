package blockchain

import (
	"github.com/gagarinchain/network/common/api"
	"github.com/gagarinchain/network/common/eth/common"
	"time"
)

func CreateBlockWithParentH(parent api.Header) api.Block {
	header := &HeaderImpl{
		height:    parent.Height() + 1,
		hash:      common.Hash{},
		txHash:    common.Hash{},
		stateHash: common.Hash{},
		dataHash:  common.Hash{},
		qcHash:    common.Hash{},
		parent:    parent.Hash(),
		timestamp: time.Now(),
	}
	header.SetHash()
	return &BlockImpl{
		header:    header,
		qc:        nil,
		signature: nil,
		txs:       nil,
		data:      nil,
	}
}
func CreateBlockWithParent(parent api.Block) api.Block {
	header := &HeaderImpl{
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
	return &BlockImpl{
		header:    header,
		qc:        parent.QC(),
		signature: nil,
		txs:       nil,
		data:      nil,
	}
}
