package tx

import (
	"bytes"
	"errors"
	"github.com/davecgh/go-spew/spew"
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
	txType     Type
	to         common.Address
	from       common.Address
	nonce      uint64
	value      *big.Int
	fee        *big.Int
	signature  *crypto.Signature
	data       []byte
	hash       common.Hash
	confirmed  bool
	serialized []byte
}

func (tx *Transaction) Serialized() []byte {
	if tx.serialized == nil { //self issued transaction
		m := tx.ToStorageProto()
		b, e := proto.Marshal(m)
		if e != nil {
			log.Error("Can't marshal message")
		}

		return b
	}
	return tx.serialized
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
		txType:    Agreement,
		to:        common.BytesToAddress(t.Hash().Bytes()[12:]),
		from:      t.From(),
		nonce:     nonce,
		value:     big.NewInt(0),
		fee:       big.NewInt(DefaultAgreementFee),
		data:      proof,
		signature: crypto.EmptySignature(),
	}
}

func (tx *Transaction) SetFrom(from common.Address) {
	tx.from = from
}

func (tx *Transaction) CreateProof(pk *crypto.PrivateKey) (e error) {
	if tx.txType != Agreement {
		return errors.New("proof is allowed only for agreements")
	}
	sig := crypto.Sign(crypto.Keccak256(tx.to.Bytes()), pk)
	spew.Dump("proof to sign", crypto.Keccak256(tx.to.Bytes()))
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

	var from []byte
	var sign *pb.Signature
	if tx.confirmed {
		from = tx.from.Bytes()
	} else {
		sign = tx.signature.ToProto()
	}
	return &pb.Transaction{
		Type:      txType,
		To:        tx.to.Bytes(),
		From:      from,
		Nonce:     tx.nonce,
		Value:     tx.value.Int64(),
		Fee:       tx.fee.Int64(),
		Signature: sign,
		Data:      tx.data,
	}
}

//internal transaction
func (tx *Transaction) ToStorageProto() *pb.TransactionS {
	spew.Dump(tx)
	var sign *pb.SignatureS
	if !tx.confirmed {
		sign = tx.signature.ToStorageProto()
	}
	return &pb.TransactionS{
		Type:      int32(tx.txType),
		From:      tx.from.Bytes(),
		To:        tx.to.Bytes(),
		Nonce:     tx.nonce,
		Value:     tx.value.Int64(),
		Fee:       tx.fee.Int64(),
		Signature: sign,
		Data:      tx.data,
		Hash:      tx.hash.Bytes(),
	}
}

func Deserialize(tran []byte) (*Transaction, error) {
	pbt := &pb.TransactionS{}
	if err := proto.Unmarshal(tran, pbt); err != nil {
		return nil, err
	}

	var signature *crypto.Signature
	confirmed := true
	if pbt.Signature != nil {
		signature = crypto.SignatureFromStorageProto(pbt.Signature)
		confirmed = false
	}
	return &Transaction{
		from:      common.BytesToAddress(pbt.From),
		txType:    Type(pbt.Type),
		hash:      common.BytesToHash(pbt.Hash),
		value:     big.NewInt(pbt.Value),
		fee:       big.NewInt(pbt.Fee),
		signature: signature,
		nonce:     pbt.Nonce,
		to:        common.BytesToAddress(pbt.To),
		data:      pbt.Data,
		confirmed: confirmed,
	}, nil
}

func CreateTransactionFromMessage(msg *pb.Transaction, isConfirmed bool) (*Transaction, error) {
	fromBytes := msg.GetFrom()
	var from common.Address
	var sign *crypto.Signature

	//TODO remove this validations to validator
	if isConfirmed && fromBytes == nil {
		return nil, errors.New("no from for confirmed transaction")
	}
	if !isConfirmed && msg.GetSignature() == nil {
		return nil, errors.New("no signature for unconfirmed transaction")
	}

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
		from,
		msg.GetNonce(),
		big.NewInt(msg.GetValue()),
		big.NewInt(msg.GetFee()),
		msg.GetData(),
	)

	if len(fromBytes) == 20 && !bytes.Equal(common.Address{}.Bytes(), fromBytes) {
		from = common.BytesToAddress(msg.GetFrom())
	} else {
		sign = crypto.SignatureFromProto(msg.GetSignature())
		from = crypto.PubkeyToAddress(crypto.NewPublicKey(sign.Pub()))
		tx.from = from
		hash := Hash(tx)
		log.Debug("validate", from.Hex())
		log.Debug("validate", hash.Hex())
		res := crypto.Verify(hash.Bytes(), sign)
		if !res {
			return nil, errors.New("bad signature")
		}
	}

	tx.signature = sign
	return tx, nil
}
func CreateTransactionFromStorage(msg *pb.TransactionS) (*Transaction, error) {
	var sign *crypto.Signature
	from := common.BytesToAddress(msg.GetFrom())
	if msg.GetSignature() != nil {
		sign = crypto.SignatureFromStorageProto(msg.GetSignature())
	}

	txType := Type(msg.Type)

	tx := CreateTransaction(txType,
		common.BytesToAddress(msg.GetTo()),
		from,
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
	hash := tx.Hash()
	sig := crypto.Sign(hash.Bytes(), key)

	log.Debug("signing ", tx.from.Hex())
	log.Debug("signing ", hash.Hex())

	if sig == nil {
		log.Error("Can't sign message")
	}

	tx.signature = sig
}

func (tx *Transaction) Hash() common.Hash {
	if bytes.Equal(tx.hash.Bytes(), common.Hash{}.Bytes()) { //not initialized
		return Hash(tx)
	}
	return tx.hash
}

func Hash(tx *Transaction) common.Hash {
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

	msg := &pb.Transaction{
		Type:  txType,
		To:    tx.to.Bytes(),
		From:  tx.from.Bytes(),
		Nonce: tx.nonce,
		Value: tx.value.Int64(),
		Fee:   tx.fee.Int64(),
		Data:  tx.data,
	}
	b, e := proto.Marshal(msg)
	if e != nil {
		log.Error("Can't calculate hash")
	}

	return crypto.Keccak256Hash(b)
}

func (tx *Transaction) DropSignature() {
	tx.signature = nil
	tx.confirmed = true

}
