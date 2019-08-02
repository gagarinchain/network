package blockchain

import (
	"github.com/gagarinchain/network/common/protobuff"
	tx2 "github.com/gagarinchain/network/common/tx"
	"math/big"
)

type TransactionValidator struct {
}

func NewTransactionValidator() *TransactionValidator {
	return &TransactionValidator{}
}

func (tx *TransactionValidator) IsValid(entity interface{}) (bool, error) {
	t := entity.(*tx2.Transaction)
	return t.Value().Cmp(big.NewInt(0)) != 0, nil
}

func (tx *TransactionValidator) Supported(mType pb.Message_MessageType) bool {
	return mType == pb.Message_TRANSACTION
}

func (tx *TransactionValidator) GetId() interface{} {
	return "TransactionValidator"
}
