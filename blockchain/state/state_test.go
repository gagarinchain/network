package state

import (
	"github.com/davecgh/go-spew/spew"
	cmn "github.com/gagarinchain/common"
	"github.com/gagarinchain/common/api"
	"github.com/gagarinchain/common/eth/common"
	"github.com/gagarinchain/common/eth/crypto"
	tx2 "github.com/gagarinchain/common/tx"
	"github.com/gagarinchain/network/storage"
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
	record := NewRecord(snapshot, nil, []common.Address{}, &cmn.NullBus{})
	Me = generate()
	from := generate()
	to := generate()

	tr := tx2.CreateTransaction(api.Payment, to, from, 0, big.NewInt(100), big.NewInt(1), []byte("123456"))
	r, err := record.ApplyTransaction(tr)
	assert.Equal(t, len(r), 0)
	assert.Equal(t, err, FutureTransactionError)
}

func TestSnapshot_ApplyTransactionInsufficientFunds(t *testing.T) {
	snapshot := NewSnapshot(crypto.Keccak256Hash([]byte("Aaaaaa Rexxar")), common.HexToAddress("0xCFC6243DFe7A9eB2E7f31a1f6b239Fcc13c26339"))
	record := NewRecord(snapshot, nil, []common.Address{}, &cmn.NullBus{})
	Me = generate()
	from := generate()
	to := generate()

	snapshot.Put(Me, NewAccount(0, big.NewInt(0)))
	snapshot.Put(from, NewAccount(0, big.NewInt(90)))

	tr := tx2.CreateTransaction(api.Payment, to, from, 1, big.NewInt(100), big.NewInt(1), []byte("aaaa Guldan"))
	r, err := record.ApplyTransaction(tr)
	assert.Equal(t, len(r), 0)

	assert.Equal(t, err, InsufficientFundsError)
}

func TestSnapshot_ApplyTransactionFutureNonce(t *testing.T) {
	snapshot := NewSnapshot(crypto.Keccak256Hash([]byte("Aaaaaa Rexxar")), common.HexToAddress("0xCFC6243DFe7A9eB2E7f31a1f6b239Fcc13c26339"))
	record := NewRecord(snapshot, nil, []common.Address{}, &cmn.NullBus{})
	Me = generate()
	from := generate()
	to := generate()

	snapshot.Put(Me, NewAccount(0, big.NewInt(0)))
	snapshot.Put(from, NewAccount(0, big.NewInt(100)))
	snapshot.Put(to, NewAccount(0, big.NewInt(90)))

	tr := tx2.CreateTransaction(api.Payment, to, from, 5, big.NewInt(100), big.NewInt(1), []byte("aaaa Guldan"))
	r, err := record.ApplyTransaction(tr)
	assert.Equal(t, len(r), 0)

	assert.Equal(t, err, FutureTransactionError)
}

func TestSnapshot_ApplyTransactionNoMe(t *testing.T) {
	snapshot := NewSnapshot(crypto.Keccak256Hash([]byte("Aaaaaa Rexxar")), common.HexToAddress("0xCFC6243DFe7A9eB2E7f31a1f6b239Fcc13c26339"))
	record := NewRecord(snapshot, nil, []common.Address{}, &cmn.NullBus{})
	Me = generate()
	from := generate()
	to := generate()

	snapshot.Put(from, NewAccount(0, big.NewInt(90)))

	tr := tx2.CreateTransaction(api.Payment, to, from, 1, big.NewInt(100), big.NewInt(1), []byte("aaaa Guldan"))
	r, err := record.ApplyTransaction(tr)
	assert.Equal(t, len(r), 0)

	assert.Equal(t, err, InsufficientFundsError)
}

