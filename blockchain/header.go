package blockchain

import (
	"github.com/gagarinchain/common/api"
	"github.com/gagarinchain/common/eth/common"
	"github.com/gagarinchain/common/eth/crypto"
	pb "github.com/gagarinchain/common/protobuff"
	"github.com/golang/protobuf/proto"
	"time"
)

type HeadersByHeight []api.Header

func (h HeadersByHeight) Len() int           { return len(h) }
func (h HeadersByHeight) Less(i, j int) bool { return h[i].Height() < h[j].Height() }
func (h HeadersByHeight) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

type HeaderImpl struct {
	height    int32
	hash      common.Hash
	txHash    common.Hash
	stateHash common.Hash
	dataHash  common.Hash
	qcHash    common.Hash
	parent    common.Hash
	timestamp time.Time
}

func (h *HeaderImpl) DataHash() common.Hash {
	return h.dataHash
}

func (h *HeaderImpl) StateHash() common.Hash {
	return h.stateHash
}

func (h *HeaderImpl) Height() int32 {
	return h.height
}
func (h *HeaderImpl) Hash() common.Hash {
	return h.hash
}
func (h *HeaderImpl) TxHash() common.Hash {
	return h.txHash
}
func (h *HeaderImpl) QCHash() common.Hash {
	return h.qcHash
}

func (h *HeaderImpl) SetQCHash(hash common.Hash) {
	h.qcHash = hash
}

func (h *HeaderImpl) Parent() common.Hash {
	return h.parent
}

func (h *HeaderImpl) Timestamp() time.Time {
	return h.timestamp
}

func createHeader(height int32, hash common.Hash, qcHash common.Hash, txHash common.Hash,
	stateHash common.Hash, dataHash common.Hash, parent common.Hash, timestamp time.Time) api.Header {
	return &HeaderImpl{
		height:    height,
		hash:      hash,
		qcHash:    qcHash,
		stateHash: stateHash,
		txHash:    txHash,
		dataHash:  dataHash,
		parent:    parent,
		timestamp: timestamp,
	}
}

func (h *HeaderImpl) IsGenesisBlock() bool {
	return h.Height() == 0
}

func (h *HeaderImpl) GetMessage() *pb.BlockHeader {
	return &pb.BlockHeader{
		Hash:       h.Hash().Bytes(),
		ParentHash: h.Parent().Bytes(),
		TxHash:     h.TxHash().Bytes(),
		StateHash:  h.StateHash().Bytes(),
		DataHash:   h.DataHash().Bytes(),
		QcHash:     h.QCHash().Bytes(),
		Height:     h.Height(),
		Timestamp:  h.Timestamp().UnixNano(),
	}
}

func (h *HeaderImpl) ToStorageProto() *pb.BlockHeaderS {
	return &pb.BlockHeaderS{
		Hash:       h.Hash().Bytes(),
		ParentHash: h.Parent().Bytes(),
		TxHash:     h.TxHash().Bytes(),
		StateHash:  h.StateHash().Bytes(),
		DataHash:   h.DataHash().Bytes(),
		QcHash:     h.QCHash().Bytes(),
		Height:     h.Height(),
		Timestamp:  h.Timestamp().UnixNano(),
	}
}

func (h *HeaderImpl) SetHash() {
	h.hash = HashHeader(h)
}

func HashHeader(header api.Header) common.Hash {
	switch t := header.(type) {
	case *HeaderImpl:
		h := *t
		h.hash = common.BytesToHash(make([]byte, common.HashLength))
		if h.IsGenesisBlock() {
			h.qcHash = common.BytesToHash(make([]byte, common.HashLength))
		}
		m := h.GetMessage()
		bytes, e := proto.Marshal(m)
		if e != nil {
			log.Error("Can't marshal message")
		}

		return crypto.Keccak256Hash(bytes)
	default:
		panic("can't calculate header of unknown impl")
	}
}

//returns 96 byte of header signature
func (h *HeaderImpl) Sign(key *crypto.PrivateKey) *crypto.Signature {
	sig := crypto.Sign(h.hash.Bytes(), key)
	if sig == nil {
		log.Error("Can't sign message")
	}

	return sig
}

func CreateBlockHeaderFromMessage(header *pb.BlockHeader) api.Header {
	return createHeader(
		header.Height,
		common.BytesToHash(header.Hash),
		common.BytesToHash(header.QcHash),
		common.BytesToHash(header.TxHash),
		common.BytesToHash(header.StateHash),
		common.BytesToHash(header.DataHash),
		common.BytesToHash(header.ParentHash),
		time.Unix(0, header.Timestamp).UTC(),
	)
}
func CreateBlockHeaderFromStorage(header *pb.BlockHeaderS) api.Header {
	return createHeader(
		header.Height,
		common.BytesToHash(header.Hash),
		common.BytesToHash(header.QcHash),
		common.BytesToHash(header.TxHash),
		common.BytesToHash(header.StateHash),
		common.BytesToHash(header.DataHash),
		common.BytesToHash(header.ParentHash),
		time.Unix(0, header.Timestamp).UTC(),
	)
}
