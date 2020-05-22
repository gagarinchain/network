package state

import (
	"github.com/davecgh/go-spew/spew"
	cmn "github.com/gagarinchain/common"
	"github.com/gagarinchain/common/api"
	"github.com/gagarinchain/common/eth/common"
	"github.com/gagarinchain/common/eth/crypto"
	"github.com/gagarinchain/network/blockchain/tx"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

func TestSnapshot_Empty(t *testing.T) {
	snapshot := NewSnapshot(crypto.Keccak256Hash([]byte("Aaaaaa Rexxar")), common.HexToAddress("0xCFC6243DFe7A9eB2E7f31a1f6b239Fcc13c26339"))

	proof := snapshot.RootProof()
	assert.Equal(t, common.BytesToHash([]byte{0}), proof)
}

func TestSnapshot_ApplyTransactionNoAccount(t *testing.T) {
	snapshot := NewSnapshot(crypto.Keccak256Hash([]byte("Aaaaaa Rexxar")), common.HexToAddress("0xCFC6243DFe7A9eB2E7f31a1f6b239Fcc13c26339"))
	record := NewRecord(snapshot, nil, &cmn.NullBus{})
	Me = generate()
	from := generate()
	to := generate()

	tr := tx.CreateTransaction(api.Payment, to, from, 0, big.NewInt(100), big.NewInt(1), []byte("123456"))
	err := record.ApplyTransaction(tr)

	assert.Equal(t, err, FutureTransactionError)
}

func TestSnapshot_ApplyTransactionInsufficientFunds(t *testing.T) {
	snapshot := NewSnapshot(crypto.Keccak256Hash([]byte("Aaaaaa Rexxar")), common.HexToAddress("0xCFC6243DFe7A9eB2E7f31a1f6b239Fcc13c26339"))
	record := NewRecord(snapshot, nil, &cmn.NullBus{})
	Me = generate()
	from := generate()
	to := generate()

	snapshot.Put(Me, NewAccount(0, big.NewInt(0)))
	snapshot.Put(from, NewAccount(0, big.NewInt(90)))

	tr := tx.CreateTransaction(api.Payment, to, from, 1, big.NewInt(100), big.NewInt(1), []byte("aaaa Guldan"))
	err := record.ApplyTransaction(tr)

	assert.Equal(t, err, InsufficientFundsError)
}

func TestSnapshot_ApplyTransactionFutureNonce(t *testing.T) {
	snapshot := NewSnapshot(crypto.Keccak256Hash([]byte("Aaaaaa Rexxar")), common.HexToAddress("0xCFC6243DFe7A9eB2E7f31a1f6b239Fcc13c26339"))
	record := NewRecord(snapshot, nil, &cmn.NullBus{})
	Me = generate()
	from := generate()
	to := generate()

	snapshot.Put(Me, NewAccount(0, big.NewInt(0)))
	snapshot.Put(from, NewAccount(0, big.NewInt(100)))
	snapshot.Put(to, NewAccount(0, big.NewInt(90)))

	tr := tx.CreateTransaction(api.Payment, to, from, 5, big.NewInt(100), big.NewInt(1), []byte("aaaa Guldan"))
	err := record.ApplyTransaction(tr)

	assert.Equal(t, err, FutureTransactionError)
}

func TestSnapshot_ApplyTransactionNoMe(t *testing.T) {
	snapshot := NewSnapshot(crypto.Keccak256Hash([]byte("Aaaaaa Rexxar")), common.HexToAddress("0xCFC6243DFe7A9eB2E7f31a1f6b239Fcc13c26339"))
	record := NewRecord(snapshot, nil, &cmn.NullBus{})
	Me = generate()
	from := generate()
	to := generate()

	snapshot.Put(from, NewAccount(0, big.NewInt(90)))

	tr := tx.CreateTransaction(api.Payment, to, from, 1, big.NewInt(100), big.NewInt(1), []byte("aaaa Guldan"))
	err := record.ApplyTransaction(tr)

	assert.Equal(t, err, InsufficientFundsError)
}

func TestSnapshot_ApplyTransactionSingleSnapshot(t *testing.T) {
	Me = generate()
	from := generate()
	to := generate()
	snapshot := NewSnapshot(crypto.Keccak256Hash([]byte("Aaaaaa Rexxar")), Me)
	rec := NewRecord(snapshot, nil, &cmn.NullBus{})

	snapshot.Put(Me, NewAccount(0, big.NewInt(0)))
	snapshot.Put(from, NewAccount(0, big.NewInt(111)))
	snapshot.Put(to, NewAccount(0, big.NewInt(90)))

	proofBefore := snapshot.RootProof()
	tr := tx.CreateTransaction(api.Payment, to, from, 1, big.NewInt(100), big.NewInt(1), []byte("aaaa Guldan"))
	err := rec.ApplyTransaction(tr)
	proofAfter := snapshot.RootProof()

	assert.Nil(t, err)

	acc, _ := rec.Get(Me)
	assert.Equal(t, big.NewInt(1), acc.Balance())

	acc, _ = rec.Get(to)
	assert.Equal(t, big.NewInt(190), acc.Balance())

	acc, _ = rec.Get(from)
	assert.Equal(t, big.NewInt(10), acc.Balance())

	assert.NotEqual(t, proofBefore, proofAfter)
}

func TestStateDB_New(t *testing.T) {
	valeera := crypto.Keccak256Hash([]byte("eeeeeee Valeera"))
	storage, _ := cmn.NewStorage("", nil)
	db := NewStateDB(storage, &cmn.NullBus{})
	db.Init(valeera, nil)

	sn, f := db.Get(valeera)
	assert.True(t, f)
	assert.NotNil(t, sn)
}

func TestStateDB_Create(t *testing.T) {
	Me = generate()

	valeera := crypto.Keccak256Hash([]byte("eeeeeee Valeera"))
	maiev := crypto.Keccak256Hash([]byte("eeeeeee Maiev"))
	guldan := crypto.Keccak256Hash([]byte("eeeeeee Guldan"))
	storage, _ := cmn.NewStorage("", nil)
	db := NewStateDB(storage, &cmn.NullBus{})
	db.Init(valeera, nil)
	maI, _ := db.Create(valeera, Me)
	ma := maI.(*RecordImpl)
	db.Commit(valeera, maiev)
	guI, _ := db.Create(valeera, Me)
	gu := guI.(*RecordImpl)
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

	assert.Equal(t, big.NewInt(111), fromMa.Balance())
	assert.Equal(t, big.NewInt(90), fromGu.Balance())
}

func TestStateDB_Release(t *testing.T) {
	Me = generate()

	valeera := crypto.Keccak256Hash([]byte("eeeeeee Valeera"))
	maiev := crypto.Keccak256Hash([]byte("eeeeeee Maiev"))
	guldan := crypto.Keccak256Hash([]byte("eeeeeee Guldan"))
	storage, _ := cmn.NewStorage("", nil)
	db := NewStateDB(storage, &cmn.NullBus{})
	db.Init(valeera, nil)
	ma, _ := db.Create(valeera, Me)
	db.Commit(valeera, maiev)
	gu, _ := db.Create(maiev, Me)
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
	db := NewStateDB(storage, &cmn.NullBus{})

	seed := make(map[common.Address]api.Account)
	a, b, c, d := generate(), generate(), generate(), generate()
	seed[a] = NewAccount(0, big.NewInt(10))
	seed[b] = NewAccount(1, big.NewInt(20))
	seed[c] = NewAccount(2, big.NewInt(30))

	valeera := crypto.Keccak256Hash([]byte("eeeeeee Valeera"))
	seedSnapshot := NewSnapshotWithAccounts(valeera, a, seed)

	if err := db.Init(valeera, seedSnapshot); err != nil {
		t.Error(err)
	}

	maiev := crypto.Keccak256Hash([]byte("eeeeeee Maiev"))
	maievRec, _ := db.Create(valeera, a)

	bforUpdateI, _ := maievRec.GetForUpdate(b)
	bforUpdate := bforUpdateI.(*AccountImpl)
	bforUpdate.nonce = 2
	bforUpdate.balance = big.NewInt(25)
	_ = maievRec.Update(b, bforUpdate)

	maievRec.Put(d, NewAccount(1, big.NewInt(45)))

	maievRec, _ = db.Commit(valeera, maiev)

	guldan := crypto.Keccak256Hash([]byte("eeeeeee Guldan"))
	guldanRecI, _ := db.Create(maiev, a)
	guldanRec := guldanRecI.(*RecordImpl)
	cforUpdateI, _ := guldanRec.GetForUpdate(c)
	cforUpdate := cforUpdateI.(*AccountImpl)
	cforUpdate.nonce = 1
	cforUpdate.balance = big.NewInt(33)
	_ = guldanRec.Update(c, cforUpdate)

	guldanRecI, _ = db.Commit(maiev, guldan)

	db2 := NewStateDB(storage, &cmn.NullBus{})

	guldanFromDb, f := db2.Get(guldan)
	if !f {
		t.Error("no record is found")
	}
	cFromDb, _ := guldanFromDb.Get(c)
	aFromDb, _ := guldanFromDb.Get(a)

	spew.Dump(guldanRec.Get(c))
	spew.Dump(cFromDb)
	assert.Equal(t, cFromDb.Balance(), big.NewInt(33))
	assert.Equal(t, aFromDb.Balance(), big.NewInt(10))
}

func generate() common.Address {
	pk, _ := crypto.GenerateKey()
	address := crypto.PubkeyToAddress(pk.PublicKey())
	return address
}
