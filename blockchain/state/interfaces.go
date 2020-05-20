package state

import (
	"github.com/gagarinchain/network/common/api"
	"github.com/gagarinchain/network/common/eth/common"
)

type DB interface {
	Init(hash common.Hash, seed *Snapshot) error
	Get(hash common.Hash) (r api.Record, f bool)
	Create(parent common.Hash, proposer common.Address) (r api.Record, e error)
	Commit(parent, pending common.Hash) (r api.Record, e error)
	Release(blockHash common.Hash) error
}
