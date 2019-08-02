package blockchain

import (
	"encoding/binary"
	"github.com/gagarinchain/network/common/eth/common"
	"github.com/gagarinchain/network/common/protobuff"
	"github.com/gogo/protobuf/proto"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/storage"
)

type Storage interface {
	PutCurrentEpoch(currentEpoch int32) error
	GetCurrentEpoch() (val int32, err error)
	PutCurrentTopHeight(currentTopHeight int32) error
	GetCurrentTopHeight() (val int32, err error)
	PutTopCommittedHeight(currentTopHeight int32) error
	GetTopCommittedHeight() (val int32, err error)
	PutBlock(b *Block) error
	GetBlock(hash common.Hash) (b *Block, er error)
	GetHeightIndexRecord(height int32) (hashes []common.Hash, err error)

	Contains(hash common.Hash) bool
	Stats() *leveldb.DBStats
}

const BlockPrefix = byte(0x0)
const HeightIndexPrefix = byte(0x1)
const CurrentEpochPrefix = byte(0x2)
const CurrentTopHeightPrefix = byte(0x3)
const TopCommittedHeightPrefix = byte(0x4)

const DefaultIntValue = -1

type StorageImpl struct {
	db   *leveldb.DB
	path string
}

func NewStorage(path string, opts *opt.Options) (Storage, error) {
	var nopts opt.Options
	if opts != nil {
		nopts = opt.Options(*opts)
	}

	var err error
	var db *leveldb.DB

	if path == "" {
		db, err = leveldb.Open(storage.NewMemStorage(), &nopts)
	} else {
		db, err = leveldb.OpenFile(path, &nopts)
		if errors.IsCorrupted(err) && !nopts.GetReadOnly() {
			db, err = leveldb.RecoverFile(path, &nopts)
		}
	}

	if err != nil {
		return nil, err
	}

	return &StorageImpl{
		db:   db,
		path: path,
	}, nil
}

func (s *StorageImpl) PutCurrentEpoch(currentEpoch int32) error {
	prefix := append(make([]byte, 1), CurrentEpochPrefix)
	return s.putInt32(prefix, currentEpoch)
}

func (s *StorageImpl) GetCurrentEpoch() (val int32, err error) {
	prefix := append(make([]byte, 1), CurrentEpochPrefix)
	return s.getInt32(prefix)
}

func (s *StorageImpl) PutCurrentTopHeight(currentTopHeight int32) error {
	prefix := append(make([]byte, 1), CurrentTopHeightPrefix)
	return s.putInt32(prefix, currentTopHeight)
}

func (s *StorageImpl) GetCurrentTopHeight() (val int32, err error) {
	prefix := append(make([]byte, 1), CurrentTopHeightPrefix)
	return s.getInt32(prefix)
}

func (s *StorageImpl) PutTopCommittedHeight(currentTopHeight int32) error {
	prefix := append(make([]byte, 1), TopCommittedHeightPrefix)
	return s.putInt32(prefix, currentTopHeight)
}

func (s *StorageImpl) GetTopCommittedHeight() (val int32, err error) {
	prefix := append(make([]byte, 1), TopCommittedHeightPrefix)
	return s.getInt32(prefix)
}

func (s *StorageImpl) putInt32(key []byte, val int32) error {
	buf := make([]byte, binary.MaxVarintLen64)
	binary.PutVarint(buf, int64(val))
	return s.db.Put(key, buf, &opt.WriteOptions{})
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

func (s *StorageImpl) PutBlock(b *Block) error {
	bytes, e := proto.Marshal(b.GetMessage())
	if e != nil {
		return e
	}
	prefix := append(make([]byte, 1), BlockPrefix)
	key := append(prefix, b.Header().Hash().Bytes()...)
	if err := s.putHeightIndexRecord(b); err != nil {
		return err
	}
	return s.db.Put(key, bytes, &opt.WriteOptions{})
}

func (s *StorageImpl) Contains(hash common.Hash) bool {
	bytes := append(make([]byte, 1), BlockPrefix)
	key := append(bytes, hash.Bytes()...)
	b, _ := s.db.Has(key, &opt.ReadOptions{})
	return b
}
func (s *StorageImpl) Stats() *leveldb.DBStats {
	stats := &leveldb.DBStats{}
	if err := s.db.Stats(stats); err != nil {
		log.Error(err)
		return nil
	}
	return stats
}

func (s *StorageImpl) GetBlock(hash common.Hash) (b *Block, er error) {
	prefix := append(make([]byte, 1), BlockPrefix)
	key := append(prefix, hash.Bytes()...)
	value, er := s.db.Get(key, &opt.ReadOptions{})
	if er != nil {
		return nil, er
	}
	block := &pb.Block{}
	er = proto.Unmarshal(value, block)
	if er != nil {
		return nil, er
	}

	return CreateBlockFromMessage(block), nil
}

func (s *StorageImpl) putHeightIndexRecord(b *Block) error {
	indexKey := getIndexKey(int64(b.Header().Height()))
	found, err := s.db.Has(indexKey, &opt.ReadOptions{})
	var indexValue []byte
	if err != nil {
		return err
	}

	if found {
		value, err := s.db.Get(indexKey, &opt.ReadOptions{})
		if err != nil {
			return err
		}
		indexValue = value
	}

	indexValue = append(indexValue, b.Header().Hash().Bytes()...)
	return s.db.Put(indexKey, indexValue, &opt.WriteOptions{})
}

func (s *StorageImpl) GetHeightIndexRecord(height int32) (hashes []common.Hash, err error) {
	indexKey := getIndexKey(int64(height))
	val, err := s.db.Get(indexKey, &opt.ReadOptions{})
	if err != nil {
		return nil, err
	}

	hashes = make([]common.Hash, len(val)/common.HashLength)

	for i := 0; i < len(val)/common.HashLength; i++ {
		hashes[i] = common.BytesToHash(val[i*common.HashLength : (i+1)*common.HashLength])
	}

	return hashes, nil
}

func getIndexKey(h int64) []byte {
	indexPrefix := append(make([]byte, 1), HeightIndexPrefix)
	buf := make([]byte, binary.MaxVarintLen64)
	binary.PutVarint(buf, h)
	indexKey := append(indexPrefix, buf...)
	return indexKey
}
