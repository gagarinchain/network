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

type Header struct {
	height    int32
	hash      common.Hash
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
func (h *Header) Parent() common.Hash {
	return h.parent
}

func (h *Header) Timestamp() time.Time {
	return h.timestamp
}
func (b *Block) QC() *QuorumCertificate {
	return b.qc
}

func (b *Block) QRef() *Header {
	return b.QC().QrefBlock()
}

func CreateGenesisTriChain() (zero *Block, one *Block, two *Block, certToHead *QuorumCertificate) {
	//TODO find out what to do with alfa cert
	zeroHeader := createHeader(0, common.BytesToHash(make([]byte, common.HashLength)), common.BytesToHash(make([]byte, common.HashLength)), time.Now().Round(time.Millisecond))
	zeroHeader.hash = crypto.Keccak256Hash([]byte("Block zero"))
	//We need block to calculate it's hash
	z := &Block{header: zeroHeader, data: []byte("Zero")}
	zeroCert := CreateQuorumCertificate([]byte("Valid"), z.header)

	firstHeader := createHeader(1, common.BytesToHash(make([]byte, common.HashLength)), zeroHeader.Hash(), time.Now().Round(time.Millisecond))
	firstHeader.hash = crypto.Keccak256Hash([]byte("Block one"))
	first := &Block{header: firstHeader, data: []byte("First"), qc: zeroCert}
	firstCert := CreateQuorumCertificate([]byte("Valid"), firstHeader)

	secondHeader := createHeader(2, common.BytesToHash(make([]byte, common.HashLength)), firstHeader.Hash(), time.Now().Round(time.Millisecond))
	secondHeader.hash = crypto.Keccak256Hash([]byte("Block two"))
	second := &Block{header: secondHeader, data: []byte("Second"), qc: firstCert}
	secondCert := CreateQuorumCertificate([]byte("Valid"), secondHeader)

	return z, first, second, secondCert
}

func createHeader(height int32, hash common.Hash, parent common.Hash, timestamp time.Time) *Header {
	return &Header{height: height, hash: hash, parent: parent, timestamp: timestamp}
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
	return &pb.Block{Header: b.Header().GetMessage(), Cert: b.QC().GetMessage(), Data: &pb.BlockData{Data: b.Data()}}
}

func CreateBlockHeaderFromMessage(header *pb.BlockHeader) *Header {
	return createHeader(header.Height, common.BytesToHash(header.DataHash), common.BytesToHash(header.ParentHash), time.Unix(0, header.Timestamp))
}

func (h *Header) GetMessage() *pb.BlockHeader {
	return &pb.BlockHeader{ParentHash: h.Parent().Bytes(), DataHash: h.Hash().Bytes(), Height: h.Height(), Timestamp: h.Timestamp().UnixNano()}
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

	return common.BytesToHash(crypto.Keccak256(bytes))
}

//returns 65 byte of header signature in [R || S || V] format
func (h *Header) Sign(key *ecdsa.PrivateKey) []byte {
	sig, err := crypto.Sign(h.hash.Bytes(), key)

	if err != nil {
		log.Error("Can't sign message", err)
	}

	return sig
}
