package tx

import (
	"errors"
	"github.com/gogo/protobuf/proto"
	"github.com/op/go-logging"
	"github.com/poslibp2p/common/eth/common"
	"github.com/poslibp2p/common/eth/crypto"
	"github.com/poslibp2p/common/protobuff"
	"math/big"
)

type Type int

const (
	Payment Type = iota
)

type Iterator interface {
	Next() *Transaction
}

type Transaction struct {
	txType    Type
	to        common.Address
	from      common.Address
	nonce     uint64
	value     *big.Int
	fee       *big.Int
	signature []byte
	data      []byte
	hash      common.Hash

	serialized []byte
}

func (tx *Transaction) Serialized() []byte {
	if tx.serialized == nil { //self issued transaction
		tx.serialize()
	}
	return tx.serialized
}

func (tx *Transaction) Hash() common.Hash {
	return tx.hash
}

func (tx *Transaction) Data() []byte {
	return tx.data
}

func (tx *Transaction) Signature() []byte {
	return tx.signature
}

func (tx *Transaction) Value() *big.Int {
	return tx.value
}

func (tx *Transaction) Nonce() uint64 {
	return tx.nonce
}

func (tx *Transaction) From() common.Address {
	return tx.from
}

func (tx *Transaction) To() common.Address {
	return tx.to
}

func (tx *Transaction) TxType() Type {
	return tx.txType
}

func (tx *Transaction) Fee() *big.Int {
	return tx.fee
}

type ByFeeAndValue []*Transaction

var log = logging.MustGetLogger("tx")

func (t ByFeeAndValue) Len() int {
	return len(t)
}

func (t ByFeeAndValue) Less(i, j int) bool {
	r := t[i].fee.Cmp(t[j].fee)
	if r == 0 {
		return t[i].value.Cmp(t[j].value) < 0
	}
	return r < 0
}

func (t ByFeeAndValue) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

func CreateTransaction(txType Type, to common.Address, from common.Address, nonce uint64, value *big.Int, fee *big.Int, data []byte, hash common.Hash) *Transaction {
	return &Transaction{txType: txType, to: to, from: from, nonce: nonce, value: value, fee: fee, data: data, hash: hash}
}

func (tx *Transaction) GetMessage() *pb.Transaction {
	var txType pb.Transaction_Type
	if tx.txType == Payment {
		txType = pb.Transaction_PAYMENT
	}
	return &pb.Transaction{
		Type:      txType,
		To:        tx.to.Bytes(),
		Nonce:     tx.nonce,
		Value:     tx.value.Int64(),
		Fee:       tx.fee.Int64(),
		Signature: tx.signature,
		Data:      tx.data,
	}
}

func (tx *Transaction) serialize() []byte {
	pbtx := tx.GetMessage()
	bytes, e := proto.Marshal(pbtx)
	if e != nil {
		log.Error("can't marshal tx", e)
		return nil
	}
	return bytes
}

func CreateTransactionFromMessage(msg *pb.Transaction) (*Transaction, error) {

	pub, e := crypto.SigToPub(Hash(msg).Bytes(), msg.GetSignature())
	if e != nil {
		return nil, errors.New("bad signature")
	}
	a := crypto.PubkeyToAddress(*pub)
	tx := CreateTransaction(Payment,
		common.BytesToAddress(msg.GetTo()),
		a,
		uint64(msg.GetNonce()),
		big.NewInt(msg.GetValue()),
		big.NewInt(msg.GetFee()),
		msg.GetData(),
		common.Hash{},
	)

	return tx, nil
}

func Hash(msg *pb.Transaction) common.Hash {
	msg.Signature = nil
	bytes, e := proto.Marshal(msg)
	if e != nil {
		log.Error("Can't calculate hash")
	}

	return crypto.Keccak256Hash(bytes)
}
