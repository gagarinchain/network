package test

import (
	bch "github.com/poslibp2p/blockchain"
	"github.com/poslibp2p/common/eth/common"
	"github.com/poslibp2p/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestIsSiblingParent(t *testing.T) {
	bc := bch.CreateBlockchainFromGenesisBlock(mockStorage(), nil)
	bc.GetGenesisBlock().SetQC(bch.CreateQuorumCertificate([]byte("valid"), bc.GetGenesisBlock().Header()))
	head := bc.GetHead()
	newBlock := bc.NewBlock(head, bc.GetGenesisCert(), []byte("newBlock"))
	if err := bc.AddBlock(newBlock); err != nil {
		t.Error("can't add block", err)
	}
	assert.True(t, bc.IsSibling(newBlock.Header(), head.Header()))

}

func TestIsSiblingAncestor(t *testing.T) {
	bc := bch.CreateBlockchainFromGenesisBlock(mockStorage(), nil)
	bc.GetGenesisBlock().SetQC(bch.CreateQuorumCertificate([]byte("valid"), bc.GetGenesisBlock().Header()))

	head := bc.GetHead()
	newBlock := bc.NewBlock(head, bc.GetGenesisCert(), []byte("newBlock"))
	if err := bc.AddBlock(newBlock); err != nil {
		t.Error("can't add block", err)
	}
	newBlock2 := bc.NewBlock(newBlock, bc.GetGenesisCert(), []byte("newBlock2"))
	if err := bc.AddBlock(newBlock2); err != nil {
		t.Error("can't add block", err)
	}
	newBlock3 := bc.NewBlock(newBlock2, bc.GetGenesisCert(), []byte("newBlock3"))
	if err := bc.AddBlock(newBlock3); err != nil {
		t.Error("can't add block", err)
	}
	assert.True(t, bc.IsSibling(newBlock3.Header(), head.Header()))

}

func TestIsSiblingReverseParentSibling(t *testing.T) {
	bc := bch.CreateBlockchainFromGenesisBlock(mockStorage(), nil)
	bc.GetGenesisBlock().SetQC(bch.CreateQuorumCertificate([]byte("valid"), bc.GetGenesisBlock().Header()))

	head := bc.GetHead()
	newBlock := bc.NewBlock(head, bc.GetGenesisCert(), []byte("first block"))
	if err := bc.AddBlock(newBlock); err != nil {
		t.Error("can't add block", err)
	}
	newBlock2 := bc.NewBlock(head, bc.GetGenesisCert(), []byte("second block"))
	if err := bc.AddBlock(newBlock2); err != nil {
		t.Error("can't add block", err)
	}

	assert.False(t, bc.IsSibling(newBlock.Header(), newBlock2.Header()))

}

func TestIsSiblingCommonParentSameHeight(t *testing.T) {
	bc := bch.CreateBlockchainFromGenesisBlock(mockStorage(), nil)
	bc.GetGenesisBlock().SetQC(bch.CreateQuorumCertificate([]byte("valid"), bc.GetGenesisBlock().Header()))

	head := bc.GetHead()
	newBlock := bc.NewBlock(head, bc.GetGenesisCert(), []byte("newBlock"))
	if err := bc.AddBlock(newBlock); err != nil {
		t.Error("can't add block", err)
	}
	newBlock2 := bc.NewBlock(head, bc.GetGenesisCert(), []byte("newBlock2"))
	if err := bc.AddBlock(newBlock2); err != nil {
		t.Error("can't add block", err)
	}

	assert.False(t, bc.IsSibling(newBlock.Header(), newBlock2.Header()))
}

func TestIsSiblingCommonParentDifferentHeight(t *testing.T) {
	bc := bch.CreateBlockchainFromGenesisBlock(mockStorage(), nil)
	bc.GetGenesisBlock().SetQC(bch.CreateQuorumCertificate([]byte("valid"), bc.GetGenesisBlock().Header()))

	head := bc.GetHead()
	newBlock := bc.NewBlock(head, bc.GetGenesisCert(), []byte("newBlock"))
	if err := bc.AddBlock(newBlock); err != nil {
		t.Error("can't add block", err)
	}
	newBlock2 := bc.NewBlock(head, bc.GetGenesisCert(), []byte("newBlock2"))
	if err := bc.AddBlock(newBlock2); err != nil {
		t.Error("can't add block", err)
	}
	newBlock3 := bc.NewBlock(newBlock2, bc.GetGenesisCert(), []byte("newBlock3"))
	if err := bc.AddBlock(newBlock3); err != nil {
		t.Error("can't add block", err)
	}

	assert.False(t, bc.IsSibling(newBlock3.Header(), newBlock.Header()))
}

