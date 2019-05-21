package test

import (
	"github.com/poslibp2p/blockchain"
	"github.com/poslibp2p/common/eth/crypto"
	"github.com/poslibp2p/common/protobuff"
	"github.com/poslibp2p/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

//Note that QC hash does not matter, since genesis.QC.header is genesis.header
func TestIsValidGenesisBlock(t *testing.T) {
	storage := &mocks.Storage{}
	bsrv := &mocks.BlockService{}
	storage.On("PutBlock", mock.AnythingOfType("*blockchain.Block")).Return(nil)
	storage.On("GetBlock", mock.AnythingOfType("common.Hash")).Return(nil, nil)
	storage.On("Contains", mock.AnythingOfType("common.Hash")).Return(false)
	storage.On("PutCurrentTopHeight", mock.AnythingOfType("int32")).Return(nil)
	bc := blockchain.CreateBlockchainFromGenesisBlock(&blockchain.Config{Storage: storage, BlockService: bsrv, Pool: mockPool(), Db: mockDB()})
	bc.GetGenesisBlock().SetQC(blockchain.CreateQuorumCertificate([]byte("valid"), bc.GetGenesisBlock().Header()))
	b, e := blockchain.IsValid(bc.GetGenesisBlock())
	assert.NoError(t, e)
	assert.True(t, b)

}

func TestIsValidBlock(t *testing.T) {
	storage := &mocks.Storage{}
	bsrv := &mocks.BlockService{}
	storage.On("PutBlock", mock.AnythingOfType("*blockchain.Block")).Return(nil)
	storage.On("GetBlock", mock.AnythingOfType("common.Hash")).Return(nil, nil)
	storage.On("Contains", mock.AnythingOfType("common.Hash")).Return(false)
	storage.On("PutCurrentTopHeight", mock.AnythingOfType("int32")).Return(nil)
	bc := blockchain.CreateBlockchainFromGenesisBlock(&blockchain.Config{Storage: storage, BlockService: bsrv, Pool: mockPool(), Db: mockDB()})
	bc.GetGenesisBlock().SetQC(blockchain.CreateQuorumCertificate([]byte("valid"), bc.GetGenesisBlock().Header()))

	newBlock := bc.NewBlock(bc.GetGenesisBlock(), bc.GetGenesisCert(), []byte("Hello Hotstuff"))
	b, e := blockchain.IsValid(newBlock)
	assert.NoError(t, e)
	assert.True(t, b)
}
func TestIsNotValidWithBrokenHash(t *testing.T) {
	storage := &mocks.Storage{}
	bsrv := &mocks.BlockService{}
	storage.On("PutBlock", mock.AnythingOfType("*blockchain.Block")).Return(nil)
	storage.On("GetBlock", mock.AnythingOfType("common.Hash")).Return(nil, nil)
	storage.On("Contains", mock.AnythingOfType("common.Hash")).Return(false)
	storage.On("PutCurrentTopHeight", mock.AnythingOfType("int32")).Return(nil)
	bc := blockchain.CreateBlockchainFromGenesisBlock(&blockchain.Config{Storage: storage, BlockService: bsrv, Pool: mockPool(), Db: mockDB()})
	bc.GetGenesisBlock().SetQC(blockchain.CreateQuorumCertificate([]byte("valid"), bc.GetGenesisBlock().Header()))

	newBlock := bc.NewBlock(bc.GetGenesisBlock(), bc.GetGenesisCert(), []byte("Hello Hotstuff"))

	msg := newBlock.GetMessage()
	msg.Header.Hash = crypto.Keccak256([]byte("Hello from other side"))

	b, e := blockchain.IsValid(blockchain.CreateBlockFromMessage(msg))
	assert.EqualError(t, e, "block hash is not valid")
	assert.False(t, b)
}

func TestIsNotValidWithBrokenDataHash(t *testing.T) {
	storage := &mocks.Storage{}
	bsrv := &mocks.BlockService{}
	storage.On("PutBlock", mock.AnythingOfType("*blockchain.Block")).Return(nil)
	storage.On("GetBlock", mock.AnythingOfType("common.Hash")).Return(nil, nil)
	storage.On("Contains", mock.AnythingOfType("common.Hash")).Return(false)
	storage.On("PutCurrentTopHeight", mock.AnythingOfType("int32")).Return(nil)
	bc := blockchain.CreateBlockchainFromGenesisBlock(&blockchain.Config{Storage: storage, BlockService: bsrv, Pool: mockPool(), Db: mockDB()})
	bc.GetGenesisBlock().SetQC(blockchain.CreateQuorumCertificate([]byte("valid"), bc.GetGenesisBlock().Header()))

	newBlock := bc.NewBlock(bc.GetGenesisBlock(), bc.GetGenesisCert(), []byte("Hello Hotstuff"))
	msg := newBlock.GetMessage()
	msg.Data = &pb.BlockData{Data: []byte("Hello from other side")}

	block := blockchain.CreateBlockFromMessage(msg)
	b, e := blockchain.IsValid(block)
	assert.EqualError(t, e, "data hash is not valid")
	assert.False(t, b)
}
func TestIsNotValidWithBrokenQCHash(t *testing.T) {
	storage := &mocks.Storage{}
	bsrv := &mocks.BlockService{}
	storage.On("PutBlock", mock.AnythingOfType("*blockchain.Block")).Return(nil)
	storage.On("GetBlock", mock.AnythingOfType("common.Hash")).Return(nil, nil)
	storage.On("Contains", mock.AnythingOfType("common.Hash")).Return(false)
	storage.On("PutCurrentTopHeight", mock.AnythingOfType("int32")).Return(nil)
	bc := blockchain.CreateBlockchainFromGenesisBlock(&blockchain.Config{Storage: storage, BlockService: bsrv, Pool: mockPool(), Db: mockDB()})
	bc.GetGenesisBlock().SetQC(blockchain.CreateQuorumCertificate([]byte("valid"), bc.GetGenesisBlock().Header()))

	newBlock := bc.NewBlock(bc.GetGenesisBlock(), bc.GetGenesisCert(), []byte("Hello Hotstuff"))
	msg := newBlock.GetMessage()
	msg.Cert.SignatureAggregate = []byte("Hello Signature")

	block := blockchain.CreateBlockFromMessage(msg)
	b, e := blockchain.IsValid(block)
	assert.EqualError(t, e, "QC hash is not valid")
	assert.False(t, b)
}
