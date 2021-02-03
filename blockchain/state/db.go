package state

import (
	cmn "github.com/gagarinchain/common"
	"github.com/gagarinchain/common/api"
	"github.com/gagarinchain/common/eth/common"
	pb "github.com/gagarinchain/common/protobuff"
	"github.com/gagarinchain/network/storage"
	"github.com/op/go-logging"
	"github.com/pkg/errors"
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
	records         map[common.Hash]api.Record
	snapPersister   *SnapshotPersister
	recordPersister *RecordPersister
	lock            sync.RWMutex
	committee       []common.Address
	bus             cmn.EventBus
}

func NewStateDB(storage storage.Storage, committee []common.Address, bus cmn.EventBus) DB {
	snapPersister := &SnapshotPersister{storage: storage}
	recordPersister := &RecordPersister{storage: storage}
	records := make(map[common.Hash]api.Record)
	db := &DBImpl{records: records, snapPersister: snapPersister, recordPersister: recordPersister, lock: sync.RWMutex{},
		committee: committee, bus: bus}

	db.lock.RLock()
	defer db.lock.RUnlock()

	hashes := snapPersister.Hashes()
	snaps := make(map[common.Hash]*Snapshot)
	for _, hash := range hashes { //parse snapshots
		value, e := snapPersister.Get(hash)
		if e != nil {
			log.Errorf("Can't find snapshot with hash %v", value)
			continue
		}
		snap, e := SnapshotFromProto(value)
		if e != nil {
			log.Error("Can't parse snapshot", e)
			continue
		}
		snaps[hash] = snap
	}

	recHashes := recordPersister.Hashes()
	//todo omit this structure and associate hash with record itself, so we can reduce memory usage
	pbRecs := make(map[common.Hash]*pb.Record)
	for _, h := range recHashes {
		recMessage, e := recordPersister.Get(h)
		if e != nil {
			log.Error("Can't parse record", e)
			return db
		}

		record, e := GetProto(recMessage)
		if e != nil {
			log.Error("Can't parse record", e)
			continue
		}
		rec, e := FromProto(record, snaps, committee, bus)
		if e != nil {
			log.Error("Can't parse record", e)
			continue
		}
		records[rec.snap.hash] = rec
		pbRecs[rec.snap.hash] = record

	}

	for _, s := range records {
		SetParentFromProto(s, pbRecs[s.Hash()], records)
	}

	return db
}

func (db *DBImpl) Init(hash common.Hash, seed *Snapshot) error {
	if len(db.records) > 0 {
		return NotEmptyInitDBError
	}

	if seed == nil {
		seed = NewSnapshot(hash, common.Address{})
	}

	rec := NewRecord(seed, nil, db.committee, db.bus)
	db.records[hash] = rec
	db.persist(rec, nil)
	return nil
}

func (db *DBImpl) Get(hash common.Hash) (r api.Record, f bool) {
	db.lock.RLock()
	defer db.lock.RUnlock()
	sn, f := db.records[hash]

	return sn, f
}

func (db *DBImpl) Create(parent common.Hash, proposer common.Address) (r api.Record, e error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	parentRecord, f := db.records[parent]
	if !f {
		return nil, errors.New("no parent is found")
	}
	pending := parentRecord.NewPendingRecord(proposer)

	return pending, nil
}

func (db *DBImpl) Commit(parent, pending common.Hash) (r api.Record, e error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	parentRecord, f := db.records[parent]
	if !f {
		return nil, errors.New("no parent is found")
	}

	pendingRecord := parentRecord.Pending()
	pendingRecord.SetHash(pending)
	parentRecord.SetPending(nil)
	parentRecord.AddSibling(pendingRecord)
	db.records[pending] = pendingRecord
	db.persist(pendingRecord, parentRecord)

	return pendingRecord, nil
}

func (db *DBImpl) persist(pendingRecord api.Record, parentRecord api.Record) {
	if pr, c := pendingRecord.(*RecordImpl); c {
		if e := db.snapPersister.Put(pr.snap); e != nil {
			log.Error("Can't persist snapshot")
		}
	}
	if e := db.recordPersister.Put(pendingRecord); e != nil {
		log.Error("Can't persist record")
	}
	if parentRecord != nil {
		if e := db.recordPersister.Put(parentRecord); e != nil {
			log.Error("Can't persist parent parent record")
		}
	}
	if parent, c := parentRecord.(*RecordImpl); c {
		if e := db.snapPersister.Put(parent.snap); e != nil {
			log.Error("Can't persist parent snapshot")
		}
	}
}

func (db *DBImpl) Release(blockHash common.Hash) error {
	n, f := db.records[blockHash]
	if !f {
		return errors.New("no snapshot found for block")
	}
	db.release(n)

	return nil
}

func (db *DBImpl) release(record api.Record) {
	delete(db.records, record.Hash())
	if e := db.recordPersister.Delete(record.Hash()); e != nil {
		log.Error("Can't delete record from storage")
	}
	if e := db.snapPersister.Delete(record.Hash()); e != nil {
		log.Error("Can't delete snapshot from storage")
	}
	for _, sibl := range record.Siblings() {
		db.release(sibl)
	}
}