func TestIsSiblingCommonParentDifferentHeight2(t *testing.T) {
	bc := bch.CreateBlockchainFromGenesisBlock(mockStorage(), nil)
	bc.GetGenesisBlock().SetQC(bch.CreateQuorumCertificate([]byte("valid"), bc.GetGenesisBlock().Header()))

	head := bc.GetHead()
	newBlock := bc.NewBlock(head, bc.GetGenesisCert(), []byte("newBlock"))
	if err := bc.AddBlock(newBlock); err != nil {
		t.Error("can't add block", err)
	}
	newBlock2 := bc.NewBlock(head, bc.GetGenesisCert(), []byte("newBlock2"))
	if err := bc.AddBlock(newBlock2); err != nil {
		t.Error("can't add block", err)
	}
	newBlock3 := bc.NewBlock(newBlock2, bc.GetGenesisCert(), []byte("newBlock3"))
	if err := bc.AddBlock(newBlock3); err != nil {
		t.Error("can't add block", err)
	}

	assert.False(t, bc.IsSibling(newBlock.Header(), newBlock3.Header()))
}

func TestWarmUpFromStorageWithGenesisBlockOnly(t *testing.T) {

	storage := &mocks.Storage{}
	storage.On("PutBlock", mock.AnythingOfType("*blockchain.Block")).Return(nil)
	storage.On("Contains", mock.AnythingOfType("common.Hash")).Return(false)
	storage.On("PutCurrentTopHeight", mock.AnythingOfType("int32")).Return(nil)
	storage.On("GetCurrentTopHeight").Return(int32(0), nil)
	storage.On("GetTopCommittedHeight").Return(int32(-1), nil)
	zero := bch.CreateGenesisBlock()
	zero.SetQC(bch.CreateQuorumCertificate([]byte("valid"), zero.Header()))
	storage.On("GetHeightIndexRecord", int32(0)).Return([]common.Hash{zero.Header().Hash()}, nil)
	storage.On("GetBlock", mock.AnythingOfType("common.Hash")).Return(zero, nil)

	bc := bch.CreateBlockchainFromStorage(storage, nil)

	assert.Equal(t, zero, bc.GetGenesisBlock())

}

func TestWarmUpFromStorageWithRichChain(t *testing.T) {
	storage, _ := bch.NewStorage("", nil)
	bc := bch.CreateBlockchainFromGenesisBlock(storage, nil)
	genesisBlock := bc.GetGenesisBlock()
	genesisBlock.SetQC(bch.CreateQuorumCertificate([]byte("valid"), genesisBlock.Header()))

	_ = bc.AddBlock(genesisBlock)

	block12 := bc.NewBlock(bc.GetHead(), bc.GetGenesisCert(), []byte("block 1<-2"))
	_ = bc.AddBlock(block12)
	block23 := bc.NewBlock(bc.GetHead(), bc.GetGenesisCert(), []byte("block 2<-3"))
	_ = bc.AddBlock(block23)
	block34 := bc.NewBlock(bc.GetHead(), bc.GetGenesisCert(), []byte("block 3<-4"))
	_ = bc.AddBlock(block34)
	block45 := bc.NewBlock(block34, bc.GetGenesisCert(), []byte("block 4<-5"))
	_ = bc.AddBlock(block45)
	block56 := bc.NewBlock(block45, bc.GetGenesisCert(), []byte("block 5<-6"))
	_ = bc.AddBlock(block56)
	block47 := bc.NewBlock(block34, bc.GetGenesisCert(), []byte("block 4<-7"))
	_ = bc.AddBlock(block47)
	bc.OnCommit(block34)

	bc2 := bch.CreateBlockchainFromStorage(storage, nil)

	assert.Equal(t, genesisBlock, bc2.GetGenesisBlock())
	assert.Equal(t, block34, bc2.GetTopCommittedBlock())
	assert.Equal(t, block56, bc2.GetHead())

}

func mockStorage() *mocks.Storage {
	storage := &mocks.Storage{}
	storage.On("PutBlock", mock.AnythingOfType("*blockchain.Block")).Return(nil)
	storage.On("GetBlock", mock.AnythingOfType("common.Hash")).Return(nil, nil)
	storage.On("Contains", mock.AnythingOfType("common.Hash")).Return(false)
	storage.On("PutCurrentTopHeight", mock.AnythingOfType("int32")).Return(nil)

	return storage
}
