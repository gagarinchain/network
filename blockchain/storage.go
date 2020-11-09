package blockchain

import (
	"errors"
	"github.com/gagarinchain/common/api"
	"github.com/gagarinchain/common/eth/common"
	pb "github.com/gagarinchain/common/protobuff"
	"github.com/gagarinchain/network/storage"
	"github.com/golang/protobuf/proto"
	"github.com/syndtr/goleveldb/leveldb"
)

var (
	NoBlockFoundError = errors.New("no block is found")
)

type BlockPersister struct {
	Storage storage.Storage
}

func (bp *BlockPersister) Persist(b api.Block) error {
	bytes, i := b.Serialize()
	if i != nil {
		return i
	}

	return bp.Storage.Put(storage.Block, b.Header().Hash().Bytes(), bytes)
}

func (bp *BlockPersister) Update(b api.Block) error {
	if bp.Contains(b.Header().Hash()) {
		return bp.Persist(b)
	}
	return NoBlockFoundError
}

func (bp *BlockPersister) Load(hash common.Hash) (b api.Block, er error) {
	value, er := bp.Storage.Get(storage.Block, hash.Bytes())
	if er == leveldb.ErrNotFound {
		return nil, NoBlockFoundError
	}
	if er != nil {
		return nil, er
	}
	if value == nil {
		return nil, NoBlockFoundError
	}
	block := &pb.BlockS{}
	er = proto.Unmarshal(value, block)
	if er != nil {
		return nil, er
	}

	return CreateBlockFromStorage(block), nil
}

func (bp *BlockPersister) Contains(hash common.Hash) bool {
	return bp.Storage.Contains(storage.Block, hash.Bytes())
}
