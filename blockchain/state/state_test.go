package state

import (
	"crypto/ecdsa"
	"github.com/davecgh/go-spew/spew"
	cmn "github.com/gagarinchain/network/common"
	"github.com/gagarinchain/network/common/eth/common"
	"github.com/gagarinchain/network/common/eth/crypto"
	"github.com/gagarinchain/network/common/tx"
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

	tr := tx.CreateTransaction(tx.Payment, to, from, 0, big.NewInt(100), big.NewInt(1), []byte("123456"))
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

	tr := tx.CreateTransaction(tx.Payment, to, from, 1, big.NewInt(100), big.NewInt(1), []byte("aaaa Guldan"))
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

	tr := tx.CreateTransaction(tx.Payment, to, from, 5, big.NewInt(100), big.NewInt(1), []byte("aaaa Guldan"))
	err := snapshot.ApplyTransaction(tr)

	assert.Equal(t, err, FutureTransactionError)
}

func TestSnapshot_ApplyTransactionNoMe(t *testing.T) {
	snapshot := NewSnapshot(crypto.Keccak256Hash([]byte("Aaaaaa Rexxar")))
	Me = generate()
	from := generate()
	to := generate()

	snapshot.Put(from, NewAccount(0, big.NewInt(90)))

	tr := tx.CreateTransaction(tx.Payment, to, from, 1, big.NewInt(100), big.NewInt(1), []byte("aaaa Guldan"))
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
	tr := tx.CreateTransaction(tx.Payment, to, from, 1, big.NewInt(100), big.NewInt(1), []byte("aaaa Guldan"))
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
	storage, _ := cmn.NewStorage("", nil)
	db := NewStateDB(storage)
	db.Init(valeera, nil)

	sn, f := db.Get(valeera)
	assert.True(t, f)
	assert.NotNil(t, sn)
}

func TestStateDB_Create(t *testing.T) {
	valeera := crypto.Keccak256Hash([]byte("eeeeeee Valeera"))
	maiev := crypto.Keccak256Hash([]byte("eeeeeee Maiev"))
	guldan := crypto.Keccak256Hash([]byte("eeeeeee Guldan"))
	storage, _ := cmn.NewStorage("", nil)
	db := NewStateDB(storage)
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
	storage, _ := cmn.NewStorage("", nil)
	db := NewStateDB(storage)
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

func TestStorageIntegration(t *testing.T) {
	storage, _ := cmn.NewStorage("", nil)
	db := NewStateDB(storage)

	seed := make(map[common.Address]*Account)
	a, b, c, d := generate(), generate(), generate(), generate()
	seed[a] = NewAccount(0, big.NewInt(10))
	seed[b] = NewAccount(1, big.NewInt(20))
	seed[c] = NewAccount(2, big.NewInt(30))

	valeera := crypto.Keccak256Hash([]byte("eeeeeee Valeera"))
	seedSnapshot := NewSnapshotWithAccounts(valeera, seed)

	if err := db.Init(valeera, seedSnapshot); err != nil {
		t.Error(err)
	}

	maiev := crypto.Keccak256Hash([]byte("eeeeeee Maiev"))
	maievSnapshot, _ := db.Create(valeera)
	maievSnapshot.Put(b, NewAccount(2, big.NewInt(25)))
	maievSnapshot.Put(d, NewAccount(1, big.NewInt(45)))
	maievSnapshot, _ = db.Commit(valeera, maiev)

	guldan := crypto.Keccak256Hash([]byte("eeeeeee Guldan"))
	guldanSnap, _ := db.Create(maiev)
	guldanSnap.Put(c, NewAccount(1, big.NewInt(33)))
	guldanSnap, _ = db.Commit(maiev, guldan)

	db2 := NewStateDB(storage)

	guldanFromDb, _ := db2.Get(guldan)
	cFromDb, _ := guldanFromDb.Get(c)
	aFromDb, _ := guldanFromDb.Get(a)
	assert.Equal(t, cFromDb.balance, big.NewInt(33))
	assert.Equal(t, aFromDb.balance, big.NewInt(10))
}

func generate() common.Address {
	pk, _ := crypto.GenerateKey()
	address := crypto.PubkeyToAddress(*pk.Public().(*ecdsa.PublicKey))
	spew.Dump(address.Hex())
	return address
}
