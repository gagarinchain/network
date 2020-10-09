package test

import (
	"context"
	cmn "github.com/gagarinchain/common"
	"github.com/gagarinchain/common/api"
	"github.com/gagarinchain/common/eth/common"
	"github.com/gagarinchain/common/eth/crypto"
	cmocks "github.com/gagarinchain/common/mocks"
	bch "github.com/gagarinchain/network/blockchain"
	"github.com/gagarinchain/network/blockchain/state"
	"github.com/gagarinchain/network/blockchain/tx"
	"github.com/gagarinchain/network/mocks"
	store "github.com/gagarinchain/network/storage"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestIsSiblingParent(t *testing.T) {
	storage := SoftStorageMock()
	bpersister := &bch.BlockPersister{storage}
	cpersister := &bch.BlockchainPersister{storage}
	bc := bch.CreateBlockchainFromGenesisBlock(&bch.BlockchainConfig{BlockPerister: bpersister, ChainPersister: cpersister, Pool: mockPool(), Db: mockDB(), ProposerGetter: MockProposerForHeight()})
	bc.GetGenesisBlock().SetQC(bch.CreateQuorumCertificate(crypto.EmptyAggregateSignatures(), bc.GetGenesisBlock().Header()))
	head := bc.GetHead()
	newBlock := bc.NewBlock(head, bc.GetGenesisCert(), []byte("newBlock"))
	if _, err := bc.AddBlock(newBlock); err != nil {
		t.Error("can't add block", err)
	}
	assert.True(t, bc.IsSibling(newBlock.Header(), head.Header()))

}

func TestIsSiblingAncestor(t *testing.T) {
	storage := SoftStorageMock()
	bpersister := &bch.BlockPersister{storage}
	cpersister := &bch.BlockchainPersister{storage}
	bc := bch.CreateBlockchainFromGenesisBlock(&bch.BlockchainConfig{BlockPerister: bpersister, ChainPersister: cpersister, Pool: mockPool(), Db: mockDB(), ProposerGetter: MockProposerForHeight()})
	bc.GetGenesisBlock().SetQC(bch.CreateQuorumCertificate(crypto.EmptyAggregateSignatures(), bc.GetGenesisBlock().Header()))

	head := bc.GetHead()
	newBlock := bc.NewBlock(head, bc.GetGenesisCert(), []byte("newBlock"))
	if _, err := bc.AddBlock(newBlock); err != nil {
		t.Error("can't add block", err)
	}
	newBlock2 := bc.NewBlock(newBlock, bc.GetGenesisCert(), []byte("newBlock2"))
	if _, err := bc.AddBlock(newBlock2); err != nil {
		t.Error("can't add block", err)
	}
	newBlock3 := bc.NewBlock(newBlock2, bc.GetGenesisCert(), []byte("newBlock3"))
	if _, err := bc.AddBlock(newBlock3); err != nil {
		t.Error("can't add block", err)
	}
	assert.True(t, bc.IsSibling(newBlock3.Header(), head.Header()))

}

func TestIsSiblingReverseParentSibling(t *testing.T) {
	storage := SoftStorageMock()
	bpersister := &bch.BlockPersister{storage}
	cpersister := &bch.BlockchainPersister{storage}
	bc := bch.CreateBlockchainFromGenesisBlock(&bch.BlockchainConfig{BlockPerister: bpersister, ChainPersister: cpersister, Pool: mockPool(), Db: mockDB(), ProposerGetter: MockProposerForHeight()})
	bc.GetGenesisBlock().SetQC(bch.CreateQuorumCertificate(crypto.EmptyAggregateSignatures(), bc.GetGenesisBlock().Header()))

	head := bc.GetHead()
	newBlock := bc.NewBlock(head, bc.GetGenesisCert(), []byte("first block"))
	if _, err := bc.AddBlock(newBlock); err != nil {
		t.Error("can't add block", err)
	}
	newBlock2 := bc.NewBlock(head, bc.GetGenesisCert(), []byte("second block"))
	if _, err := bc.AddBlock(newBlock2); err != nil {
		t.Error("can't add block", err)
	}

	assert.False(t, bc.IsSibling(newBlock.Header(), newBlock2.Header()))

}

func TestIsSiblingCommonParentSameHeight(t *testing.T) {
	storage := SoftStorageMock()
	bpersister := &bch.BlockPersister{storage}
	cpersister := &bch.BlockchainPersister{storage}
	bc := bch.CreateBlockchainFromGenesisBlock(&bch.BlockchainConfig{BlockPerister: bpersister, ChainPersister: cpersister, Pool: mockPool(), Db: mockDB(), ProposerGetter: MockProposerForHeight()})
	bc.GetGenesisBlock().SetQC(bch.CreateQuorumCertificate(crypto.EmptyAggregateSignatures(), bc.GetGenesisBlock().Header()))

	head := bc.GetHead()
	newBlock := bc.NewBlock(head, bc.GetGenesisCert(), []byte("newBlock"))
	if _, err := bc.AddBlock(newBlock); err != nil {
		t.Error("can't add block", err)
	}
	newBlock2 := bc.NewBlock(head, bc.GetGenesisCert(), []byte("newBlock2"))
	if _, err := bc.AddBlock(newBlock2); err != nil {
		t.Error("can't add block", err)
	}

	assert.False(t, bc.IsSibling(newBlock.Header(), newBlock2.Header()))
}

func TestIsSiblingCommonParentDifferentHeight(t *testing.T) {
	storage := SoftStorageMock()
	bpersister := &bch.BlockPersister{storage}
	cpersister := &bch.BlockchainPersister{storage}
	bc := bch.CreateBlockchainFromGenesisBlock(&bch.BlockchainConfig{BlockPerister: bpersister, ChainPersister: cpersister, Pool: mockPool(), Db: mockDB(), ProposerGetter: MockProposerForHeight()})
	bc.GetGenesisBlock().SetQC(bch.CreateQuorumCertificate(crypto.EmptyAggregateSignatures(), bc.GetGenesisBlock().Header()))

	head := bc.GetHead()
	newBlock := bc.NewBlock(head, bc.GetGenesisCert(), []byte("newBlock"))
	if _, err := bc.AddBlock(newBlock); err != nil {
		t.Error("can't add block", err)
	}
	newBlock2 := bc.NewBlock(head, bc.GetGenesisCert(), []byte("newBlock2"))
	if _, err := bc.AddBlock(newBlock2); err != nil {
		t.Error("can't add block", err)
	}
	newBlock3 := bc.NewBlock(newBlock2, bc.GetGenesisCert(), []byte("newBlock3"))
	if _, err := bc.AddBlock(newBlock3); err != nil {
		t.Error("can't add block", err)
	}

	assert.False(t, bc.IsSibling(newBlock3.Header(), newBlock.Header()))
}

func TestIsSiblingCommonParentDifferentHeight2(t *testing.T) {
	storage := SoftStorageMock()
	bpersister := &bch.BlockPersister{storage}
	cpersister := &bch.BlockchainPersister{storage}
	bc := bch.CreateBlockchainFromGenesisBlock(&bch.BlockchainConfig{BlockPerister: bpersister, ChainPersister: cpersister, Pool: mockPool(), Db: mockDB(), ProposerGetter: MockProposerForHeight()})
	bc.GetGenesisBlock().SetQC(bch.CreateQuorumCertificate(crypto.EmptyAggregateSignatures(), bc.GetGenesisBlock().Header()))

	head := bc.GetHead()
	newBlock := bc.NewBlock(head, bc.GetGenesisCert(), []byte("newBlock"))
	if _, err := bc.AddBlock(newBlock); err != nil {
		t.Error("can't add block", err)
	}
	newBlock2 := bc.NewBlock(head, bc.GetGenesisCert(), []byte("newBlock2"))
	if _, err := bc.AddBlock(newBlock2); err != nil {
		t.Error("can't add block", err)
	}
	newBlock3 := bc.NewBlock(newBlock2, bc.GetGenesisCert(), []byte("newBlock3"))
	if _, err := bc.AddBlock(newBlock3); err != nil {
		t.Error("can't add block", err)
	}

	assert.False(t, bc.IsSibling(newBlock.Header(), newBlock3.Header()))
}

func TestWarmUpFromStorageWithGenesisBlockOnly(t *testing.T) {

	zero := bch.CreateGenesisBlock()
	zero.SetQC(bch.CreateQuorumCertificate(crypto.EmptyAggregateSignatures(), zero.Header()))

	storage := &mocks.Storage{}
	storage.On("Put", mock.AnythingOfType("storage.ResourceType"), mock.AnythingOfType("[]uint8"), mock.AnythingOfType("[]uint8")).Return(nil)
	storage.On("Contains", mock.AnythingOfType("storage.ResourceType"), mock.AnythingOfType("[]uint8")).Return(false)

	storage.On("Get",
		mock.MatchedBy(func(t store.ResourceType) bool {
			return t == store.HeightIndex
		}),
		mock.AnythingOfType("[]uint8")).Return(zero.Header().Hash().Bytes(), nil)
	zeroBBytes, _ := proto.Marshal(zero.GetMessage())
	storage.On("Get",
		mock.MatchedBy(func(t store.ResourceType) bool {
			return t == store.Block
		}),
		mock.AnythingOfType("[]uint8")).Return(zeroBBytes, nil)
	storage.On("Get",
		mock.MatchedBy(func(t store.ResourceType) bool {
			return t == store.CurrentTopHeight
		}),
		mock.AnythingOfType("[]uint8")).Return(store.Int32ToByte(0), nil)
	storage.On("Get",
		mock.MatchedBy(func(t store.ResourceType) bool {
			return t == store.TopCommittedHeight
		}),
		mock.AnythingOfType("[]uint8")).Return(store.Int32ToByte(-1), nil)

	bc := bch.CreateBlockchainFromStorage(&bch.BlockchainConfig{
		Db:      mockDB(),
		Pool:    mockPool(),
		Storage: storage,
	})

	assert.Equal(t, zero, bc.GetGenesisBlock())

}

func TestOnCommit(t *testing.T) {
	storage, _ := store.NewStorage("", nil)
	bpersister := &bch.BlockPersister{storage}
	cpersister := &bch.BlockchainPersister{storage}
	bc := bch.CreateBlockchainFromGenesisBlock(&bch.BlockchainConfig{BlockPerister: bpersister, ChainPersister: cpersister, Pool: mockPool(), Db: mockDB(), ProposerGetter: MockProposerForHeight()})
	genesisBlock := bc.GetGenesisBlock()
	genesisBlock.SetQC(bch.CreateQuorumCertificate(crypto.EmptyAggregateSignatures(), genesisBlock.Header()))
	_, _ = bc.AddBlock(genesisBlock)

	block10 := bc.NewBlock(genesisBlock, bc.GetGenesisCert(), []byte("block 0<-0"))
	block11 := bc.NewBlock(genesisBlock, bc.GetGenesisCert(), []byte("block 0<-1"))

	block20 := bc.NewBlock(block10, bc.GetGenesisCert(), []byte("block 0<-0"))
	block21 := bc.NewBlock(block11, bc.GetGenesisCert(), []byte("block 1<-1"))
	block22 := bc.NewBlock(block11, bc.GetGenesisCert(), []byte("block 1<-2"))

	block30 := bc.NewBlock(block20, bc.GetGenesisCert(), []byte("block 0<-0"))
	block31 := bc.NewBlock(block21, bc.GetGenesisCert(), []byte("block 1<-1"))
	block32 := bc.NewBlock(block22, bc.GetGenesisCert(), []byte("block 2<-2"))

	block40 := bc.NewBlock(block30, bc.GetGenesisCert(), []byte("block 0<-0"))
	block41 := bc.NewBlock(block31, bc.GetGenesisCert(), []byte("block 1<-1"))
	block42 := bc.NewBlock(block31, bc.GetGenesisCert(), []byte("block 1<-2"))
	block43 := bc.NewBlock(block32, bc.GetGenesisCert(), []byte("block 2<-3"))
	block44 := bc.NewBlock(block32, bc.GetGenesisCert(), []byte("block 2<-4"))

	block50 := bc.NewBlock(block41, bc.GetGenesisCert(), []byte("block 1<-0"))
	block51 := bc.NewBlock(block42, bc.GetGenesisCert(), []byte("block 2<-1"))
	block52 := bc.NewBlock(block43, bc.GetGenesisCert(), []byte("block 3<-2"))

	block60 := bc.NewBlock(block50, bc.GetGenesisCert(), []byte("block 0<-0"))

	_, _ = bc.AddBlock(block10)
	_, _ = bc.AddBlock(block11)
	_, _ = bc.AddBlock(block20)
	_, _ = bc.AddBlock(block21)
	_, _ = bc.AddBlock(block22)
	_, _ = bc.AddBlock(block30)
	_, _ = bc.AddBlock(block31)
	_, _ = bc.AddBlock(block32)
	_, _ = bc.AddBlock(block40)
	_, _ = bc.AddBlock(block41)
	_, _ = bc.AddBlock(block42)
	_, _ = bc.AddBlock(block43)
	_, _ = bc.AddBlock(block44)
	_, _ = bc.AddBlock(block50)
	_, _ = bc.AddBlock(block51)
	_, _ = bc.AddBlock(block52)
	_, _ = bc.AddBlock(block60)

	toCommit, orphans, err := bc.OnCommit(block21)

	assert.NoError(t, err)
	assert.Equal(t, genesisBlock, toCommit[0])
	assert.Equal(t, block11, toCommit[1])
	assert.Equal(t, block21, toCommit[2])
	assert.Equal(t, 3, len(toCommit))

	o0, _ := orphans.Get(int32(0))
	o1, _ := orphans.Get(int32(1))
	o2, _ := orphans.Get(int32(2))
	o3, _ := orphans.Get(int32(3))
	o4, _ := orphans.Get(int32(4))
	o5, _ := orphans.Get(int32(5))
	o6, _ := orphans.Get(int32(6))
	assert.Nil(t, o0)

	assert.NotNil(t, o1)
	b1 := o1.([]api.Block)
	assert.Equal(t, 1, len(b1))
	assert.Equal(t, block10, b1[0])

	assert.NotNil(t, o2)
	b2 := o2.([]api.Block)
	assert.Equal(t, 2, len(b2))
	assert.Equal(t, block20, b2[0])
	assert.Equal(t, block22, b2[1])

	assert.NotNil(t, o3)
	b3 := o3.([]api.Block)
	assert.Equal(t, 2, len(b3))
	assert.Equal(t, block30, b3[0])
	assert.Equal(t, block32, b3[1])

	assert.NotNil(t, o4)
	b4 := o4.([]api.Block)
	assert.Equal(t, 3, len(b4))
	assert.Equal(t, block40, b4[0])
	assert.Equal(t, block43, b4[1])
	assert.Equal(t, block44, b4[2])

	assert.NotNil(t, o5)
	b5 := o5.([]api.Block)
	assert.Equal(t, 1, len(b5))
	assert.Equal(t, block52, b5[0])

	assert.Nil(t, o6)

	val, _ := cpersister.GetTopCommittedHeight()
	assert.Equal(t, block21.Height(), val)
}

func TestSignatureUpdate(t *testing.T) {
	storage, _ := store.NewStorage("", nil)
	bpersister := &bch.BlockPersister{storage}
	cpersister := &bch.BlockchainPersister{storage}
	bc := bch.CreateBlockchainFromGenesisBlock(&bch.BlockchainConfig{BlockPerister: bpersister, ChainPersister: cpersister, Pool: mockPool(), Db: mockDB(), ProposerGetter: MockProposerForHeight()})
	genesisBlock := bc.GetGenesisBlock()
	genesisBlock.SetQC(bch.CreateQuorumCertificate(crypto.EmptyAggregateSignatures(), genesisBlock.Header()))
	_, _ = bc.AddBlock(genesisBlock)

	block10 := bc.NewBlock(genesisBlock, bc.GetGenesisCert(), []byte("block 0<-0"))

	block20 := bc.NewBlock(block10, bch.CreateQuorumCertificate(crypto.EmptyAggregateSignatures(), block10.Header()), []byte("block 0<-0"))

	_, _ = bc.AddBlock(block10)
	_, _ = bc.AddBlock(block20)

	assert.Equal(t, bc.GetGenesisBlock().Signature(), crypto.EmptyAggregateSignatures())
	assert.Equal(t, bc.GetBlockByHash(block10.Header().Hash()).Signature(), crypto.EmptyAggregateSignatures())
	assert.Nil(t, bc.GetBlockByHash(block20.Header().Hash()).Signature())

}

func TestWarmUpFromStorageWithRichChain(t *testing.T) {
	storage, _ := store.NewStorage("", nil)
	bpersister := &bch.BlockPersister{Storage: storage}
	cpersister := &bch.BlockchainPersister{Storage: storage}

	bc := bch.CreateBlockchainFromGenesisBlock(&bch.BlockchainConfig{BlockPerister: bpersister, ChainPersister: cpersister, Pool: mockPool(), Db: mockDB(), ProposerGetter: MockProposerForHeight()})
	genesisBlock := bc.GetGenesisBlock()
	genesisQC := bch.CreateQuorumCertificate(crypto.EmptyAggregateSignatures(), genesisBlock.Header())
	genesisBlock.SetQC(genesisQC)
	bc.UpdateGenesisBlockQC(genesisQC)

	block12 := bc.NewBlock(bc.GetHead(), bc.GetGenesisCert(), []byte("block 1<-2"))
	_, _ = bc.AddBlock(block12)
	block23 := bc.NewBlock(bc.GetHead(), bc.GetGenesisCert(), []byte("block 2<-3"))
	_, _ = bc.AddBlock(block23)
	block34 := bc.NewBlock(bc.GetHead(), bc.GetGenesisCert(), []byte("block 3<-4"))
	_, _ = bc.AddBlock(block34)
	block45 := bc.NewBlock(block34, bc.GetGenesisCert(), []byte("block 4<-5"))
	_, _ = bc.AddBlock(block45)
	block56 := bc.NewBlock(block45, bc.GetGenesisCert(), []byte("block 5<-6"))
	_, _ = bc.AddBlock(block56)
	block47 := bc.NewBlock(block34, bc.GetGenesisCert(), []byte("block 4<-7"))
	_, _ = bc.AddBlock(block47)
	bc.OnCommit(block34)

	bc2 := bch.CreateBlockchainFromStorage(&bch.BlockchainConfig{
		Db:      mockDB(),
		Pool:    mockPool(),
		Storage: storage,
		Delta:   5000,
	})

	assert.Equal(t, genesisBlock, bc2.GetGenesisBlock())
	assert.Equal(t, block34, bc2.GetTopCommittedBlock())
	assert.Equal(t, []api.Block{block45, block47}, bc2.GetBlockByHeight(4))
	assert.Equal(t, block56, bc2.GetHead())

}

func mockPool() tx.TransactionPool {
	pool := &mocks.TransactionPool{}
	pool.On("RemoveAll")

	txs := make(chan []api.Transaction)
	close(txs)

	pool.On("Drain", mock.MatchedBy(func(ctx context.Context) bool { return true })).Return(txs)
	iterator := &cmocks.Iterator{}
	iterator.On("Next").Return(nil)
	pool.On("Iterator").Return(iterator)
	return pool
}

func mockDB() state.DB {
	db := &mocks.DB{}
	db.On("Get", mock.AnythingOfType("common.Hash")).Return(nil, false)
	db.On("Init", mock.AnythingOfType("common.Hash"), mock.AnythingOfType("*state.Snapshot")).Return(nil)
	snapshot := state.NewSnapshot(crypto.Keccak256Hash(), common.Address{})
	record := state.NewRecord(snapshot, nil, &cmn.NullBus{})
	db.On("Create", mock.AnythingOfType("common.Hash"), mock.AnythingOfType("common.Address")).Return(record, nil)
	db.On("Commit", mock.AnythingOfType("common.Hash"), mock.AnythingOfType("common.Hash")).Return(record, nil)
	db.On("Release", mock.AnythingOfType("common.Hash")).Return(nil)

	return db
}
