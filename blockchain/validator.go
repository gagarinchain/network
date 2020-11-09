package blockchain

import (
	"bytes"
	"errors"
	cmn "github.com/gagarinchain/common"
	"github.com/gagarinchain/common/api"
	"github.com/gagarinchain/common/eth/common"
	"github.com/gagarinchain/common/eth/crypto"
	"github.com/gagarinchain/common/protobuff"
	"github.com/golang/protobuf/proto"
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
	t := entity.(api.Transaction)
	switch t.TxType() {
	case api.Payment:
		if !v.validateValue(t) {
			return false, ValueNotValid
		}
		if !v.validateFee(t) {
			return false, FeeNotValid
		}
	case api.Settlement:
		if !v.validateValue(t) {
			return false, ValueNotValid
		}

		if t.Fee().Cmp(big.NewInt(api.DefaultSettlementReward)) < 0 {
			return false, FeeNotValid
		}

		if !bytes.Equal(t.To().Bytes(), common.BytesToAddress(t.Hash().Bytes()[12:]).Bytes()) {
			return false, SettlementAddressNotValid
		}
	case api.Agreement:
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
	case api.Proof:
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

func (v *TransactionValidator) validateFee(t api.Transaction) bool {
	return t.Fee().Cmp(big.NewInt(0)) > 0
}

func (v *TransactionValidator) validateValue(t api.Transaction) bool {
	return t.Value().Cmp(big.NewInt(0)) > 0
}

func (v *TransactionValidator) Supported(mType pb.Message_MessageType) bool {
	return mType == pb.Message_TRANSACTION
}

func (v *TransactionValidator) GetId() interface{} {
	return "TransactionValidator"
}

type BlockValidator struct {
	committee       []*cmn.Peer
	headerValidator *HeaderValidator
	txVal           *TransactionValidator
}

func NewBlockValidator(committee []*cmn.Peer, txVal *TransactionValidator, headerValidator *HeaderValidator) *BlockValidator {
	return &BlockValidator{committee: committee, txVal: txVal, headerValidator: headerValidator}
}

//TODO add txhash validation
func (b *BlockValidator) IsValid(entity interface{}) (bool, error) {
	if entity == nil {
		return false, errors.New("entity is nil")
	}

	block := entity.(api.Block)

	isValid, e := b.headerValidator.IsValid(block.Header())

	if e != nil || !isValid {
		return isValid, e
	}

	//Skip checks for genesis block
	if block.Header().IsGenesisBlock() {
		return true, nil
	}

	dataHash := crypto.Keccak256(block.Data())
	if common.BytesToHash(dataHash) != block.Header().DataHash() {
		log.Debugf("calculated %v, received %v", dataHash, block.Header().TxHash())
		return false, errors.New("data hash is not valid")
	}

	valid, e := block.QC().IsValid(block.Header().QCHash(), cmn.PeersToPubs(b.committee))
	if !valid {
		return valid, e
	}

	if block.Signature() != nil {
		if !block.Signature().IsValid(block.Header().Hash().Bytes(), cmn.PeersToPubs(b.committee)) {
			return false, errors.New("block signature is not valid")
		}
	} else {
		iterator := block.Txs()
		for iterator.HasNext() {
			next := iterator.Next()
			isValid, e := b.txVal.IsValid(next)
			if !isValid {
				return isValid, e
			}
		}
	}

	return true, nil
}

func (b *BlockValidator) Supported(mType pb.Message_MessageType) bool {
	return mType == pb.Message_BLOCK_RESPONSE
}

func (b *BlockValidator) GetId() interface{} {
	return "Block"
}

type HeaderValidator struct {
}

func (b *HeaderValidator) IsValid(entity interface{}) (bool, error) {
	if entity == nil {
		return false, errors.New("entity is nil")
	}

	header := entity.(*HeaderImpl)

	if header.Height() < 0 {
		return false, errors.New("header height is negative")
	}
	if len(header.Hash().Bytes()) != 32 {
		return false, errors.New("header hash length is not valid")
	}
	if len(header.Parent().Bytes()) != 32 {
		return false, errors.New("header parent hash length is not valid")
	}
	if len(header.QCHash().Bytes()) != 32 {
		return false, errors.New("header QC hash length is not valid")
	}
	if len(header.DataHash().Bytes()) != 32 {
		return false, errors.New("header data hash length is not valid")
	}
	if len(header.StateHash().Bytes()) != 32 {
		return false, errors.New("header state hash length is not valid")
	}
	//if len(header.TxHash().Bytes()) != 32 {
	//	return false, errors.New("header state hash length is not valid")
	//}

	//Skip checks for genesis block
	if header.IsGenesisBlock() {
		return true, nil
	}

	hash := HashHeader(header)
	if header.Hash() != hash {
		log.Debugf("calculated %v, received %v", hash, header.Hash())
		return false, errors.New("block hash is not valid")
	}

	return true, nil
}

func (b *HeaderValidator) Supported(mType pb.Message_MessageType) bool {
	return mType == pb.Message_HEADERS_RESPONSE
}

func (b *HeaderValidator) GetId() interface{} {
	return "Header"
}
