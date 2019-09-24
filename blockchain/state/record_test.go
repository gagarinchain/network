package state

import (
	"github.com/gagarinchain/network/common/eth/common"
	"github.com/gagarinchain/network/common/eth/crypto"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

func TestNewRecordFirst(t *testing.T) {
	firstBlockHash := common.BytesToHash(crypto.Keccak256([]byte("genesis block")))
	firstProposer := generate()
	somebody := generate()
	snapshot := NewSnapshot(firstBlockHash, firstProposer)
	snapshot.Put(somebody, NewAccount(0, big.NewInt(5)))
	record := NewRecord(snapshot, nil)

	acc, f := record.Get(somebody)
	assert.True(t, f)
	assert.Equal(t, big.NewInt(5), acc.balance)
}

func TestNewRecordWithParent(t *testing.T) {
	firstBlockHash := common.BytesToHash(crypto.Keccak256([]byte("genesis block")))
	firstProposer := generate()
	somebody := generate()
	snapshot := NewSnapshot(firstBlockHash, firstProposer)
	snapshot.Put(somebody, NewAccount(0, big.NewInt(5)))
	firstRecord := NewRecord(snapshot, nil)

	secondBlockHash := common.BytesToHash(crypto.Keccak256([]byte("second block")))
	secondProposer := generate()
	anotherSnapshot := NewSnapshot(secondBlockHash, secondProposer)
	somebody2 := generate()
	anotherSnapshot.Put(somebody2, NewAccount(2, big.NewInt(10)))
	secondRecord := NewRecord(anotherSnapshot, firstRecord)

	acc, f := secondRecord.Get(somebody)
	assert.True(t, f)
	assert.Equal(t, big.NewInt(5), acc.balance)

	acc2, f := secondRecord.Get(somebody2)
	assert.True(t, f)
	assert.Equal(t, big.NewInt(10), acc2.balance)
}
