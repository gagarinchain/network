package tx

import (
	"bytes"
	"errors"
	"github.com/gagarinchain/network/common/eth/common"
	"github.com/gagarinchain/network/common/eth/crypto"
	"github.com/gagarinchain/network/common/protobuff"
	"github.com/gogo/protobuf/proto"
	"github.com/op/go-logging"
	"math/big"
)

type Type int

const (
	SettlementAddressHex         = "0x6522b1ac0c0c078f1fcc696b9cf72c59bb3624b7d2a9d82059b2f3832fd9973d"
	DefaultSettlementReward      = 10 //probably this value should be set from config or via consensus or moved to different TX field
	DefaultAgreementFee          = 2
	Payment                 Type = iota
	Slashing                Type = iota
	Settlement              Type = iota
	Agreement               Type = iota
	Proof                   Type = iota
	Redeem                  Type = iota
)

type Iterator interface {
	Next() *Transaction
	HasNext() bool
}

type Transaction struct {
	txType    Type
	to        common.Address
	from      common.Address
	nonce     uint64
	value     *big.Int
	fee       *big.Int
	signature *crypto.Signature
	data      []byte
	hashKey   common.Hash

	serialized []byte
}

func (tx *Transaction) Serialized() []byte {
	if tx.serialized == nil { //self issued transaction
		tx.serialized = tx.serialize()
	}
	return tx.serialized
}

func (tx *Transaction) HashKey() common.Hash {
	if bytes.Equal(tx.hashKey.Bytes(), common.Hash{}.Bytes()) { //noy initialized
		tx.hashKey = tx.CalculateHashKey()
	}
	return tx.hashKey
}

func (tx *Transaction) Data() []byte {
	return tx.data
}

