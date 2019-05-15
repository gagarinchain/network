package state

import (
	"crypto/ecdsa"
	"github.com/poslibp2p/common/eth/common"
	"github.com/poslibp2p/common/eth/crypto"
	"github.com/poslibp2p/common/tx"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

func TestSnapshot_Empty(t *testing.T) {
	snapshot := NewSnapshot(crypto.Keccak256Hash([]byte("Aaaaaa Rexxar")))

	hash := snapshot.Proof()
	assert.Equal(t, crypto.Keccak256Hash([]byte("")).Hex(), hash.Hex())
}

func TestSnapshot_ApplyTransactionNoAccount(t *testing.T) {
	snapshot := NewSnapshot(crypto.Keccak256Hash([]byte("Aaaaaa Rexxar")))
	Me = generate()
	from := generate()
	to := generate()

	tr := tx.CreateTransaction(tx.Payment, to, from, 0, big.NewInt(100), big.NewInt(1), []byte(""), crypto.Keccak256Hash([]byte("123456")))
	err := snapshot.ApplyTransaction(tr)

	assert.Equal(t, err, FutureTransactionError)
}

func TestSnapshot_ApplyTransactionInsufficientFunds(t *testing.T) {
	snapshot := NewSnapshot(crypto.Keccak256Hash([]byte("Aaaaaa Rexxar")))
	Me = generate()
	from := generate()
	to := generate()

	snapshot.Put(Me, NewAccount(0, big.NewInt(0)))
	snapshot.Put(from, NewAccount(0, big.NewInt(90)))

	tr := tx.CreateTransaction(tx.Payment, to, from, 1, big.NewInt(100), big.NewInt(1), []byte(""), crypto.Keccak256Hash([]byte("aaaa Guldan")))
	err := snapshot.ApplyTransaction(tr)

	assert.Equal(t, err, InsufficientFundsError)
}

func TestSnapshot_ApplyTransactionFutureNonce(t *testing.T) {
	snapshot := NewSnapshot(crypto.Keccak256Hash([]byte("Aaaaaa Rexxar")))
	Me = generate()
	from := generate()
	to := generate()

	snapshot.Put(Me, NewAccount(0, big.NewInt(0)))
	snapshot.Put(from, NewAccount(0, big.NewInt(100)))
	snapshot.Put(to, NewAccount(0, big.NewInt(90)))

	tr := tx.CreateTransaction(tx.Payment, to, from, 5, big.NewInt(100), big.NewInt(1), []byte(""), crypto.Keccak256Hash([]byte("aaaa Guldan")))
	err := snapshot.ApplyTransaction(tr)

	assert.Equal(t, err, FutureTransactionError)
}

func TestSnapshot_ApplyTransactionNoMe(t *testing.T) {
	snapshot := NewSnapshot(crypto.Keccak256Hash([]byte("Aaaaaa Rexxar")))
	Me = generate()
	from := generate()
	to := generate()

	snapshot.Put(from, NewAccount(0, big.NewInt(90)))

	tr := tx.CreateTransaction(tx.Payment, to, from, 1, big.NewInt(100), big.NewInt(1), []byte(""), crypto.Keccak256Hash([]byte("aaaa Guldan")))
	err := snapshot.ApplyTransaction(tr)

	assert.Equal(t, err, InsufficientFundsError)
}

func TestSnapshot_ApplyTransaction(t *testing.T) {
	snapshot := NewSnapshot(crypto.Keccak256Hash([]byte("Aaaaaa Rexxar")))
	Me = generate()
	from := generate()
	to := generate()

	snapshot.Put(Me, NewAccount(0, big.NewInt(0)))
	snapshot.Put(from, NewAccount(0, big.NewInt(111)))
	snapshot.Put(to, NewAccount(0, big.NewInt(90)))

	proofBefore := snapshot.Proof()
	tr := tx.CreateTransaction(tx.Payment, to, from, 1, big.NewInt(100), big.NewInt(1), []byte(""), crypto.Keccak256Hash([]byte("aaaa Guldan")))
	err := snapshot.ApplyTransaction(tr)
	proofAfter := snapshot.Proof()

	assert.Nil(t, err)

	acc, _ := snapshot.Get(Me)
	assert.Equal(t, big.NewInt(1), acc.balance)

	acc, _ = snapshot.Get(to)
	assert.Equal(t, big.NewInt(190), acc.balance)

	acc, _ = snapshot.Get(from)
	assert.Equal(t, big.NewInt(10), acc.balance)

	assert.NotEqual(t, proofBefore, proofAfter)
}

func TestStateDB_New(t *testing.T) {
	valeera := crypto.Keccak256Hash([]byte("eeeeeee Valeera"))
	db := NewStateDB()
	db.Init(valeera, nil)

	sn, f := db.Get(valeera)
	assert.True(t, f)
	assert.NotNil(t, sn)
}

func TestStateDB_Create(t *testing.T) {
	valeera := crypto.Keccak256Hash([]byte("eeeeeee Valeera"))
	maiev := crypto.Keccak256Hash([]byte("eeeeeee Maiev"))
	guldan := crypto.Keccak256Hash([]byte("eeeeeee Guldan"))
	db := NewStateDB()
	db.Init(valeera, nil)
	ma, _ := db.Create(valeera)
	db.Commit(valeera, maiev)
	gu, _ := db.Create(valeera)
	db.Commit(valeera, guldan)

	Me = generate()
	from := generate()
	to := generate()

	ma.Put(Me, NewAccount(0, big.NewInt(0)))
	ma.Put(from, NewAccount(0, big.NewInt(111)))
	gu.Put(from, NewAccount(0, big.NewInt(90)))
	gu.Put(to, NewAccount(0, big.NewInt(30)))

	m, _ := db.Get(maiev)
	fromMa, _ := m.Get(from)
	g, _ := db.Get(guldan)
	fromGu, _ := g.Get(from)

	assert.Equal(t, big.NewInt(111), fromMa.balance)
	assert.Equal(t, big.NewInt(90), fromGu.balance)
}

func TestStateDB_Release(t *testing.T) {
	valeera := crypto.Keccak256Hash([]byte("eeeeeee Valeera"))
	maiev := crypto.Keccak256Hash([]byte("eeeeeee Maiev"))
	guldan := crypto.Keccak256Hash([]byte("eeeeeee Guldan"))
	db := NewStateDB()
	db.Init(valeera, nil)
	ma, _ := db.Create(valeera)
	db.Commit(valeera, maiev)
	gu, _ := db.Create(maiev)
	db.Commit(maiev, guldan)

	Me = generate()
	from := generate()
	to := generate()

	ma.Put(Me, NewAccount(0, big.NewInt(0)))
	ma.Put(from, NewAccount(0, big.NewInt(111)))
	gu.Put(from, NewAccount(0, big.NewInt(90)))
	gu.Put(to, NewAccount(0, big.NewInt(30)))

	e := db.Release(maiev)

	assert.NoError(t, e)

	_, f := db.Get(valeera)
	assert.True(t, f)
	_, fm := db.Get(maiev)
	assert.False(t, fm)
	_, fg := db.Get(guldan)
	assert.False(t, fg)

}

func generate() common.Address {
	pk, _ := crypto.GenerateKey()
	address := crypto.PubkeyToAddress(*pk.Public().(*ecdsa.PublicKey))
	return address
}
