package state

import (
	"github.com/gagarinchain/network"
	"github.com/gagarinchain/network/common/eth/common"
	"github.com/gagarinchain/network/common/protobuff"
	"github.com/gogo/protobuf/proto"
	"github.com/op/go-logging"
	"github.com/pkg/errors"
	"math/big"
	"sync"
)

var (
	Me                      common.Address
	InsufficientFundsError  = errors.New("insufficient funds")
	ExpiredTransactionError = errors.New("expired transaction")
	FutureTransactionError  = errors.New("future transaction")
	WrongProofOrigin        = errors.New("wrong proof origin")
	NotEmptyInitDBError     = errors.New("can't initialize not empty DB")
	log                     = logging.MustGetLogger("state")
)

type DBImpl struct {
	snapshots map[common.Hash]*Snapshot
	persister *SnapshotPersister
	lock      sync.RWMutex
}

func NewStateDB(storage gagarinchain.Storage) DB {
	persister := &SnapshotPersister{storage: storage}
	snapshots := make(map[common.Hash]*Snapshot)
	db := &DBImpl{snapshots: snapshots, persister: persister, lock: sync.RWMutex{}}

	db.lock.RLock()
	defer db.lock.RUnlock()

	hashes := persister.Hashes()
	messages := make(map[common.Hash]*pb.Snapshot)
	for _, hash := range hashes {
		value, e := persister.Get(hash)
		if e != nil {
			log.Errorf("Can't find snapshot with hash %v", value)
			continue
		}
		m := &pb.Snapshot{}
		if err := proto.Unmarshal(value, m); err != nil {
			log.Errorf("Can't unmarshal snapshot with hash %v", value)
			continue
		}

		messages[hash] = m
	}

	for k := range messages {
		_, e := createSnapshot(k, messages, snapshots)
		if e != nil {
			log.Error("Can't parse snapshots", e)
			return db
		}
	}
	return db
}

func createSnapshot(key common.Hash, msgs map[common.Hash]*pb.Snapshot, snaps map[common.Hash]*Snapshot) (s *Snapshot, e error) {
	msg := msgs[key]

	s, e = FromProtoWithoutSiblings(msg)
	if e != nil {
		log.Error("Can't deserialize snapshot", e)
		return
	}
	snaps[key] = s

	for _, sibl := range msg.Siblings {
		siblHash := common.BytesToHash(sibl)
		sibling, f := snaps[siblHash]
		if !f { //not found parsed subtree, go on parsing sibling
			sibling, e = createSnapshot(siblHash, msgs, snaps)
			if e != nil {
				log.Error("Can't create sibling", e)
				continue
			}
		}
		s.siblings = append(s.siblings, sibling)
	}
	delete(msgs, key)

	return
}

func (db *DBImpl) Init(hash common.Hash, seed *Snapshot) error {
	if len(db.snapshots) > 0 {
		return NotEmptyInitDBError
	}

	if seed == nil {
		seed = NewSnapshot(hash, common.Address{})
	}

	db.snapshots[hash] = seed
	if err := db.persister.Put(seed); err != nil {
		return err
	}
	return nil
}

func (db *DBImpl) Get(hash common.Hash) (s *Snapshot, f bool) {
	db.lock.RLock()
	defer db.lock.RUnlock()
	sn, f := db.snapshots[hash]

	return sn, f
}

func (db *DBImpl) Create(parent common.Hash, proposer common.Address) (s *Snapshot, e error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	parentSnapshot, f := db.snapshots[parent]
	if !f {
		return nil, errors.New("no prent is found")
	}

	snapshot := parentSnapshot.NewPendingSnapshot(proposer)

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
	if e := db.persister.Put(pendingSnapshot); e != nil {
		log.Error("Can't persist snapshot")
	}

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
	if e := db.persister.Delete(snapshot.hash); e != nil {
		log.Error("Can't delete snapshot from storage")
	}
	for _, sibl := range snapshot.siblings {
		db.release(sibl)
	}
}

type Account struct {
	nonce   uint64
	balance *big.Int
	origin  common.Address
	voters  []common.Address
}

func (a *Account) Voters() []common.Address {
	return a.voters
}

func (a *Account) Balance() *big.Int {
	return a.balance
}

func (a *Account) Nonce() uint64 {
	return a.nonce
}

func NewAccount(nonce uint64, balance *big.Int) *Account {
	return &Account{nonce: nonce, balance: balance}
}

func (a *Account) Copy() *Account {
	return NewAccount(a.nonce, new(big.Int).Set(a.balance))
}
