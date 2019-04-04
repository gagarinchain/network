package blockchain

import (
	"crypto/ecdsa"
	"github.com/gogo/protobuf/proto"
	"github.com/poslibp2p/eth/common"
	"github.com/poslibp2p/eth/crypto"
	"github.com/poslibp2p/message/protobuff"
	"time"
)

type Block struct {
	header *Header
	qc     *QuorumCertificate
	data   []byte
}

//TODO fix hash - datahash mess
type Header struct {
	height    int32
	hash      common.Hash
	dataHash  common.Hash
	qcHash    common.Hash
	parent    common.Hash
	timestamp time.Time
}

func (b *Block) Header() *Header {
	return b.header
}
func (b *Block) Data() []byte {
	return b.data
}

func (h *Header) Height() int32 {
	return h.height
}
func (h *Header) Hash() common.Hash {
	return h.hash
}
func (h *Header) DataHash() common.Hash {
	return h.dataHash
}
func (h *Header) QCHash() common.Hash {
	return h.qcHash
}
func (h *Header) Parent() common.Hash {
	return h.parent
}

func (h *Header) Timestamp() time.Time {
	return h.timestamp
}
func (b *Block) QC() *QuorumCertificate {
	return b.qc
}
func (b *Block) SetQC(qc *QuorumCertificate) {
	b.qc = qc
}
func (b *Block) QRef() *Header {
	return b.QC().QrefBlock()
}

func CreateGenesisBlock() (zero *Block) {
	//TODO find out what to do with alfa cert
	data := []byte("Zero")
	zeroHeader := createHeader(0, common.BytesToHash(make([]byte, common.HashLength)), common.BytesToHash(make([]byte, common.HashLength)),
		crypto.Keccak256Hash(data), common.BytesToHash(make([]byte, common.HashLength)), time.Now().Round(time.Millisecond))
	zeroHeader.SetHash()
	//We need block to calculate it's hash
	zero = &Block{header: zeroHeader, data: data}

	return zero
}

func createHeader(height int32, hash common.Hash, qcHash common.Hash, dataHash common.Hash, parent common.Hash, timestamp time.Time) *Header {
	return &Header{
		height:    height,
		hash:      hash,
		qcHash:    qcHash,
		dataHash:  dataHash,
		parent:    parent,
		timestamp: timestamp,
	}
}

func (h *Header) IsGenesisBlock() bool {
	return h.Height() == 0
}

func CreateBlockFromMessage(block *pb.Block) *Block {
	header := CreateBlockHeaderFromMessage(block.Header)

	certificate := CreateQuorumCertificate(block.Cert.GetSignatureAggregate(), CreateBlockHeaderFromMessage(block.Cert.Header))
	return &Block{header: header, qc: certificate, data: block.Data.Data}
}

func (b *Block) GetMessage() *pb.Block {
	var qc *pb.QuorumCertificate
	if b.qc != nil {
		qc = b.QC().GetMessage()
	}
	return &pb.Block{Header: b.Header().GetMessage(), Cert: qc, Data: &pb.BlockData{Data: b.Data()}}
}

func CreateBlockHeaderFromMessage(header *pb.BlockHeader) *Header {
	return createHeader(header.Height, common.BytesToHash(header.Hash), common.BytesToHash(header.QcHash), common.BytesToHash(header.DataHash), common.BytesToHash(header.ParentHash), time.Unix(0, header.Timestamp))
}

func (h *Header) GetMessage() *pb.BlockHeader {
	return &pb.BlockHeader{
		Hash:       h.Hash().Bytes(),
		ParentHash: h.Parent().Bytes(),
		DataHash:   h.DataHash().Bytes(),
		QcHash:     h.QCHash().Bytes(),
		Height:     h.Height(),
		Timestamp:  h.Timestamp().UnixNano(),
	}
}

func (h *Header) SetHash() {
	h.hash = HashHeader(*h)
}

//We intentionally use this method on structure not on pointer to it, we need of blockHeader here
func HashHeader(h Header) common.Hash {
	h.hash = common.Hash{}
	m := h.GetMessage()
	bytes, e := proto.Marshal(m)
	if e != nil {
		log.Error("Can't marshal message")
	}

	return crypto.Keccak256Hash(bytes)
}

//returns 65 byte of header signature in [R || S || V] format
func (h *Header) Sign(key *ecdsa.PrivateKey) []byte {
	sig, err := crypto.Sign(h.hash.Bytes(), key)

	if err != nil {
		log.Error("Can't sign message", err)
	}

	return sig
}
