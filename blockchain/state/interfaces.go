package state

import (
	"github.com/poslibp2p/common/eth/common"
)

type DB interface {
	Init(hash common.Hash, seed *Snapshot) error
	Get(hash common.Hash) (s *Snapshot, f bool)
	Create(parent common.Hash) (s *Snapshot, e error)
	Commit(parent, pending common.Hash) (s *Snapshot, e error)
	Release(blockHash common.Hash) error
}
