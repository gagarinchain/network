package blockchain

import (
	"bytes"
	"errors"
	cmn "github.com/gagarinchain/network/common"
	"github.com/gagarinchain/network/common/eth/common"
	"github.com/gagarinchain/network/common/eth/crypto"
	"github.com/gagarinchain/network/common/protobuff"
	"github.com/gagarinchain/network/common/tx"
	"github.com/gogo/protobuf/proto"
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
	committee []*cmn.Peer
}

func NewTransactionValidator(committee []*cmn.Peer) *TransactionValidator {
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

		sPb := &pb.Signature{}
		if err := proto.Unmarshal(t.Data(), sPb); err != nil {
			return false, err
		}
		sign := crypto.SignatureFromProto(sPb)
		if sign == nil {
			return false, CustodianProofNotValid
		}
		b := crypto.Verify(crypto.Keccak256(t.To().Bytes()), sign)
		if !b {
			return false, CustodianProofNotValid
		}
		a := crypto.PubkeyToAddress(crypto.NewPublicKey(sign.Pub()))

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

		if len(t.Data()) < 96 {
			return false, AggregateProofNotValid
		}

		aggrPb := &pb.SignatureAggregate{}
		if err := proto.Unmarshal(t.Data(), aggrPb); err != nil {
			return false, err
		}
		aggregate := crypto.AggregateFromProto(aggrPb)

		if aggregate.N() < 2*len(v.committee)/3+1 {
			return false, AggregateProofNotValid
		}
		var pubs []*crypto.PublicKey
		for _, p := range v.committee {
			pubs = append(pubs, p.PublicKey())
		}

		isValid := aggregate.IsValid(crypto.Keccak256(t.To().Bytes()), pubs)
		if !isValid {
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
