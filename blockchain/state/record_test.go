package state

import (
	common2 "github.com/gagarinchain/common"
	"github.com/gagarinchain/common/eth/common"
	"github.com/gagarinchain/common/eth/crypto"
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
	record := NewRecord(snapshot, nil, []common.Address{}, &common2.NullBus{})

	acc, f := record.Get(somebody)
	assert.True(t, f)
	assert.Equal(t, big.NewInt(5), acc.Balance())
}

func TestNewRecordWithParent(t *testing.T) {
	firstBlockHash := common.BytesToHash(crypto.Keccak256([]byte("genesis block")))
	firstProposer := generate()
	somebody := generate()
	snapshot := NewSnapshot(firstBlockHash, firstProposer)
	snapshot.Put(somebody, NewAccount(0, big.NewInt(5)))
	firstRecord := NewRecord(snapshot, nil, []common.Address{}, &common2.NullBus{})

	secondBlockHash := common.BytesToHash(crypto.Keccak256([]byte("second block")))
	secondProposer := generate()
	anotherSnapshot := NewSnapshot(secondBlockHash, secondProposer)
	somebody2 := generate()
	anotherSnapshot.Put(somebody2, NewAccount(2, big.NewInt(10)))
	secondRecord := NewRecord(anotherSnapshot, firstRecord, []common.Address{}, &common2.NullBus{})

	acc, f := secondRecord.Get(somebody)
	assert.True(t, f)
	assert.Equal(t, big.NewInt(5), acc.Balance())

	acc2, f := secondRecord.Get(somebody2)
	assert.True(t, f)
	assert.Equal(t, big.NewInt(10), acc2.Balance())
}
