package tx

import (
	"bytes"
	"errors"
	"github.com/davecgh/go-spew/spew"
	"github.com/gagarinchain/common/api"
	"github.com/gagarinchain/common/eth/common"
	"github.com/gagarinchain/common/eth/crypto"
	"github.com/gagarinchain/common/protobuff"
	"github.com/gogo/protobuf/proto"
	"github.com/op/go-logging"
	"math/big"
)

type TransactionImpl struct {
	txType     api.Type
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

func (tx *TransactionImpl) Serialized() []byte {
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

func (tx *TransactionImpl) Data() []byte {
	return tx.data
}

func (tx *TransactionImpl) Signature() *crypto.Signature {
	return tx.signature
}

func (tx *TransactionImpl) Value() *big.Int {
	return tx.value
}

func (tx *TransactionImpl) Nonce() uint64 {
	return tx.nonce
}

func (tx *TransactionImpl) From() common.Address {
	return tx.from
}

func (tx *TransactionImpl) To() common.Address {
	return tx.to
}

func (tx *TransactionImpl) SetTo(to common.Address) {
	tx.to = to
}

func (tx *TransactionImpl) TxType() api.Type {
	return tx.txType
}

func (tx *TransactionImpl) Fee() *big.Int {
	return tx.fee
}

type ByFeeAndNonce []api.Transaction

var log = logging.MustGetLogger("tx")

func (t ByFeeAndNonce) Len() int {
	return len(t)
}

func (t ByFeeAndNonce) Less(i, j int) bool {
	r := t[i].Fee().Cmp(t[j].Fee())
	if r == 0 {
		return t[i].Nonce() < t[j].Nonce()
	}
	return r < 0
}

func (t ByFeeAndNonce) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

func CreateTransaction(txType api.Type, to common.Address, from common.Address, nonce uint64, value *big.Int, fee *big.Int, data []byte) *TransactionImpl {
	return &TransactionImpl{
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

func CreateAgreement(t api.Transaction, nonce uint64, proof []byte) *TransactionImpl {
	return &TransactionImpl{
		txType:    api.Agreement,
		to:        common.BytesToAddress(t.Hash().Bytes()[12:]),
		from:      t.From(),
		nonce:     nonce,
		value:     big.NewInt(0),
		fee:       big.NewInt(api.DefaultAgreementFee),
		data:      proof,
		signature: crypto.EmptySignature(),
	}
}

func (tx *TransactionImpl) SetFrom(from common.Address) {
	tx.from = from
}

func (tx *TransactionImpl) CreateProof(pk *crypto.PrivateKey) (e error) {
	if tx.txType != api.Agreement {
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
func (tx *TransactionImpl) RecoverProver() (aggregate *crypto.SignatureAggregate, e error) {
	if tx.txType != api.Proof {
		return aggregate, errors.New("proof is allowed only for proof tx")
	}

	aggr := &pb.SignatureAggregate{}
	if err := proto.Unmarshal(tx.data, aggr); err != nil {
		return nil, err
	}

	return crypto.AggregateFromProto(aggr), nil
}

func (tx *TransactionImpl) GetMessage() *pb.Transaction {
	var txType pb.Transaction_Type

	switch tx.txType {
	case api.Payment:
		txType = pb.Transaction_PAYMENT
	case api.Slashing:
		txType = pb.Transaction_SLASHING
	case api.Settlement:
		txType = pb.Transaction_SETTLEMENT
	case api.Agreement:
		txType = pb.Transaction_AGREEMENT
	case api.Proof:
		txType = pb.Transaction_PROOF
	case api.Redeem:
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
func (tx *TransactionImpl) ToStorageProto() *pb.TransactionS {
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

func Deserialize(tran []byte) (*TransactionImpl, error) {
	pbt := &pb.TransactionS{}
	if err := proto.Unmarshal(tran, pbt); err != nil {
		return nil, err
	}

	return CreateTransactionFromStorage(pbt)
}

func CreateTransactionFromStorage(msg *pb.TransactionS) (*TransactionImpl, error) {
	var sign *crypto.Signature
	confirmed := true
	from := common.BytesToAddress(msg.GetFrom())
	if msg.GetSignature() != nil {
		sign = crypto.SignatureFromStorageProto(msg.GetSignature())
		confirmed = false
	}

	txType := api.Type(msg.Type)

	tx := CreateTransaction(txType,
		common.BytesToAddress(msg.GetTo()),
		from,
		msg.GetNonce(),
		big.NewInt(msg.GetValue()),
		big.NewInt(msg.GetFee()),
		msg.GetData(),
	)

	tx.signature = sign
	tx.confirmed = confirmed
	return tx, nil
}

func CreateTransactionFromMessage(msg *pb.Transaction, isConfirmed bool) (*TransactionImpl, error) {
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

	var txType api.Type
	switch msg.Type {
	case pb.Transaction_PAYMENT:
		txType = api.Payment
	case pb.Transaction_SLASHING:
		txType = api.Slashing
	case pb.Transaction_SETTLEMENT:
		txType = api.Settlement
	case pb.Transaction_AGREEMENT:
		txType = api.Agreement
	case pb.Transaction_PROOF:
		txType = api.Proof
	case pb.Transaction_REDEEM:
		txType = api.Redeem
	}

	tx := CreateTransaction(txType,
		common.BytesToAddress(msg.GetTo()),
		from,
		msg.GetNonce(),
		big.NewInt(msg.GetValue()),
		big.NewInt(msg.GetFee()),
		msg.GetData(),
	)
	tx.confirmed = isConfirmed

	if len(fromBytes) == 20 && !bytes.Equal(common.Address{}.Bytes(), fromBytes) {
		tx.from = common.BytesToAddress(msg.GetFrom())
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

func (tx *TransactionImpl) Sign(key *crypto.PrivateKey) {
	hash := tx.Hash()
	sig := crypto.Sign(hash.Bytes(), key)

	log.Debug("signing ", tx.from.Hex())
	log.Debug("signing ", hash.Hex())

	if sig == nil {
		log.Error("Can't sign message")
	}

	tx.signature = sig
}

func (tx *TransactionImpl) Hash() common.Hash {
	if bytes.Equal(tx.hash.Bytes(), common.Hash{}.Bytes()) { //not initialized
		return Hash(tx)
	}
	return tx.hash
}

func Hash(tx api.Transaction) common.Hash {
	switch typ := tx.(type) {
	case *TransactionImpl:
		t := *typ
		var txType pb.Transaction_Type

		switch t.txType {
		case api.Payment:
			txType = pb.Transaction_PAYMENT
		case api.Slashing:
			txType = pb.Transaction_SLASHING
		case api.Settlement:
			txType = pb.Transaction_SETTLEMENT
		case api.Agreement:
			txType = pb.Transaction_AGREEMENT
		case api.Proof:
			txType = pb.Transaction_PROOF
		case api.Redeem:
			txType = pb.Transaction_REDEEM
		}

		msg := &pb.Transaction{
			Type:  txType,
			To:    t.to.Bytes(),
			From:  t.from.Bytes(),
			Nonce: t.nonce,
			Value: t.value.Int64(),
			Fee:   t.fee.Int64(),
			Data:  t.data,
		}
		b, e := proto.Marshal(msg)
		if e != nil {
			log.Error("Can't calculate hash")
		}

		return crypto.Keccak256Hash(b)
	default:
		panic("unknown tx implementation")
	}
}

func (tx *TransactionImpl) DropSignature() {
	tx.signature = nil
	tx.confirmed = true

}
