package state

import (
	"github.com/gagarinchain/common/eth/common"
	pb "github.com/gagarinchain/common/protobuff"
	"math/big"
)

type ReceiptImpl struct {
	txHash    common.Hash
	txIndex   int32
	from      common.Address
	to        common.Address
	value     *big.Int
	toValue   *big.Int
	fromValue *big.Int
}

func (r *ReceiptImpl) ToStorageProto() *pb.Receipt {
	return &pb.Receipt{
		TxHash:    r.txHash.Bytes(),
		From:      r.from.Bytes(),
		To:        r.to.Bytes(),
		Value:     r.value.Int64(),
		ToValue:   r.toValue.Int64(),
		FromValue: r.fromValue.Int64(),
	}
}

//TODO fill real tx index
func ReceiptFromStorage(r *pb.Receipt) *ReceiptImpl {
	return &ReceiptImpl{
		txHash:    common.BytesToHash(r.TxHash),
		txIndex:   0,
		from:      common.BytesToAddress(r.From),
		to:        common.BytesToAddress(r.To),
		value:     big.NewInt(r.Value),
		toValue:   big.NewInt(r.ToValue),
		fromValue: big.NewInt(r.FromValue),
	}
}

func (r *ReceiptImpl) FromValue() *big.Int {
	return r.fromValue
}

func (r *ReceiptImpl) ToValue() *big.Int {
	return r.toValue
}

func NewReceipt(txHash common.Hash, txIndex int32, from common.Address, to common.Address, value *big.Int, toValue *big.Int, fromValue *big.Int) *ReceiptImpl {
	return &ReceiptImpl{txHash: txHash, txIndex: txIndex, from: from, to: to, value: value, toValue: toValue, fromValue: fromValue}
}

func (r *ReceiptImpl) Value() *big.Int {
	return r.value
}

func (r *ReceiptImpl) To() common.Address {
	return r.to
}

func (r *ReceiptImpl) From() common.Address {
	return r.from
}

func (r *ReceiptImpl) TxIndex() int32 {
	return r.txIndex
}

func (r *ReceiptImpl) TxHash() common.Hash {
	return r.txHash
}
