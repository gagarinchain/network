package test

import (
	bch "github.com/poslibp2p/blockchain"
	"github.com/poslibp2p/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestIsSiblingParent(t *testing.T) {
	bc := bch.CreateBlockchainFromGenesisBlock(mockStorage(), nil)
	head := bc.GetHead()
	newBlock := bc.NewBlock(head, bc.GetGenesisCert(), []byte("newBlock"))
	if err := bc.AddBlock(newBlock); err != nil {
		t.Error("can't add block", err)
	}
	assert.True(t, bc.IsSibling(newBlock.Header(), head.Header()))

}

func TestIsSiblingAncestor(t *testing.T) {
	bc := bch.CreateBlockchainFromGenesisBlock(mockStorage(), nil)
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
	head := bc.GetHead()
	newBlock := bc.NewBlock(head, bc.GetGenesisCert(), []byte(""))
	if err := bc.AddBlock(newBlock); err != nil {
		t.Error("can't add block", err)
	}

	assert.False(t, bc.IsSibling(head.Header(), newBlock.Header()))

}

func TestIsSiblingCommonParentSameHeight(t *testing.T) {
	bc := bch.CreateBlockchainFromGenesisBlock(mockStorage(), nil)
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

func mockStorage() bch.Storage {
	storage := &mocks.Storage{}
	storage.On("PutBlock", mock.AnythingOfType("*blockchain.Block")).Return(nil)
	storage.On("GetBlock", mock.AnythingOfType("common.Hash")).Return(nil, nil)
	storage.On("Contains", mock.AnythingOfType("common.Hash")).Return(false)

	return storage
}
