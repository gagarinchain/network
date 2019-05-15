package state

import (
	"github.com/op/go-logging"
	"github.com/pkg/errors"
	"github.com/poslibp2p/common/eth/common"
	"math/big"
	"sync"
)

var (
	Me                      common.Address
	InsufficientFundsError  = errors.New("insufficient funds")
	ExpiredTransactionError = errors.New("expired transaction")
	FutureTransactionError  = errors.New("future transaction")
	NotEmptyInitDBError     = errors.New("can't initialize not empty DB")
	log                     = logging.MustGetLogger("blockchain")
)

type DBImpl struct {
	snapshots map[common.Hash]*Snapshot
	lock      sync.RWMutex
}

func NewStateDB() DB {
	db := &DBImpl{snapshots: make(map[common.Hash]*Snapshot), lock: sync.RWMutex{}}

	db.lock.RLock()
	defer db.lock.RUnlock()

	return db
}

func (db *DBImpl) Init(hash common.Hash, seed *Snapshot) error {
	if len(db.snapshots) > 0 {
		return NotEmptyInitDBError
	}

	if seed == nil {
		seed = NewSnapshot(hash)
	}

	db.snapshots[hash] = seed
	return nil
}

func (db *DBImpl) Get(hash common.Hash) (s *Snapshot, f bool) {
	db.lock.RLock()
	defer db.lock.RUnlock()
	sn, f := db.snapshots[hash]

	return sn, f
}

func (db *DBImpl) Create(parent common.Hash) (s *Snapshot, e error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	parentSnapshot, f := db.snapshots[parent]
	if !f {
		return nil, errors.New("no prent is found")
	}

	cpy := parentSnapshot.trie.Copy()
	snapshot := &Snapshot{trie: cpy}
	parentSnapshot.siblings = append(parentSnapshot.siblings, snapshot)
	parentSnapshot.SetPending(snapshot)

	return snapshot, nil
}

func (db *DBImpl) Commit(parent, pending common.Hash) (s *Snapshot, e error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	parentSnapshot, f := db.snapshots[parent]
	if !f {
		return nil, errors.New("no prent is found")
	}

	pendingSnapshot := parentSnapshot.Pending()
	pendingSnapshot.hash = pending
	parentSnapshot.pending = nil
	db.snapshots[pending] = pendingSnapshot

	return pendingSnapshot, nil
}

func (db *DBImpl) Release(blockHash common.Hash) error {
	s, f := db.snapshots[blockHash]
	if !f {
		return errors.New("no snapshot found for block")
	}
	db.release(s)

	return nil
}

func (db *DBImpl) release(snapshot *Snapshot) {
	delete(db.snapshots, snapshot.hash)
	for _, sibl := range snapshot.siblings {
		db.release(sibl)
	}
}

type Account struct {
	nonce   uint64
	balance *big.Int
}

func NewAccount(nonce uint64, balance *big.Int) *Account {
	return &Account{nonce: nonce, balance: balance}
}

func (a *Account) Copy() *Account {
	return NewAccount(a.nonce, new(big.Int).Set(a.balance))
}
