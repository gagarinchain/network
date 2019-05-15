package blockchain

import "github.com/poslibp2p/common/protobuff"

type TransactionValidator struct {
}

func (tx *TransactionValidator) IsValid(entity interface{}) (bool, error) {
	return false, nil
}

func (tx *TransactionValidator) Supported(mType pb.Message_MessageType) bool {
	return mType == pb.Message_TRANSACTION
}

func (tx *TransactionValidator) GetId() interface{} {
	return "TransactionValidator"
}