func TestApplyTransactionToFromProposerAllDifferent(t *testing.T) {
	Me = generate()
	from := generate()
	to := generate()
	snapshot := NewSnapshot(crypto.Keccak256Hash([]byte("Aaaaaa Rexxar")), Me)
	rec := NewRecord(snapshot, nil, []common.Address{}, &cmn.NullBus{})

	snapshot.Put(Me, NewAccount(0, big.NewInt(0)))
	snapshot.Put(from, NewAccount(0, big.NewInt(111)))
	snapshot.Put(to, NewAccount(0, big.NewInt(90)))

	proofBefore := snapshot.RootProof()
	tr := tx2.CreateTransaction(api.Payment, to, from, 1, big.NewInt(100), big.NewInt(1), []byte("aaaa Guldan"))
	r, err := rec.ApplyTransaction(tr)
	assert.Equal(t, len(r), 2)

	var rPayment, rFee api.Receipt
	if r[0].To() == to { //R0 is payment, R1 is fee
		rPayment = r[0]
		rFee = r[1]
	} else {
		rPayment = r[1]
		rFee = r[0]
	}
	assert.Equal(t, rPayment.From(), from)
	assert.Equal(t, rPayment.FromValue(), big.NewInt(10))
	assert.Equal(t, rPayment.ToValue(), big.NewInt(190))
	assert.Equal(t, rFee.From(), from)
	assert.Equal(t, rFee.FromValue(), big.NewInt(10))
	assert.Equal(t, rFee.ToValue(), big.NewInt(1))
	assert.Equal(t, rFee.To(), Me)

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
func TestApplyTransactionMixed(t *testing.T) {
	Me = generate()
	from := generate()
	to := generate()
	snapshot := NewSnapshot(crypto.Keccak256Hash([]byte("Aaaaaa Rexxar")), Me)
	rec := NewRecord(snapshot, nil, []common.Address{}, &cmn.NullBus{})

	snapshot.Put(Me, NewAccount(0, big.NewInt(100)))
	snapshot.Put(from, NewAccount(0, big.NewInt(111)))
	snapshot.Put(to, NewAccount(0, big.NewInt(90)))

	proofBefore := snapshot.RootProof()
	txs := []api.Transaction{
		tx2.CreateTransaction(api.Payment, Me, from, 1, big.NewInt(10), big.NewInt(1), []byte("aaaa Guldan")),
		tx2.CreateTransaction(api.Payment, to, from, 2, big.NewInt(6), big.NewInt(2), []byte("aaaa Guldan")),
		tx2.CreateTransaction(api.Payment, to, from, 3, big.NewInt(12), big.NewInt(3), []byte("aaaa Guldan")),
		tx2.CreateTransaction(api.Payment, to, Me, 1, big.NewInt(34), big.NewInt(4), []byte("aaaa Guldan")),
		tx2.CreateTransaction(api.Payment, from, Me, 2, big.NewInt(11), big.NewInt(5), []byte("aaaa Guldan")),
	}

	var rs []api.Receipt
	for _, tr := range txs {
		r, err := rec.ApplyTransaction(tr)
		assert.Nil(t, err)
		rs = append(rs, r...)
	}
	assert.Equal(t, len(rs), 8)

	rPayment, rFee := separateFeeAndPayment(rs[0], rs[1], Me)
	assert.Equal(t, rPayment.Value(), big.NewInt(10))
	assert.Equal(t, rFee.Value(), big.NewInt(1))
	rPayment, rFee = separateFeeAndPayment(rs[2], rs[3], to)
	assert.Equal(t, rPayment.Value(), big.NewInt(6))
	assert.Equal(t, rFee.Value(), big.NewInt(2))
	rPayment, rFee = separateFeeAndPayment(rs[4], rs[5], to)
	assert.Equal(t, rPayment.Value(), big.NewInt(12))
	assert.Equal(t, rFee.Value(), big.NewInt(3))
	assert.Equal(t, rs[6].Value(), big.NewInt(34))
	assert.Equal(t, rs[7].Value(), big.NewInt(11))

	proofAfter := snapshot.RootProof()

	acc, _ := rec.Get(Me)
	assert.Equal(t, big.NewInt(100+(10+1)+2+3-34-11), acc.Balance())

	acc, _ = rec.Get(to)
	assert.Equal(t, big.NewInt(90+6+12+34), acc.Balance())

	acc, _ = rec.Get(from)
	assert.Equal(t, big.NewInt(111-(10+1)-(6+2)-(12+3)+11), acc.Balance())

	assert.NotEqual(t, proofBefore, proofAfter)
}

func separateFeeAndPayment(rs0 api.Receipt, rs1 api.Receipt, to common.Address) (api.Receipt, api.Receipt) {
	var rPayment, rFee api.Receipt
	if rs0.To() == to { //R0 is payment, R1 is fee
		rPayment = rs0
		rFee = rs1
	} else {
		rPayment = rs1
		rFee = rs0
	}
	return rPayment, rFee
}

func TestApplyTransactionFromProposerTheSame(t *testing.T) {
	Me = generate()
	to := generate()
	snapshot := NewSnapshot(crypto.Keccak256Hash([]byte("Aaaaaa Rexxar")), Me)
	rec := NewRecord(snapshot, nil, []common.Address{}, &cmn.NullBus{})

	snapshot.Put(Me, NewAccount(0, big.NewInt(111)))
	snapshot.Put(to, NewAccount(0, big.NewInt(90)))

	proofBefore := snapshot.RootProof()
	tr := tx2.CreateTransaction(api.Payment, to, Me, 1, big.NewInt(100), big.NewInt(1), []byte("aaaa Guldan"))
	r, err := rec.ApplyTransaction(tr)

	assert.Equal(t, 1, len(r))
	assert.Equal(t, r[0].From(), Me)
	assert.Equal(t, r[0].To(), to)
	assert.Equal(t, r[0].FromValue(), big.NewInt(11))
	assert.Equal(t, r[0].ToValue(), big.NewInt(190))
	assert.Equal(t, r[0].Value(), big.NewInt(100))

	proofAfter := snapshot.RootProof()

	assert.Nil(t, err)

	acc, _ := rec.Get(Me)
	assert.Equal(t, big.NewInt(11), acc.Balance())

	acc, _ = rec.Get(to)
	assert.Equal(t, big.NewInt(190), acc.Balance())

	assert.NotEqual(t, proofBefore, proofAfter)
}

func TestStateDB_New(t *testing.T) {
	valeera := crypto.Keccak256Hash([]byte("eeeeeee Valeera"))
	s, _ := storage.NewStorage("", nil)
	db := NewStateDB(s, []common.Address{}, &cmn.NullBus{})
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
	s, _ := storage.NewStorage("", nil)
	db := NewStateDB(s, []common.Address{}, &cmn.NullBus{})
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
	s, _ := storage.NewStorage("", nil)
	db := NewStateDB(s, []common.Address{}, &cmn.NullBus{})
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
	s, _ := storage.NewStorage("", nil)
	db := NewStateDB(s, []common.Address{}, &cmn.NullBus{})

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

	db2 := NewStateDB(s, []common.Address{}, &cmn.NullBus{})

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
