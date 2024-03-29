package test

import (
	"github.com/gagarinchain/common/api"
	"github.com/gagarinchain/common/eth/common"
	"github.com/gagarinchain/common/eth/crypto"
	"github.com/gagarinchain/common/protobuff"
	tx2 "github.com/gagarinchain/common/tx"
	"github.com/gagarinchain/network/blockchain"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

//Note that QC hash does not matter, since genesis.QC.header is genesis.header
func TestIsValidGenesisBlock(t *testing.T) {
	storage := SoftStorageMock()
	bpersister := &blockchain.BlockPersister{storage}
	cpersister := &blockchain.BlockchainPersister{storage}
	bc := blockchain.CreateBlockchainFromGenesisBlock(&blockchain.BlockchainConfig{
		BlockPerister: bpersister, ChainPersister: cpersister, Pool: mockPool(), Db: mockDB(), ProposerGetter: MockProposerForHeight(),
	})
	committee := mockCommittee(t)
	aggregate := mockSignatureAggregateValid(bc.GetGenesisBlock().Header().Hash().Bytes(), committee)
	headerValidator := &blockchain.HeaderValidator{}
	validator := blockchain.NewBlockValidator(committee, blockchain.NewTransactionValidator(committee), headerValidator)
	bc.UpdateGenesisBlockQC(blockchain.CreateQuorumCertificate(aggregate, bc.GetGenesisBlock().Header(), api.Empty))
	b, e := validator.IsValid(bc.GetGenesisBlock())
	assert.NoError(t, e)
	assert.True(t, b)

}

func TestIsValidBlock(t *testing.T) {
	storage := SoftStorageMock()
	bpersister := &blockchain.BlockPersister{storage}
	cpersister := &blockchain.BlockchainPersister{storage}
	bc := blockchain.CreateBlockchainFromGenesisBlock(&blockchain.BlockchainConfig{
		BlockPerister: bpersister, ChainPersister: cpersister, Pool: mockPool(), Db: mockDB(), ProposerGetter: MockProposerForHeight(),
	})
	committee := mockCommittee(t)
	aggregate := mockSignatureAggregateValid(bc.GetGenesisBlock().Header().Hash().Bytes(), committee)
	bc.UpdateGenesisBlockQC(blockchain.CreateQuorumCertificate(aggregate, bc.GetGenesisBlock().Header(), api.Empty))

	newBlock1, _ := bc.NewBlock(bc.GetGenesisBlock(), bc.GetGenesisCert(), []byte("Hello Hotstuff"))
	hash1 := newBlock1.Header().Hash()
	aggregate1 := mockSignatureAggregateValid(hash1.Bytes(), committee)
	certificate := blockchain.CreateQuorumCertificate(aggregate1, newBlock1.Header(), api.QRef)

	newBlock2, _ := bc.NewBlock(newBlock1, certificate, []byte("Hello Hotstuff2"))
	headerValidator := &blockchain.HeaderValidator{}
	b, e := blockchain.NewBlockValidator(committee, blockchain.NewTransactionValidator(committee), headerValidator).IsValid(newBlock2)
	assert.NoError(t, e)
	assert.True(t, b)
}

func TestIsValidBlockWithSignature(t *testing.T) {
	storage := SoftStorageMock()
	bpersister := &blockchain.BlockPersister{storage}
	cpersister := &blockchain.BlockchainPersister{storage}
	bc := blockchain.CreateBlockchainFromGenesisBlock(&blockchain.BlockchainConfig{
		BlockPerister: bpersister, ChainPersister: cpersister, Pool: mockPool(), Db: mockDB(), ProposerGetter: MockProposerForHeight(),
	})
	committee := mockCommittee(t)
	aggregate := mockSignatureAggregateValid(bc.GetGenesisBlock().Header().Hash().Bytes(), committee)
	bc.UpdateGenesisBlockQC(blockchain.CreateQuorumCertificate(aggregate, bc.GetGenesisBlock().Header(), api.Empty))

	newBlock1, _ := bc.NewBlock(bc.GetGenesisBlock(), bc.GetGenesisCert(), []byte("Hello Hotstuff"))
	newBlock1.AddTransaction(tx2.CreateTransaction(api.Payment, common.Address{}, common.Address{}, 1, big.NewInt(2), big.NewInt(1), nil))
	hash1 := newBlock1.Header().Hash()
	aggregate1 := mockSignatureAggregateValid(hash1.Bytes(), committee)
	certificate := blockchain.CreateQuorumCertificate(aggregate1, newBlock1.Header(), api.QRef)
	newBlock1.SetSignature(certificate.SignatureAggregate())

	headerValidator := &blockchain.HeaderValidator{}
	b, e := blockchain.NewBlockValidator(committee, blockchain.NewTransactionValidator(committee), headerValidator).IsValid(newBlock1)
	assert.NoError(t, e)
	assert.True(t, b)
}

func TestIsValidBlockWithTransaction(t *testing.T) {
	storage := SoftStorageMock()
	bpersister := &blockchain.BlockPersister{storage}
	cpersister := &blockchain.BlockchainPersister{storage}
	bc := blockchain.CreateBlockchainFromGenesisBlock(&blockchain.BlockchainConfig{
		BlockPerister: bpersister, ChainPersister: cpersister, Pool: mockPool(), Db: mockDB(), ProposerGetter: MockProposerForHeight(),
	})
	committee := mockCommittee(t)
	aggregate := mockSignatureAggregateValid(bc.GetGenesisBlock().Header().Hash().Bytes(), committee)
	bc.UpdateGenesisBlockQC(blockchain.CreateQuorumCertificate(aggregate, bc.GetGenesisBlock().Header(), api.Empty))

	newBlock1, _ := bc.NewBlock(bc.GetGenesisBlock(), bc.GetGenesisCert(), []byte("Hello Hotstuff"))
	hash1 := newBlock1.Header().Hash()
	aggregate1 := mockSignatureAggregateValid(hash1.Bytes(), committee)
	certificate := blockchain.CreateQuorumCertificate(aggregate1, newBlock1.Header(), api.QRef)

	newBlock2, _ := bc.NewBlock(newBlock1, certificate, []byte("Hello Hotstuff2"))
	newBlock2.AddTransaction(tx2.CreateTransaction(api.Payment, common.Address{}, common.Address{}, 1, big.NewInt(2), big.NewInt(1), nil))

	headerValidator := &blockchain.HeaderValidator{}
	b, e := blockchain.NewBlockValidator(committee, blockchain.NewTransactionValidator(committee), headerValidator).IsValid(newBlock2)
	assert.NoError(t, e)
	assert.True(t, b)
}

func TestIsNotValidBlockWithTransaction(t *testing.T) {
	storage := SoftStorageMock()
	bpersister := &blockchain.BlockPersister{storage}
	cpersister := &blockchain.BlockchainPersister{storage}
	bc := blockchain.CreateBlockchainFromGenesisBlock(&blockchain.BlockchainConfig{
		BlockPerister: bpersister, ChainPersister: cpersister, Pool: mockPool(), Db: mockDB(), ProposerGetter: MockProposerForHeight(),
	})
	committee := mockCommittee(t)
	aggregate := mockSignatureAggregateValid(bc.GetGenesisBlock().Header().Hash().Bytes(), committee)
	bc.UpdateGenesisBlockQC(blockchain.CreateQuorumCertificate(aggregate, bc.GetGenesisBlock().Header(), api.Empty))

	newBlock1, _ := bc.NewBlock(bc.GetGenesisBlock(), bc.GetGenesisCert(), []byte("Hello Hotstuff"))
	hash1 := newBlock1.Header().Hash()
	aggregate1 := mockSignatureAggregateValid(hash1.Bytes(), committee)
	certificate := blockchain.CreateQuorumCertificate(aggregate1, newBlock1.Header(), api.QRef)

	newBlock2, _ := bc.NewBlock(newBlock1, certificate, []byte("Hello Hotstuff2"))
	newBlock2.AddTransaction(tx2.CreateTransaction(api.Payment, common.Address{}, common.Address{}, 1, big.NewInt(2), big.NewInt(0), nil))

	headerValidator := &blockchain.HeaderValidator{}
	b, e := blockchain.NewBlockValidator(committee, blockchain.NewTransactionValidator(committee), headerValidator).IsValid(newBlock2)
	assert.Error(t, e)
	assert.False(t, b)
}

func TestIsNotValidWithBrokenHash(t *testing.T) {
	storage := SoftStorageMock()
	bpersister := &blockchain.BlockPersister{storage}
	cpersister := &blockchain.BlockchainPersister{storage}
	bc := blockchain.CreateBlockchainFromGenesisBlock(&blockchain.BlockchainConfig{
		BlockPerister: bpersister, ChainPersister: cpersister, Pool: mockPool(), Db: mockDB(), ProposerGetter: MockProposerForHeight(),
	})
	committee := mockCommittee(t)
	aggregate := mockSignatureAggregateValid(bc.GetGenesisBlock().Header().Hash().Bytes(), committee)
	bc.GetGenesisBlock().SetQC(blockchain.CreateQuorumCertificate(aggregate, bc.GetGenesisBlock().Header(), api.Empty))

	newBlock, _ := bc.NewBlock(bc.GetGenesisBlock(), bc.GetGenesisCert(), []byte("Hello Hotstuff"))

	msg := newBlock.GetMessage()
	msg.Header.Hash = crypto.Keccak256([]byte("Hello from other side"))

	headerValidator := &blockchain.HeaderValidator{}
	validator := blockchain.NewBlockValidator(committee, blockchain.NewTransactionValidator(committee), headerValidator)
	message, _ := blockchain.CreateBlockFromMessage(msg)
	b, e := validator.IsValid(message)
	assert.EqualError(t, e, "block hash is not valid")
	assert.False(t, b)
}

func TestIsNotValidWithBrokenDataHash(t *testing.T) {
	storage := SoftStorageMock()

	bpersister := &blockchain.BlockPersister{storage}
	cpersister := &blockchain.BlockchainPersister{storage}
	bc := blockchain.CreateBlockchainFromGenesisBlock(&blockchain.BlockchainConfig{
		BlockPerister: bpersister, ChainPersister: cpersister, Pool: mockPool(), Db: mockDB(), ProposerGetter: MockProposerForHeight(),
	})
	committee := mockCommittee(t)
	aggregate := mockSignatureAggregateValid(bc.GetGenesisBlock().Header().Hash().Bytes(), committee)
	bc.GetGenesisBlock().SetQC(blockchain.CreateQuorumCertificate(aggregate, bc.GetGenesisBlock().Header(), api.Empty))

	newBlock, _ := bc.NewBlock(bc.GetGenesisBlock(), bc.GetGenesisCert(), []byte("Hello Hotstuff"))
	msg := newBlock.GetMessage()
	msg.Data = &pb.BlockData{Data: []byte("Hello from other side")}

	block, _ := blockchain.CreateBlockFromMessage(msg)
	headerValidator := &blockchain.HeaderValidator{}
	validator := blockchain.NewBlockValidator(committee, blockchain.NewTransactionValidator(committee), headerValidator)
	b, e := validator.IsValid(block)
	assert.EqualError(t, e, "data hash is not valid")
	assert.False(t, b)
}

func TestIsNotValidWithBrokenQRefHash(t *testing.T) {
	storage := SoftStorageMock()

	bpersister := &blockchain.BlockPersister{storage}
	cpersister := &blockchain.BlockchainPersister{storage}
	bc := blockchain.CreateBlockchainFromGenesisBlock(&blockchain.BlockchainConfig{
		BlockPerister: bpersister, ChainPersister: cpersister, Pool: mockPool(), Db: mockDB(), ProposerGetter: MockProposerForHeight(),
	})
	committee := mockCommittee(t)
	aggregate := mockSignatureAggregateValid(bc.GetGenesisBlock().Header().Hash().Bytes(), committee)
	bc.GetGenesisBlock().SetQC(blockchain.CreateQuorumCertificate(aggregate, bc.GetGenesisBlock().Header(), api.Empty))

	newBlock, _ := bc.NewBlock(bc.GetGenesisBlock(), bc.GetGenesisCert(), []byte("Hello Hotstuff"))
	msg := newBlock.GetMessage()
	msg.Cert.Header.Height = 2
	block, _ := blockchain.CreateBlockFromMessage(msg)

	headerValidator := &blockchain.HeaderValidator{}
	validator := blockchain.NewBlockValidator(committee, blockchain.NewTransactionValidator(committee), headerValidator)
	b, e := validator.IsValid(block)
	assert.EqualError(t, e, "QC.Qref hash is not valid")
	assert.False(t, b)
}

func TestIsNotValidWithBrokenQCSignature(t *testing.T) {
	storage := SoftStorageMock()
	bpersister := &blockchain.BlockPersister{storage}
	cpersister := &blockchain.BlockchainPersister{storage}
	bc := blockchain.CreateBlockchainFromGenesisBlock(&blockchain.BlockchainConfig{
		BlockPerister: bpersister, ChainPersister: cpersister, Pool: mockPool(), Db: mockDB(), ProposerGetter: MockProposerForHeight(),
	})
	committee := mockCommittee(t)
	aggregate := mockSignatureAggregateValid(bc.GetGenesisBlock().Header().Hash().Bytes(), committee)
	bc.GetGenesisBlock().SetQC(blockchain.CreateQuorumCertificate(aggregate, bc.GetGenesisBlock().Header(), api.Empty))

	newBlock1, _ := bc.NewBlock(bc.GetGenesisBlock(), bc.GetGenesisCert(), []byte("Hello Hotstuff"))
	aggregate1 := mockSignatureAggregateNotValid(newBlock1.Header().Hash().Bytes(), committee)
	newBlock2, _ := bc.NewBlock(newBlock1, blockchain.CreateQuorumCertificate(aggregate1, newBlock1.Header(), api.QRef), []byte("Hello Hotstuff2"))

	headerValidator := &blockchain.HeaderValidator{}
	b, e := blockchain.NewBlockValidator(committee, blockchain.NewTransactionValidator(committee), headerValidator).IsValid(newBlock2)
	assert.False(t, b)
	assert.EqualError(t, e, "QC is not valid")
}
func TestIsNotValidWithNotEnpoughQCSignature(t *testing.T) {
	storage := SoftStorageMock()
	bpersister := &blockchain.BlockPersister{storage}
	cpersister := &blockchain.BlockchainPersister{storage}
	bc := blockchain.CreateBlockchainFromGenesisBlock(&blockchain.BlockchainConfig{
		BlockPerister: bpersister, ChainPersister: cpersister, Pool: mockPool(), Db: mockDB(), ProposerGetter: MockProposerForHeight(),
	})
	committee := mockCommittee(t)
	aggregate := mockSignatureAggregateValid(bc.GetGenesisBlock().Header().Hash().Bytes(), committee)
	bc.GetGenesisBlock().SetQC(blockchain.CreateQuorumCertificate(aggregate, bc.GetGenesisBlock().Header(), api.Empty))

	newBlock1, _ := bc.NewBlock(bc.GetGenesisBlock(), bc.GetGenesisCert(), []byte("Hello Hotstuff"))
	aggregate1 := mockSignatureAggregateNotEnough(newBlock1.Header().Hash().Bytes(), committee)
	newBlock2, _ := bc.NewBlock(newBlock1, blockchain.CreateQuorumCertificate(aggregate1, newBlock1.Header(), api.QRef), []byte("Hello Hotstuff2"))

	headerValidator := &blockchain.HeaderValidator{}
	b, e := blockchain.NewBlockValidator(committee, blockchain.NewTransactionValidator(committee), headerValidator).IsValid(newBlock2)
	assert.False(t, b)
	assert.EqualError(t, e, "QC contains less than 2f + 1 signatures")
}

func TestIsNotValidWithEmptyQCSignature(t *testing.T) {
	storage := SoftStorageMock()
	bpersister := &blockchain.BlockPersister{storage}
	cpersister := &blockchain.BlockchainPersister{storage}
	bc := blockchain.CreateBlockchainFromGenesisBlock(&blockchain.BlockchainConfig{
		BlockPerister: bpersister, ChainPersister: cpersister, Pool: mockPool(), Db: mockDB(), ProposerGetter: MockProposerForHeight(),
	})
	committee := mockCommittee(t)
	aggregate := mockSignatureAggregateValid(bc.GetGenesisBlock().Header().Hash().Bytes(), committee)
	bc.GetGenesisBlock().SetQC(blockchain.CreateQuorumCertificate(aggregate, bc.GetGenesisBlock().Header(), api.Empty))

	newBlock1, _ := bc.NewBlock(bc.GetGenesisBlock(), bc.GetGenesisCert(), []byte("Hello Hotstuff"))
	newBlock2, _ := bc.NewBlock(newBlock1, blockchain.CreateQuorumCertificate(crypto.EmptyAggregateSignatures(), newBlock1.Header(), api.QRef),
		[]byte("Hello Hotstuff2"))

	headerValidator := &blockchain.HeaderValidator{}
	b, e := blockchain.NewBlockValidator(committee, blockchain.NewTransactionValidator(committee), headerValidator).IsValid(newBlock2)
	assert.EqualError(t, e, "QC contains less than 2f + 1 signatures")
	assert.False(t, b)
}

func TestIsValidWithNoSignatureAndEmptySignature(t *testing.T) {
	storage := SoftStorageMock()
	bpersister := &blockchain.BlockPersister{storage}
	cpersister := &blockchain.BlockchainPersister{storage}
	bc := blockchain.CreateBlockchainFromGenesisBlock(&blockchain.BlockchainConfig{
		BlockPerister: bpersister, ChainPersister: cpersister, Pool: mockPool(), Db: mockDB(), ProposerGetter: MockProposerForHeight(),
	})
	committee := mockCommittee(t)
	aggregate := mockSignatureAggregateValid(bc.GetGenesisBlock().Header().Hash().Bytes(), committee)
	bc.GetGenesisBlock().SetQC(blockchain.CreateQuorumCertificate(aggregate, bc.GetGenesisBlock().Header(), api.Empty))

	newBlock1, _ := bc.NewBlock(bc.GetGenesisBlock(), bc.GetGenesisCert(), []byte("Hello Hotstuff"))
	newBlock2, _ := bc.NewBlock(newBlock1, blockchain.CreateQuorumCertificate(crypto.EmptyAggregateSignatures(), newBlock1.Header(), api.QRef),
		[]byte("Hello Hotstuff2"))
	bc.AddBlock(newBlock1)
	bc.AddBlock(newBlock2)

	b1 := bc.GetBlockByHash(newBlock1.Header().Hash())

	headerValidator := &blockchain.HeaderValidator{}
	b, e := blockchain.NewBlockValidator(committee, blockchain.NewTransactionValidator(committee), headerValidator).IsValid(b1)
	assert.Nil(t, e)
	assert.True(t, b)
}
