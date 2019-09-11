package state

import (
	"github.com/gagarinchain/network"
	"github.com/gagarinchain/network/common/eth/common"
	pb "github.com/gagarinchain/network/common/protobuff"
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
	records         map[common.Hash]*Record
	snapPersister   *SnapshotPersister
	recordPersister *RecordPersister
	lock            sync.RWMutex
}

func NewStateDB(storage gagarinchain.Storage) DB {
	snapPersister := &SnapshotPersister{storage: storage}
	recordPersister := &RecordPersister{storage: storage}
	records := make(map[common.Hash]*Record)
	db := &DBImpl{records: records, snapPersister: snapPersister, recordPersister: recordPersister, lock: sync.RWMutex{}}

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
		rec, e := FromProto(record, snaps)
		if e != nil {
			log.Error("Can't parse record", e)
			continue
		}
		records[rec.snap.hash] = rec
		pbRecs[rec.snap.hash] = record

	}

	for _, s := range records {
		s.SetParentFromProto(pbRecs[s.snap.hash], records)
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

	rec := NewRecord(seed, nil)
	db.records[hash] = rec
	db.persist(rec, nil)
	return nil
}

func (db *DBImpl) Get(hash common.Hash) (r *Record, f bool) {
	db.lock.RLock()
	defer db.lock.RUnlock()
	sn, f := db.records[hash]

	return sn, f
}

func (db *DBImpl) Create(parent common.Hash, proposer common.Address) (r *Record, e error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	parentRecord, f := db.records[parent]
	if !f {
		return nil, errors.New("no prent is found")
	}
	pending := parentRecord.NewPendingRecord(proposer)

	return pending, nil
}

func (db *DBImpl) Commit(parent, pending common.Hash) (r *Record, e error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	parentRecord, f := db.records[parent]
	if !f {
		return nil, errors.New("no prent is found")
	}

	pendingRecord := parentRecord.Pending()
	pendingRecord.snap.hash = pending
	parentRecord.pending = nil
	parentRecord.siblings = append(parentRecord.siblings, pendingRecord)
	db.records[pending] = pendingRecord
	db.persist(pendingRecord, parentRecord)

	return pendingRecord, nil
}

func (db *DBImpl) persist(pendingRecord *Record, parentRecord *Record) {
	if e := db.snapPersister.Put(pendingRecord.snap); e != nil {
		log.Error("Can't persist snapshot")
	}
	if e := db.recordPersister.Put(pendingRecord); e != nil {
		log.Error("Can't persist record")
	}
	if parentRecord != nil {
		if e := db.recordPersister.Put(parentRecord); e != nil {
			log.Error("Can't persist parent record")
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

func (db *DBImpl) release(record *Record) {
	delete(db.records, record.snap.hash)
	if e := db.recordPersister.Delete(record.snap.hash); e != nil {
		log.Error("Can't delete record from storage")
	}
	if e := db.snapPersister.Delete(record.snap.hash); e != nil {
		log.Error("Can't delete snapshot from storage")
	}
	for _, sibl := range record.siblings {
		db.release(sibl)
	}
}
