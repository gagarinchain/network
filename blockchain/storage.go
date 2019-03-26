package blockchain

import (
	"github.com/gogo/protobuf/proto"
	"github.com/poslibp2p/eth/common"
	"github.com/poslibp2p/message/protobuff"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/storage"
)

type Storage interface {
	PutBlock(b *Block) error
	GetBlock(hash common.Hash) (b *Block, er error)
	Contains(hash common.Hash) bool
}

type StorageImpl struct {
	DB   *leveldb.DB
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
		DB:   db,
		path: path,
	}, nil
}

func (s *StorageImpl) PutBlock(b *Block) error {
	bytes, e := proto.Marshal(b.GetMessage())
	if e != nil {
		return e
	}
	e = s.DB.Put(b.Header().Hash().Bytes(), bytes, &opt.WriteOptions{})

	if e != nil {
		return e
	}

	return nil
}

func (s *StorageImpl) Contains(hash common.Hash) bool {
	b, _ := s.DB.Has(hash.Bytes(), &opt.ReadOptions{})
	return b
}
func (s *StorageImpl) GetBlock(hash common.Hash) (b *Block, er error) {
	value, er := s.DB.Get(hash.Bytes(), &opt.ReadOptions{})
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