func (tx *Transaction) Signature() *crypto.Signature {
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

func (tx *Transaction) SetTo(to common.Address) {
	tx.to = to
}

func (tx *Transaction) TxType() Type {
	return tx.txType
}

func (tx *Transaction) Fee() *big.Int {
	return tx.fee
}

type ByFeeAndNonce []*Transaction

var log = logging.MustGetLogger("tx")

func (t ByFeeAndNonce) Len() int {
	return len(t)
}

func (t ByFeeAndNonce) Less(i, j int) bool {
	r := t[i].fee.Cmp(t[j].fee)
	if r == 0 {
		return t[i].nonce < t[j].nonce
	}
	return r < 0
}

func (t ByFeeAndNonce) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

func CreateTransaction(txType Type, to common.Address, from common.Address, nonce uint64, value *big.Int, fee *big.Int, data []byte) *Transaction {
	return &Transaction{
		txType:    txType,
		to:        to,
		from:      from,
		nonce:     nonce,
		value:     value,
		fee:       fee,
		data:      data,
		signature: crypto.EmptySignature(),
	}
}

func CreateAgreement(t *Transaction, nonce uint64, proof []byte) *Transaction {
	return &Transaction{
		to:        common.BytesToAddress(t.Hash().Bytes()[12:]),
		txType:    Agreement,
		value:     big.NewInt(0),
		fee:       big.NewInt(DefaultAgreementFee),
		nonce:     nonce,
		data:      proof,
		signature: crypto.EmptySignature(),
	}
}

func (tx *Transaction) CreateProof(pk *crypto.PrivateKey) (e error) {
	if tx.txType != Agreement {
		return errors.New("proof is allowed only for agreements")
	}
	sig := crypto.Sign(crypto.Keccak256(tx.to.Bytes()), pk)

	if sig == nil {
		return errors.New("can't create proof")
	}
	toProto := sig.ToProto()
	marshal, e := proto.Marshal(toProto)
	if e != nil {
		return errors.New("can't create proof")
	}
	tx.data = marshal

	return
}
func (tx *Transaction) RecoverProver() (aggregate *crypto.SignatureAggregate, e error) {
	if tx.txType != Proof {
		return aggregate, errors.New("proof is allowed only for proof tx")
	}

	aggr := &pb.SignatureAggregate{}
	if err := proto.Unmarshal(tx.data, aggr); err != nil {
		return nil, err
	}

	return crypto.AggregateFromProto(aggr), nil
}

func (tx *Transaction) GetMessage() *pb.Transaction {
	var txType pb.Transaction_Type

	switch tx.txType {
	case Payment:
		txType = pb.Transaction_PAYMENT
	case Slashing:
		txType = pb.Transaction_SLASHING
	case Settlement:
		txType = pb.Transaction_SETTLEMENT
	case Agreement:
		txType = pb.Transaction_AGREEMENT
	case Proof:
		txType = pb.Transaction_PROOF
	case Redeem:
		txType = pb.Transaction_REDEEM
	}

	return &pb.Transaction{
		Type:      txType,
		To:        tx.to.Bytes(),
		Nonce:     tx.nonce,
		Value:     tx.value.Int64(),
		Fee:       tx.fee.Int64(),
		Signature: tx.signature.ToProto(),
		Data:      tx.data,
	}
}

//internal transaction
func (tx *Transaction) serialize() []byte {
	pbtx := &pb.Tx{
		Type:      int32(tx.txType),
		From:      tx.from.Bytes(),
		To:        tx.to.Bytes(),
		Nonce:     tx.nonce,
		Value:     tx.value.Bytes(),
		Fee:       tx.fee.Bytes(),
		Signature: tx.signature.ToStorageProto(),
		Data:      tx.data,
		HashKey:   tx.hashKey.Bytes(),
	}

	bytes, e := proto.Marshal(pbtx)
	if e != nil {
		log.Error("can't marshal tx", e)
		return nil
	}
	return bytes
}

func Deserialize(tran []byte) (*Transaction, error) {
	pbt := &pb.Tx{}
	if err := proto.Unmarshal(tran, pbt); err != nil {
		return nil, err
	}

	return &Transaction{
		from:      common.BytesToAddress(pbt.From),
		txType:    Type(pbt.Type),
		hashKey:   common.BytesToHash(pbt.HashKey),
		value:     big.NewInt(0).SetBytes(pbt.Value),
		fee:       big.NewInt(0).SetBytes(pbt.Fee),
		signature: crypto.SignatureFromStorage(pbt.Signature),
		nonce:     pbt.Nonce,
		to:        common.BytesToAddress(pbt.To),
		data:      pbt.Data,
	}, nil
}

func CreateTransactionFromMessage(msg *pb.Transaction) (*Transaction, error) {
	hash := Hash(*msg)
	sign := crypto.SignatureFromProto(msg.GetSignature())
	res := crypto.Verify(hash.Bytes(), sign)
	if !res {
		return nil, errors.New("bad signature")
	}

	a := crypto.PubkeyToAddress(crypto.NewPublicKey(sign.Pub()))
	var txType Type
	switch msg.Type {
	case pb.Transaction_PAYMENT:
		txType = Payment
	case pb.Transaction_SLASHING:
		txType = Slashing
	case pb.Transaction_SETTLEMENT:
		txType = Settlement
	case pb.Transaction_AGREEMENT:
		txType = Agreement
	case pb.Transaction_PROOF:
		txType = Proof
	case pb.Transaction_REDEEM:
		txType = Redeem
	}

	tx := CreateTransaction(txType,
		common.BytesToAddress(msg.GetTo()),
		a,
		msg.GetNonce(),
		big.NewInt(msg.GetValue()),
		big.NewInt(msg.GetFee()),
		msg.GetData(),
	)
	tx.signature = sign
	return tx, nil
}

//For test purposes
func (tx *Transaction) Sign(key *crypto.PrivateKey) {
	pbtx := tx.GetMessage()
	hash := Hash(*pbtx)
	sig := crypto.Sign(hash.Bytes(), key)

	if sig == nil {
		log.Error("Can't sign message")
	}

	tx.signature = sig
}

func (tx *Transaction) Hash() common.Hash {
	pbtx := tx.GetMessage()
	return Hash(*pbtx)
}

func Hash(msg pb.Transaction) common.Hash {
	msg.Signature = nil
	bytes, e := proto.Marshal(&msg)
	if e != nil {
		log.Error("Can't calculate hash")
	}

	return crypto.Keccak256Hash(bytes)
}

//we actually can't simply use hash of pb.Transaction as a key, because we can have the same message with signature excluded (we don't have TO field)
//that mean that we have to use hash of message with signature for key and hash of message without signature for signing, what seems like mess
func (tx *Transaction) CalculateHashKey() common.Hash {
	tx.hashKey = common.Hash{}
	m := tx.serialize()

	return crypto.Keccak256Hash(m)
}
