package blockchain

import (
	"crypto/ecdsa"
	"github.com/gagarinchain/network/common/eth/common"
	"github.com/gagarinchain/network/common/eth/crypto"
	"github.com/gagarinchain/network/common/tx"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

func TestPaymentValidationBadFee(t *testing.T) {
	tran := tx.CreateTransaction(tx.Payment, GenerateAddress(), GenerateAddress(), 0, big.NewInt(1), big.NewInt(0), nil)

	validator := NewTransactionValidator([]common.Address{GenerateAddress()})
	_, e := validator.IsValid(tran)
	assert.Equal(t, FeeNotValid, e)
}

func TestPaymentValidationBadValue(t *testing.T) {
	tran := tx.CreateTransaction(tx.Payment, GenerateAddress(), GenerateAddress(), 0, big.NewInt(-1), big.NewInt(1), nil)

	validator := NewTransactionValidator([]common.Address{GenerateAddress()})
	_, e := validator.IsValid(tran)
	assert.Equal(t, ValueNotValid, e)
}
func TestSettlementValidationBadValue(t *testing.T) {
	tran := tx.CreateTransaction(tx.Settlement, common.HexToAddress(tx.SettlementAddressHex), GenerateAddress(), 0, big.NewInt(0), big.NewInt(15), nil)

	validator := NewTransactionValidator([]common.Address{GenerateAddress()})
	_, e := validator.IsValid(tran)
	assert.Equal(t, ValueNotValid, e)
}
func TestSettlementValidationBadFee(t *testing.T) {
	tran := tx.CreateTransaction(tx.Settlement, common.HexToAddress(tx.SettlementAddressHex), GenerateAddress(), 0, big.NewInt(1), big.NewInt(5), nil)

	validator := NewTransactionValidator([]common.Address{GenerateAddress()})
	_, e := validator.IsValid(tran)
	assert.Equal(t, FeeNotValid, e)
}
func TestSettlementValidationBadAddress(t *testing.T) {
	tran := tx.CreateTransaction(tx.Settlement, GenerateAddress(), GenerateAddress(), 0, big.NewInt(145), big.NewInt(15), nil)

	validator := NewTransactionValidator([]common.Address{GenerateAddress()})
	_, e := validator.IsValid(tran)
	assert.Equal(t, SettlementAddressNotValid, e)
}
func TestAgreementValidationBadValue(t *testing.T) {
	tran := tx.CreateTransaction(tx.Agreement, GenerateAddress(), GenerateAddress(), 0, big.NewInt(10), big.NewInt(15), nil)

	validator := NewTransactionValidator([]common.Address{GenerateAddress()})
	_, e := validator.IsValid(tran)
	assert.Equal(t, ValueNotValid, e)
}
func TestAgreementValidationBadFee(t *testing.T) {
	tran := tx.CreateTransaction(tx.Agreement, GenerateAddress(), GenerateAddress(), 0, big.NewInt(0), big.NewInt(0), nil)

	validator := NewTransactionValidator([]common.Address{GenerateAddress()})
	_, e := validator.IsValid(tran)
	assert.Equal(t, FeeNotValid, e)
}
func TestAgreementValidationBadSignature(t *testing.T) {
	tran := tx.CreateTransaction(tx.Agreement, GenerateAddress(), GenerateAddress(), 0, big.NewInt(0), big.NewInt(10), nil)

	validator := NewTransactionValidator([]common.Address{GenerateAddress()})
	_, e := validator.IsValid(tran)
	assert.Equal(t, CustodianProofNotValid, e)
}
func TestAgreementValid(t *testing.T) {
	pk, _ := crypto.GenerateKey()
	from := crypto.PubkeyToAddress(*pk.Public().(*ecdsa.PublicKey))
	to := GenerateAddress()

	tran := tx.CreateTransaction(tx.Settlement, common.HexToAddress(tx.SettlementAddressHex), GenerateAddress(), 0, big.NewInt(100),
		big.NewInt(tx.DefaultSettlementReward+5), nil)
	hash := tran.Hash()
	settleAddress := common.BytesToAddress(hash.Bytes()[12:])
	sig, _ := crypto.Sign(crypto.Keccak256(settleAddress.Bytes()), pk)

	tran2 := tx.CreateTransaction(tx.Agreement, settleAddress, from, 0, big.NewInt(0), big.NewInt(10), sig)

	validator := NewTransactionValidator([]common.Address{to})
	bol, _ := validator.IsValid(tran2)
	assert.True(t, bol)
}

func TestProofValidationBadValue(t *testing.T) {
	tran := tx.CreateTransaction(tx.Proof, GenerateAddress(), GenerateAddress(), 0, big.NewInt(10), big.NewInt(15), nil)

	validator := NewTransactionValidator([]common.Address{GenerateAddress()})
	_, e := validator.IsValid(tran)
	assert.Equal(t, ValueNotValid, e)
}
func TestProofValidationBadFee(t *testing.T) {
	tran := tx.CreateTransaction(tx.Proof, GenerateAddress(), GenerateAddress(), 0, big.NewInt(0), big.NewInt(0), nil)

	validator := NewTransactionValidator([]common.Address{GenerateAddress()})
	_, e := validator.IsValid(tran)
	assert.Equal(t, FeeNotValid, e)
}

func TestProofValidationNotValidSignature(t *testing.T) {
	tran := tx.CreateTransaction(tx.Proof, GenerateAddress(), GenerateAddress(), 0, big.NewInt(0), big.NewInt(21), nil)

	validator := NewTransactionValidator([]common.Address{GenerateAddress()})
	_, e := validator.IsValid(tran)
	assert.Equal(t, AggregateProofNotValid, e)
}

func TestProofValid(t *testing.T) {
	var committee []common.Address
	var proofs []byte

	pk, _ := crypto.GenerateKey()
	from := crypto.PubkeyToAddress(*pk.Public().(*ecdsa.PublicKey))
	to := GenerateAddress()

	tran := tx.CreateTransaction(tx.Proof, to, from, 0, big.NewInt(0), big.NewInt(10), nil)
	hash := tran.Hash()

	for i := 0; i < 10; i++ {
		pk, _ := crypto.GenerateKey()
		from := crypto.PubkeyToAddress(*pk.Public().(*ecdsa.PublicKey))
		committee = append(committee, from)
		sig, _ := crypto.Sign(hash.Bytes(), pk)
		proofs = append(proofs, sig...)
	}
	tran2 := tx.CreateTransaction(tx.Proof, to, from, 0, big.NewInt(0), big.NewInt(10), proofs)
	tran3 := tx.CreateTransaction(tx.Proof, to, from, 0, big.NewInt(0), big.NewInt(10), proofs[:325])
	validator := NewTransactionValidator(committee)

	//enough
	_, e := validator.IsValid(tran2)
	//not enough
	_, e = validator.IsValid(tran3)
	assert.Equal(t, AggregateProofNotValid, e)
}

func GenerateAddress() common.Address {
	pk, _ := crypto.GenerateKey()
	address := crypto.PubkeyToAddress(*pk.Public().(*ecdsa.PublicKey))
	return address
}
