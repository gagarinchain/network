package common

import (
	"encoding/binary"
	net "github.com/gagarinchain/network"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/storage"
	"github.com/syndtr/goleveldb/leveldb/util"
	"path"
)

const DefaultIntValue = -1
const StorageName = "gagarin"

type StorageImpl struct {
	db   *leveldb.DB
	path string
}

func NewStorage(p string, opts *opt.Options) (net.Storage, error) {
	var nopts opt.Options

	var err error
	var db *leveldb.DB

	if p == "" {
		db, err = leveldb.Open(storage.NewMemStorage(), &nopts)
	} else {
		p = path.Join(p, StorageName)
		db, err = leveldb.OpenFile(p, &nopts)
		log.Debugf("Created storage at %v", p)
		if errors.IsCorrupted(err) && !nopts.GetReadOnly() {
			db, err = leveldb.RecoverFile(p, &nopts)
		}
	}

	if err != nil {
		return nil, err
	}

	return &StorageImpl{
		db:   db,
		path: p,
	}, nil
}

func (s *StorageImpl) Put(rtype net.ResourceType, key []byte, value []byte) error {
	bytes := append(make([]byte, 1), byte(rtype))
	k := append(bytes, key...)
	return s.db.Put(k, value, &opt.WriteOptions{})
}

func (s *StorageImpl) Get(rtype net.ResourceType, key []byte) (value []byte, err error) {
	prefix := append(make([]byte, 1), byte(rtype))
	k := append(prefix, key...)
	return s.db.Get(k, &opt.ReadOptions{})
}

func (s *StorageImpl) Contains(rtype net.ResourceType, key []byte) bool {
	bytes := append(make([]byte, 1), byte(rtype))
	k := append(bytes, key...)
	b, _ := s.db.Has(k, &opt.ReadOptions{})
	return b
}

func (s *StorageImpl) Delete(rtype net.ResourceType, key []byte) error {
	bytes := append(make([]byte, 1), byte(rtype))
	k := append(bytes, key...)
	return s.db.Delete(k, &opt.WriteOptions{})
}

func (s *StorageImpl) Keys(rtype net.ResourceType, keyPrefix []byte) (keys [][]byte) {
	bytes := append(make([]byte, 1), byte(rtype))
	iter := s.db.NewIterator(util.BytesPrefix(append(bytes, keyPrefix...)), nil)

	for iter.Next() {
		key := iter.Key()
		keyCopy := make([]byte, len(key))
		copy(keyCopy, key)
		keys = append(keys, keyCopy)

	}

	iter.Release()

	return keys
}

func Int32ToByte(val int32) []byte {
	buf := make([]byte, binary.MaxVarintLen64)
	binary.PutVarint(buf, int64(val))

	return buf
}

func ByteToInt32(value []byte) (val int32, err error) {
	int64Val, n := binary.Varint(value)
	if n > binary.MaxVarintLen32 {
		return DefaultIntValue, errors.New("wrong int32 length")
	}
	return int32(int64Val), nil
}

func (s *StorageImpl) getInt32(key []byte) (val int32, err error) {
	value, err := s.db.Get(key, &opt.ReadOptions{})
	if err != nil {
		return DefaultIntValue, err
	}
	int64Val, n := binary.Varint(value)
	if n > binary.MaxVarintLen32 {
		return DefaultIntValue, errors.New("wrong int32 length")
	}
	return int32(int64Val), nil
}

func (s *StorageImpl) Stats() *leveldb.DBStats {
	stats := &leveldb.DBStats{}
	if err := s.db.Stats(stats); err != nil {
		log.Error(err)
		return nil
	}
	return stats
}
