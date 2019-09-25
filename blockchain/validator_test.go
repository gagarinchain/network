package blockchain

import (
	"github.com/davecgh/go-spew/spew"
	common2 "github.com/gagarinchain/network/common"
	"github.com/gagarinchain/network/common/eth/common"
	"github.com/gagarinchain/network/common/eth/crypto"
	"github.com/gagarinchain/network/common/tx"
	"github.com/gogo/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

func TestPaymentValidationBadFee(t *testing.T) {
	tran := tx.CreateTransaction(tx.Payment, GeneratePeer().GetAddress(), GeneratePeer().GetAddress(), 0, big.NewInt(1), big.NewInt(0), nil)

	validator := NewTransactionValidator([]*common2.Peer{GeneratePeer()})
	_, e := validator.IsValid(tran)
	assert.Equal(t, FeeNotValid, e)
}

func TestPaymentValidationBadValue(t *testing.T) {
	tran := tx.CreateTransaction(tx.Payment, GeneratePeer().GetAddress(), GeneratePeer().GetAddress(), 0, big.NewInt(-1), big.NewInt(1), nil)

	validator := NewTransactionValidator([]*common2.Peer{GeneratePeer()})
	_, e := validator.IsValid(tran)
	assert.Equal(t, ValueNotValid, e)
}
func TestSettlementValidationBadValue(t *testing.T) {
	tran := tx.CreateTransaction(tx.Settlement, common.HexToAddress(tx.SettlementAddressHex), GeneratePeer().GetAddress(), 0, big.NewInt(0), big.NewInt(15), nil)

	validator := NewTransactionValidator([]*common2.Peer{GeneratePeer()})
	_, e := validator.IsValid(tran)
	assert.Equal(t, ValueNotValid, e)
}
func TestSettlementValidationBadFee(t *testing.T) {
	tran := tx.CreateTransaction(tx.Settlement, common.HexToAddress(tx.SettlementAddressHex), GeneratePeer().GetAddress(), 0, big.NewInt(1), big.NewInt(5), nil)

	validator := NewTransactionValidator([]*common2.Peer{GeneratePeer()})
	_, e := validator.IsValid(tran)
	assert.Equal(t, FeeNotValid, e)
}
func TestSettlementValidationBadAddress(t *testing.T) {
	tran := tx.CreateTransaction(tx.Settlement, GeneratePeer().GetAddress(), GeneratePeer().GetAddress(), 0, big.NewInt(145), big.NewInt(15), nil)

	validator := NewTransactionValidator([]*common2.Peer{GeneratePeer()})
	_, e := validator.IsValid(tran)
	assert.Equal(t, SettlementAddressNotValid, e)
}
func TestAgreementValidationBadValue(t *testing.T) {
	tran := tx.CreateTransaction(tx.Agreement, GeneratePeer().GetAddress(), GeneratePeer().GetAddress(), 0, big.NewInt(10), big.NewInt(15), nil)

	validator := NewTransactionValidator([]*common2.Peer{GeneratePeer()})
	_, e := validator.IsValid(tran)
	assert.Equal(t, ValueNotValid, e)
}
func TestAgreementValidationBadFee(t *testing.T) {
	tran := tx.CreateTransaction(tx.Agreement, GeneratePeer().GetAddress(), GeneratePeer().GetAddress(), 0, big.NewInt(0), big.NewInt(0), nil)

	validator := NewTransactionValidator([]*common2.Peer{GeneratePeer()})
	_, e := validator.IsValid(tran)
	assert.Equal(t, FeeNotValid, e)
}
func TestAgreementValidationBadSignature(t *testing.T) {

	tran := tx.CreateTransaction(tx.Agreement, GeneratePeer().GetAddress(), GeneratePeer().GetAddress(), 0, big.NewInt(0), big.NewInt(10), nil)

	validator := NewTransactionValidator([]*common2.Peer{GeneratePeer()})
	_, e := validator.IsValid(tran)
	assert.Equal(t, CustodianProofNotValid, e)
}
func TestAgreementValid(t *testing.T) {
	pk, _ := crypto.GenerateKey()
	from := crypto.PubkeyToAddress(pk.PublicKey())

	tran := tx.CreateTransaction(tx.Settlement, common.HexToAddress(tx.SettlementAddressHex), GeneratePeer().GetAddress(), 0, big.NewInt(100),
		big.NewInt(tx.DefaultSettlementReward+5), nil)
	hash := tran.Hash()
	settleAddress := common.BytesToAddress(hash.Bytes()[12:])
	sig := crypto.Sign(crypto.Keccak256(settleAddress.Bytes()), pk)
	bytes, _ := proto.Marshal(sig.ToProto())
	tran2 := tx.CreateTransaction(tx.Agreement, settleAddress, from, 0, big.NewInt(0), big.NewInt(10), bytes)

	validator := NewTransactionValidator([]*common2.Peer{GeneratePeer()})
	bol, _ := validator.IsValid(tran2)
	assert.True(t, bol)
}

func TestProofValidationBadValue(t *testing.T) {
	tran := tx.CreateTransaction(tx.Proof, GeneratePeer().GetAddress(), GeneratePeer().GetAddress(), 0, big.NewInt(10), big.NewInt(15), nil)

	validator := NewTransactionValidator([]*common2.Peer{GeneratePeer()})
	_, e := validator.IsValid(tran)
	assert.Equal(t, ValueNotValid, e)
}
func TestProofValidationBadFee(t *testing.T) {
	tran := tx.CreateTransaction(tx.Proof, GeneratePeer().GetAddress(), GeneratePeer().GetAddress(), 0, big.NewInt(0), big.NewInt(0), nil)

	validator := NewTransactionValidator([]*common2.Peer{GeneratePeer()})
	_, e := validator.IsValid(tran)
	assert.Equal(t, FeeNotValid, e)
}

func TestProofValidationNotValidSignature(t *testing.T) {
	tran := tx.CreateTransaction(tx.Proof, GeneratePeer().GetAddress(), GeneratePeer().GetAddress(), 0, big.NewInt(0), big.NewInt(21), nil)

	validator := NewTransactionValidator([]*common2.Peer{GeneratePeer()})
	_, e := validator.IsValid(tran)
	assert.Equal(t, AggregateProofNotValid, e)
}

func TestProofValid(t *testing.T) {
	var committee []*common2.Peer
	proofs := make(map[common.Address]*crypto.Signature)
	proofs2 := make(map[common.Address]*crypto.Signature)
	var signs []*crypto.Signature
	var signs2 []*crypto.Signature

	pk, _ := crypto.GenerateKey()
	from := crypto.PubkeyToAddress(pk.PublicKey())
	to := GeneratePeer().GetAddress()

	tran := tx.CreateTransaction(tx.Proof, to, from, 0, big.NewInt(0), big.NewInt(10), nil)
	hash := tran.Hash()

	for i := 0; i < 10; i++ {
		p := GeneratePeer()
		committee = append(committee, p)
		sig := crypto.Sign(hash.Bytes(), p.GetPrivateKey())
		signs = append(signs, sig)
		proofs[p.GetAddress()] = sig
		if i < 6 {
			signs2 = append(signs2, sig)
			proofs2[p.GetAddress()] = sig
		}
	}

	bitmap, _ := GetBitmap(committee, proofs)
	aggregate := crypto.AggregateSignatures(bitmap, signs)
	bytes, _ := proto.Marshal(aggregate.ToProto())
	tran2 := tx.CreateTransaction(tx.Proof, to, from, 0, big.NewInt(0), big.NewInt(10), bytes)

	bitmap2, _ := GetBitmap(committee, proofs2)
	aggregate2 := crypto.AggregateSignatures(bitmap2, signs2)
	bytes2, _ := proto.Marshal(aggregate2.ToProto())
	tran3 := tx.CreateTransaction(tx.Proof, to, from, 0, big.NewInt(0), big.NewInt(10), bytes2)

	validator := NewTransactionValidator(committee)

	//enough
	spew.Dump(aggregate)
	_, e := validator.IsValid(tran2)
	//not enough
	_, e = validator.IsValid(tran3)
	assert.Equal(t, AggregateProofNotValid, e)
}

func GeneratePeer() *common2.Peer {
	pk, _ := crypto.GenerateKey()
	return common2.CreatePeer(pk.PublicKey(), pk, &peer.AddrInfo{})
}

func GetBitmap(committee []*common2.Peer, src map[common.Address]*crypto.Signature) (*big.Int, int) {
	bitmap := big.NewInt(0)
	n := 0

	for i, p := range committee {
		if _, f := src[p.GetAddress()]; f {
			bitmap.SetBit(bitmap, i, 1)
			n++
		}
	}
	return bitmap, n
}
