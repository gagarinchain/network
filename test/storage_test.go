package test

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/gagarinchain/network/blockchain"
	"github.com/gagarinchain/network/common"
	"github.com/gagarinchain/network/hotstuff"
	"github.com/gagarinchain/network/mocks"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestStorageCreation(t *testing.T) {
	_, e := common.NewStorage("test.db", nil)
	if e != nil {
		t.Error(e)
	}
	if e := os.RemoveAll("test.db"); e != nil {
		t.Error(t, e)
	}
}

func TestStorageBlockAddition(t *testing.T) {
	storage, e := common.NewStorage("test.db", nil)
	if e != nil {
		t.Error(e)
	}
	defer cleanUpDb(t)

	service := &mocks.BlockService{}
	bpersister := &blockchain.BlockPersister{storage}
	cpersister := &blockchain.BlockchainPersister{storage}
	bc := blockchain.CreateBlockchainFromGenesisBlock(&blockchain.BlockchainConfig{
		ChainPersister: cpersister, BlockPerister: bpersister, BlockService: service, Pool: mockPool(), Db: mockDB(),
	})
	bc.GetGenesisBlock().SetQC(blockchain.CreateQuorumCertificate([]byte("valid"), bc.GetGenesisBlock().Header()))

	b := bc.NewBlock(bc.GetHead(), bc.GetGenesisCert(), []byte("random data"))
	e = bc.AddBlock(b)
	if e != nil {
		t.Error(e)
	}

	fromStorage, e := bpersister.Load(b.Header().Hash())

	assert.Equal(t, b, fromStorage)
	spew.Dump(storage.Stats())
}

func TestPutGetCurrentEpoch(t *testing.T) {
	s, e := common.NewStorage("test.db", nil)
	if e != nil {
		t.Error(e)
	}
	defer cleanUpDb(t)
	persister := &hotstuff.PacerPersister{s}

	if err := persister.PutCurrentEpoch(int32(15)); err != nil {
		t.Error(e)
	}

	val, e := persister.GetCurrentEpoch()
	if e != nil {
		t.Error(e)
	}

	assert.Equal(t, int32(15), val)
}

func TestPutGetCurrentTopHeight(t *testing.T) {
	s, e := common.NewStorage("test.db", nil)
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

func cleanUpDb(t *testing.T) {
	if e := os.RemoveAll("test.db"); e != nil {
		t.Error(t, e)
	}
}
