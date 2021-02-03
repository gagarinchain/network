package storage

import "github.com/syndtr/goleveldb/leveldb"

type Persistable interface {
	Persist(storage Storage) error
}

type ResourceType byte

const Block = ResourceType(0x0)
const HeightIndex = ResourceType(0x1)
const CurrentEpoch = ResourceType(0x2)
const CurrentView = ResourceType(0x3)
const TopCommittedHeight = ResourceType(0x4)
const CurrentTopHeight = ResourceType(0x5)
const Snapshot = ResourceType(0x6)
const Record = ResourceType(0x7)
const VHeight = ResourceType(0x8)
const LastExecutedBlock = ResourceType(0x9)
const HQC = ResourceType(0xa)
const HC = ResourceType(0xb)

type Storage interface {
	Put(rtype ResourceType, key []byte, value []byte) error
	Get(rtype ResourceType, key []byte) (value []byte, err error)
	Contains(rtype ResourceType, key []byte) bool
	Delete(rtype ResourceType, key []byte) error
	Keys(rtype ResourceType, keyPrefix []byte) (keys [][]byte)
	Stats() *leveldb.DBStats
	Close()
}
