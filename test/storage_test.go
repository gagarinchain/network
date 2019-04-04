package test

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/poslibp2p/blockchain"
	"github.com/poslibp2p/mocks"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestStorageCreation(t *testing.T) {
	_, e := blockchain.NewStorage("test.db", nil)
	if e != nil {
		t.Error(e)
	}
	if e := os.RemoveAll("test.db"); e != nil {
		t.Error(t, e)
	}
}

func TestStorageBlockAddition(t *testing.T) {
	storage, e := blockchain.NewStorage("test.db", nil)
	if e != nil {
		t.Error(e)
	}
	defer cleanUpDb(t)

	service := &mocks.BlockService{}
	bc := blockchain.CreateBlockchainFromGenesisBlock(storage, service)
	bc.GetGenesisBlock().SetQC(blockchain.CreateQuorumCertificate([]byte("valid"), bc.GetGenesisBlock().Header()))

	b := bc.NewBlock(bc.GetHead(), bc.GetGenesisCert(), []byte("random data"))
	e = bc.AddBlock(b)
	if e != nil {
		t.Error(e)
	}

	fromStorage, e := storage.GetBlock(b.Header().Hash())

	assert.Equal(t, b, fromStorage)
	spew.Dump(storage.Stats())
}

func TestPutGetCurrentEpoch(t *testing.T) {
	s, e := blockchain.NewStorage("test.db", nil)
	if e != nil {
		t.Error(e)
	}
	defer cleanUpDb(t)

	if err := s.PutCurrentEpoch(int32(15)); err != nil {
		t.Error(e)
	}

	val, e := s.GetCurrentEpoch()
	if e != nil {
		t.Error(e)
	}

	assert.Equal(t, int32(15), val)
}

func TestPutGetCurrentTopHeight(t *testing.T) {
	s, e := blockchain.NewStorage("test.db", nil)
	if e != nil {
		t.Error(e)
	}
	defer cleanUpDb(t)

	if err := s.PutCurrentTopHeight(int32(11231235)); err != nil {
		t.Error(e)
	}

	val, e := s.GetCurrentTopHeight()
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
