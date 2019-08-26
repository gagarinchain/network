package tx

import (
	"bytes"
	"crypto/ecdsa"
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
	DefaultSettlementReward      = 10 //probably this value should be set from config or via consensus
	DefaultAgreementFee          = 2
	Payment                 Type = iota
	Slashing                Type = iota
	Settlement              Type = iota
	Agreement               Type = iota
	Proof                   Type = iota
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
	signature []byte
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

func (tx *Transaction) SetTo(to common.Address) {
	tx.to = to
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

func CreateTransaction(txType Type, to common.Address, from common.Address, nonce uint64, value *big.Int, fee *big.Int, data []byte) *Transaction {
	return &Transaction{
		txType: txType,
		to:     to,
		from:   from,
		nonce:  nonce,
		value:  value,
		fee:    fee,
		data:   data,
	}
}

func CreateAgreement(t *Transaction, nonce uint64, proof []byte) *Transaction {
	return &Transaction{
		to:     common.BytesToAddress(t.Hash().Bytes()[12:]),
		txType: Agreement,
		value:  big.NewInt(0),
		fee:    big.NewInt(DefaultAgreementFee),
		nonce:  nonce,
		data:   proof,
	}
}

func (tx *Transaction) CreateProof(pk *ecdsa.PrivateKey) (e error) {
	if tx.txType != Agreement {
		return errors.New("proof is allowed only for agreements")
	}
	sig, e := crypto.Sign(crypto.Keccak256(tx.to.Bytes()), pk)

	if e != nil {
		return e
	}
	tx.data = sig

	return
}
func (tx *Transaction) RecoverProver() (provers []common.Address, e error) {
	if tx.txType != Proof {
		return provers, errors.New("proof is allowed only for proof tx")
	}

	for i := 0; i < len(tx.Data()); i += 65 {
		sig := tx.Data()[i : i+65]
		pub, e := crypto.SigToPub(crypto.Keccak256(tx.to.Bytes()), sig)
		if e != nil {
			return provers, errors.New("bad signature")
		}
		a := crypto.PubkeyToAddress(*pub)
		provers = append(provers, a)
	}

	return
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

//internal transaction
func (tx *Transaction) serialize() []byte {
	pbtx := &pb.Tx{
		Type:      int32(tx.txType),
		From:      tx.from.Bytes(),
		To:        tx.to.Bytes(),
		Nonce:     tx.nonce,
		Value:     tx.value.Bytes(),
		Fee:       tx.fee.Bytes(),
		Signature: tx.signature,
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
		signature: pbt.Signature,
		nonce:     pbt.Nonce,
		to:        common.BytesToAddress(pbt.To),
		data:      pbt.Data,
	}, nil
}

func CreateTransactionFromMessage(msg *pb.Transaction) (*Transaction, error) {
	hash := Hash(*msg)
	pub, e := crypto.SigToPub(hash.Bytes(), msg.GetSignature())
	if e != nil {
		return nil, errors.New("bad signature")
	}
	a := crypto.PubkeyToAddress(*pub)

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
	}

	tx := CreateTransaction(txType,
		common.BytesToAddress(msg.GetTo()),
		a,
		uint64(msg.GetNonce()),
		big.NewInt(msg.GetValue()),
		big.NewInt(msg.GetFee()),
		msg.GetData(),
	)

	return tx, nil
}

//For test purposes
func (tx *Transaction) Sign(key *ecdsa.PrivateKey) {
	pbtx := tx.GetMessage()
	hash := Hash(*pbtx)
	sig, err := crypto.Sign(hash.Bytes(), key)

	if err != nil {
		log.Error("Can't sign message", err)
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
