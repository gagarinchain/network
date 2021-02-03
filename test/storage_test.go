package test

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/gagarinchain/common/api"
	"github.com/gagarinchain/common/eth/crypto"
	"github.com/gagarinchain/network/blockchain"
	"github.com/gagarinchain/network/hotstuff"
	storage "github.com/gagarinchain/network/storage"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestStorageCreation(t *testing.T) {
	_, e := storage.NewStorage("test.db", nil)
	if e != nil {
		t.Error(e)
	}
	if e := os.RemoveAll("test.db"); e != nil {
		t.Error(t, e)
	}
}

func TestStorageBlockAddition(t *testing.T) {
	store, e := storage.NewStorage("test.db", nil)
	if e != nil {
		t.Error(e)
	}
	defer cleanUpDb(t)

	bpersister := &blockchain.BlockPersister{store}
	cpersister := &blockchain.BlockchainPersister{store}
	bc := blockchain.CreateBlockchainFromGenesisBlock(&blockchain.BlockchainConfig{
		ChainPersister: cpersister, BlockPerister: bpersister, Pool: mockPool(), Db: mockDB(), ProposerGetter: MockProposerForHeight(),
	})
	bc.GetGenesisBlock().SetQC(blockchain.CreateQuorumCertificate(crypto.EmptyAggregateSignatures(), bc.GetGenesisBlock().Header(), api.QRef))

	b, _ := bc.NewBlock(bc.GetHead(), bc.GetGenesisCert(), []byte("random data"))
	_, e = bc.AddBlock(b)
	if e != nil {
		t.Error(e)
	}

	fromStorage, e := bpersister.Load(b.Header().Hash())

	assert.Equal(t, b, fromStorage)
	spew.Dump(store.Stats())
}

func TestPutGetCurrentTopHeight(t *testing.T) {
	s, e := storage.NewStorage("test.db", nil)
	if e != nil {
		t.Error(e)
	}
	defer cleanUpDb(t)
	cpersister := &blockchain.BlockchainPersister{s}

	if err := cpersister.PutCurrentTopHeight(int32(11231235)); err != nil {
		t.Error(e)
	}

	val, e := cpersister.GetCurrentTopHeight()
	if e != nil {
		t.Error(e)
	}

	assert.Equal(t, int32(11231235), val)
}

func TestHCPersistence(t *testing.T) {
	s, e := storage.NewStorage("test.db", nil)
	if e != nil {
		t.Error(e)
	}
	defer cleanUpDb(t)

	bpersister := &blockchain.BlockPersister{s}
	cpersister := &blockchain.BlockchainPersister{s}

	p := hotstuff.ProtocolPersister{Storage: s}
	bc := blockchain.CreateBlockchainFromGenesisBlock(&blockchain.BlockchainConfig{
		ChainPersister: cpersister, BlockPerister: bpersister, Pool: mockPool(), Db: mockDB(), ProposerGetter: MockProposerForHeight(),
	})
	cert := blockchain.CreateQuorumCertificate(crypto.EmptyAggregateSignatures(), bc.GetGenesisBlock().Header(), api.QRef)

	e = p.PutHC(cert)
	if e != nil {
		t.Error(e)
	}

	hc, e := p.GetHC()
	if e != nil {
		t.Error(e)
	}
	assert.Equal(t, cert, hc)

	sc := blockchain.CreateSynchronizeCertificate(crypto.EmptyAggregateSignatures(), 2)

	e = p.PutHC(sc)
	if e != nil {
		t.Error(e)
	}

	fromdb, e := p.GetHC()
	if e != nil {
		t.Error(e)
	}
	assert.Equal(t, sc, fromdb)

}

func cleanUpDb(t *testing.T) {
	if e := os.RemoveAll("test.db"); e != nil {
		t.Error(t, e)
	}
}
