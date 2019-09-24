package test

import (
	"github.com/gagarinchain/network/blockchain"
	"github.com/gagarinchain/network/common/eth/crypto"
	"github.com/gagarinchain/network/common/protobuff"
	"github.com/gagarinchain/network/mocks"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

//Note that QC hash does not matter, since genesis.QC.header is genesis.header
func TestIsValidGenesisBlock(t *testing.T) {
	storage := SoftStorageMock()
	bsrv := &mocks.BlockService{}
	bpersister := &blockchain.BlockPersister{storage}
	cpersister := &blockchain.BlockchainPersister{storage}
	bc := blockchain.CreateBlockchainFromGenesisBlock(&blockchain.BlockchainConfig{
		BlockPerister: bpersister, ChainPersister: cpersister, BlockService: bsrv, Pool: mockPool(), Db: mockDB(), ProposerGetter: MockProposerForHeight(),
	})
	bc.GetGenesisBlock().SetQC(blockchain.CreateQuorumCertificate(crypto.EmptyAggregateSignatures(), bc.GetGenesisBlock().Header()))
	b, e := blockchain.IsValid(bc.GetGenesisBlock())
	assert.NoError(t, e)
	assert.True(t, b)

}

func TestIsValidBlock(t *testing.T) {
	storage := SoftStorageMock()
	bsrv := &mocks.BlockService{}
	bpersister := &blockchain.BlockPersister{storage}
	cpersister := &blockchain.BlockchainPersister{storage}
	bc := blockchain.CreateBlockchainFromGenesisBlock(&blockchain.BlockchainConfig{
		BlockPerister: bpersister, ChainPersister: cpersister, BlockService: bsrv, Pool: mockPool(), Db: mockDB(), ProposerGetter: MockProposerForHeight(),
	})
	bc.GetGenesisBlock().SetQC(blockchain.CreateQuorumCertificate(crypto.EmptyAggregateSignatures(), bc.GetGenesisBlock().Header()))

	newBlock := bc.NewBlock(bc.GetGenesisBlock(), bc.GetGenesisCert(), []byte("Hello Hotstuff"))
	b, e := blockchain.IsValid(newBlock)
	assert.NoError(t, e)
	assert.True(t, b)
}
func TestIsNotValidWithBrokenHash(t *testing.T) {
	storage := SoftStorageMock()
	bsrv := &mocks.BlockService{}
	bpersister := &blockchain.BlockPersister{storage}
	cpersister := &blockchain.BlockchainPersister{storage}
	bc := blockchain.CreateBlockchainFromGenesisBlock(&blockchain.BlockchainConfig{
		BlockPerister: bpersister, ChainPersister: cpersister, BlockService: bsrv, Pool: mockPool(), Db: mockDB(), ProposerGetter: MockProposerForHeight(),
	})
	bc.GetGenesisBlock().SetQC(blockchain.CreateQuorumCertificate(crypto.EmptyAggregateSignatures(), bc.GetGenesisBlock().Header()))

	newBlock := bc.NewBlock(bc.GetGenesisBlock(), bc.GetGenesisCert(), []byte("Hello Hotstuff"))

	msg := newBlock.GetMessage()
	msg.Header.Hash = crypto.Keccak256([]byte("Hello from other side"))

	b, e := blockchain.IsValid(blockchain.CreateBlockFromMessage(msg))
	assert.EqualError(t, e, "block hash is not valid")
	assert.False(t, b)
}

func TestIsNotValidWithBrokenDataHash(t *testing.T) {
	storage := SoftStorageMock()
	bsrv := &mocks.BlockService{}

	bpersister := &blockchain.BlockPersister{storage}
	cpersister := &blockchain.BlockchainPersister{storage}
	bc := blockchain.CreateBlockchainFromGenesisBlock(&blockchain.BlockchainConfig{
		BlockPerister: bpersister, ChainPersister: cpersister, BlockService: bsrv, Pool: mockPool(), Db: mockDB(), ProposerGetter: MockProposerForHeight(),
	})
	bc.GetGenesisBlock().SetQC(blockchain.CreateQuorumCertificate(crypto.EmptyAggregateSignatures(), bc.GetGenesisBlock().Header()))

	newBlock := bc.NewBlock(bc.GetGenesisBlock(), bc.GetGenesisCert(), []byte("Hello Hotstuff"))
	msg := newBlock.GetMessage()
	msg.Data = &pb.BlockData{Data: []byte("Hello from other side")}

	block := blockchain.CreateBlockFromMessage(msg)
	b, e := blockchain.IsValid(block)
	assert.EqualError(t, e, "data hash is not valid")
	assert.False(t, b)
}
func TestIsNotValidWithBrokenQCHash(t *testing.T) {
	storage := SoftStorageMock()
	bsrv := &mocks.BlockService{}

	bpersister := &blockchain.BlockPersister{storage}
	cpersister := &blockchain.BlockchainPersister{storage}
	bc := blockchain.CreateBlockchainFromGenesisBlock(&blockchain.BlockchainConfig{
		BlockPerister: bpersister, ChainPersister: cpersister, BlockService: bsrv, Pool: mockPool(), Db: mockDB(), ProposerGetter: MockProposerForHeight(),
	})
	bc.GetGenesisBlock().SetQC(blockchain.CreateQuorumCertificate(crypto.EmptyAggregateSignatures(), bc.GetGenesisBlock().Header()))

	newBlock := bc.NewBlock(bc.GetGenesisBlock(), bc.GetGenesisCert(), []byte("Hello Hotstuff"))
	msg := newBlock.GetMessage()
	s := crypto.AggregateSignatures(big.NewInt(1), 1, nil)
	msg.Cert.SignatureAggregate = s.ToProto()

	block := blockchain.CreateBlockFromMessage(msg)
	b, e := blockchain.IsValid(block)
	assert.EqualError(t, e, "QC hash is not valid")
	assert.False(t, b)
}
