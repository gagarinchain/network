package test

import (
	bch "github.com/poslibp2p/blockchain"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestIsSiblingParent(t *testing.T) {
	bc := bch.CreateBlockchainFromGenesisBlock()
	head := bc.GetHead()
	newBlock := bc.NewBlock(head, bc.GetGenesisCert(), []byte("newBlock"))
	if err := bc.AddBlock(newBlock); err != nil {
		t.Error("can't add block", err)
	}
	assert.True(t, bc.IsSibling(newBlock.Header(), head.Header()))

}

func TestIsSiblingAncestor(t *testing.T) {
	bc := bch.CreateBlockchainFromGenesisBlock()
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
	bc := bch.CreateBlockchainFromGenesisBlock()
	head := bc.GetHead()
	newBlock := bc.NewBlock(head, bc.GetGenesisCert(), []byte(""))
	if err := bc.AddBlock(newBlock); err != nil {
		t.Error("can't add block", err)
	}

	assert.False(t, bc.IsSibling(head.Header(), newBlock.Header()))

}

func TestIsSiblingCommonParentSameHeight(t *testing.T) {
	bc := bch.CreateBlockchainFromGenesisBlock()
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
	bc := bch.CreateBlockchainFromGenesisBlock()
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
	bc := bch.CreateBlockchainFromGenesisBlock()
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
