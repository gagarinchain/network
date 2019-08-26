package blockchain

import (
	"bytes"
	"errors"
	"github.com/gagarinchain/network/common/eth/common"
	"github.com/gagarinchain/network/common/eth/crypto"
	"github.com/gagarinchain/network/common/protobuff"
	"github.com/gagarinchain/network/common/tx"
	"math/big"
)

var (
	ValueNotValid             = errors.New("balance is not valid")
	FeeNotValid               = errors.New("fee is not valid")
	SettlementAddressNotValid = errors.New("settlement address is not valid")
	OracleProofNotValid       = errors.New("oracle proof is not valid")
	CustodianProofNotValid    = errors.New("custodian proof is not valid")
	AggregateProofNotValid    = errors.New("aggregate proof is not valid")
)

type TransactionValidator struct {
	committee []common.Address
}

func NewTransactionValidator(committee []common.Address) *TransactionValidator {
	return &TransactionValidator{committee: committee}
}

func (v *TransactionValidator) IsValid(entity interface{}) (bool, error) {
	t := entity.(*tx.Transaction)
	switch t.TxType() {
	case tx.Payment:
		if !v.validateValue(t) {
			return false, ValueNotValid
		}
		if !v.validateFee(t) {
			return false, FeeNotValid
		}
	case tx.Settlement:
		if !v.validateValue(t) {
			return false, ValueNotValid
		}

		if t.Fee().Cmp(big.NewInt(tx.DefaultSettlementReward)) <= 0 {
			return false, FeeNotValid
		}

		if !bytes.Equal(t.To().Bytes(), common.HexToAddress(tx.SettlementAddressHex).Bytes()) {
			return false, SettlementAddressNotValid
		}
		isValidOracle := true //make remote grpc
		if !isValidOracle {
			return false, OracleProofNotValid
		}
	case tx.Agreement:
		if t.Value().Cmp(big.NewInt(0)) != 0 {
			return false, ValueNotValid
		}
		if !v.validateFee(t) {
			return false, FeeNotValid
		}
		pub, e := crypto.SigToPub(crypto.Keccak256(t.To().Bytes()), t.Data())
		if e != nil {
			return false, CustodianProofNotValid
		}
		a := crypto.PubkeyToAddress(*pub)

		if !bytes.Equal(a.Bytes(), t.From().Bytes()) {
			return false, CustodianProofNotValid
		}
	case tx.Proof:
		if t.Value().Cmp(big.NewInt(0)) != 0 {
			return false, ValueNotValid
		}
		if !v.validateFee(t) {
			return false, FeeNotValid
		}

		if len(t.Data()) < 65 {
			return false, AggregateProofNotValid
		}

		addresses := map[common.Address]struct{}{}
		for _, a := range v.committee {
			addresses[a] = struct{}{}
		}

		provers, e := t.RecoverProver()
		if e != nil {
			return false, e
		}

		for _, p := range provers {
			delete(addresses, p)
		}

		if len(addresses) >= len(v.committee)/3+1 {
			return false, AggregateProofNotValid
		}
	}

	return true, nil
}

func (v *TransactionValidator) validateFee(t *tx.Transaction) bool {
	return t.Fee().Cmp(big.NewInt(0)) > 0
}

func (v *TransactionValidator) validateValue(t *tx.Transaction) bool {
	return t.Value().Cmp(big.NewInt(0)) > 0
}

func (v *TransactionValidator) Supported(mType pb.Message_MessageType) bool {
	return mType == pb.Message_TRANSACTION
}

func (v *TransactionValidator) GetId() interface{} {
	return "TransactionValidator"
}
