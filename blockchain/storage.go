package blockchain

import (
	"errors"
	net "github.com/gagarinchain/network"
	"github.com/gagarinchain/network/common/eth/common"
	pb "github.com/gagarinchain/network/common/protobuff"
	"github.com/golang/protobuf/proto"
)

var (
	NoBlockFoundError = errors.New("no block is found")
)

type BlockPersister struct {
	Storage net.Storage
}

func (bp *BlockPersister) Persist(b *Block) error {
	bytes, i := b.Serialize()
	if i != nil {
		return i
	}

	return bp.Storage.Put(net.Block, b.Header().Hash().Bytes(), bytes)
}

func (b *Block) Serialize() ([]byte, error) {
	bytes, e := proto.Marshal(b.ToStorageProto())
	if e != nil {
		return nil, e
	}
	return bytes, nil
}

func (bp *BlockPersister) Update(b *Block) error {
	if bp.Contains(b.Header().hash) {
		return bp.Persist(b)
	}
	return NoBlockFoundError
}

func (bp *BlockPersister) Load(hash common.Hash) (b *Block, er error) {
	value, er := bp.Storage.Get(net.Block, hash.Bytes())
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
	return bp.Storage.Contains(net.Block, hash.Bytes())
}
