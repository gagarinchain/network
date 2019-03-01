package blockchain

import (
	"github.com/golang/protobuf/proto"
	"github.com/poslibp2p/eth/common"
	"github.com/poslibp2p/eth/crypto"
	"github.com/poslibp2p/message/protobuff"
	"time"
)

type Block struct {
	header *Header
	data   []byte
}

type Header struct {
	height    int32
	hash      common.Hash
	parent    *Header
	timestamp time.Time
	qc        *QuorumCertificate
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

func (h *Header) Parent() *Header {
	return h.parent
}
func (h *Header) Timestamp() time.Time {
	return h.timestamp
}
func (h *Header) QC() *QuorumCertificate {
	return h.qc
}

func (h *Header) QRef() *Header {
	return h.QC().QrefBlock()
}

func (h *Header) IsSibling(sibl *Header) bool {
	return sibl.Parent().IsSibling(h)
}

func CreateGenesisTriChain() (zero *Block, one *Block, two *Block, certToHead *QuorumCertificate) {
	//TODO find out what to do with alfa cert
	zeroHeader := createHeader(0, common.BytesToHash(make([]byte, common.HashLength)), nil, time.Now(), nil)
	zeroHeader.hash = crypto.Keccak256Hash([]byte("Block zero"))
	//We need block to calculate it's hash
	z := &Block{header: zeroHeader, data: []byte("Zero")}
	zeroCert := CreateQuorumCertificate([]byte("Valid"), z.header)

	firstHeader := createHeader(1, common.BytesToHash(make([]byte, common.HashLength)), zeroHeader, time.Now(), zeroCert)
	firstHeader.hash = crypto.Keccak256Hash([]byte("Block one"))
	first := &Block{header: firstHeader, data: []byte("First")}
	firstCert := CreateQuorumCertificate([]byte("Valid"), firstHeader)

	secondHeader := createHeader(2, common.BytesToHash(make([]byte, common.HashLength)), firstHeader, time.Now(), firstCert)
	secondHeader.hash = crypto.Keccak256Hash([]byte("Block two"))
	second := &Block{header: secondHeader, data: []byte("Second")}
	secondCert := CreateQuorumCertificate([]byte("Valid"), secondHeader)

	return z, first, second, secondCert
}

func CreateBlockAndSetHash(header *Header, data []byte) *Block {
	b := &Block{header: header, data: data}
	b.SetHash()
	return b
}

func (b *Block) SetHash() {
	msg := b.GetMessage()
	bytes, e := proto.Marshal(msg)
	if e == nil {
		b.header.hash = crypto.Keccak256Hash(bytes)
	}
}

func NewBlock(parent *Block, qc *QuorumCertificate, data []byte) *Block {
	hash := common.BytesToHash([]byte(""))
	header := &Header{height: parent.Header().Height() + 1, hash: hash, parent: parent.Header(), qc: qc}
	block := &Block{header: header, data: data}

	block.SetHash()
	return block
}

func createHeader(height int32, hash common.Hash, parent *Header, timestamp time.Time, qc *QuorumCertificate) *Header {
	return &Header{height: height, hash: hash, parent: parent, timestamp: timestamp, qc: qc}
}

func CreateHeader2(parent *Header, qc *QuorumCertificate) *Header {
	return createHeader(parent.height+1, common.BytesToHash(make([]byte, common.HashLength)), parent, time.Now(), qc)
}

func (h *Header) GetMessage() (msg *pb.BlockHeader) {
	return &pb.BlockHeader{ParentHash: h.Parent().Hash().Bytes(), DataHash: h.Hash().Bytes(), Height: h.Height(), Timestamp: h.Timestamp().Unix()}
}

func (h *Header) Serialize() []byte {
	msg := h.GetMessage()

	bytes, e := proto.Marshal(msg)
	if e != nil {
		log.Error("Can't marshall message", e)
	}

	return bytes
}

func (b *Block) GetMessage() (msg *pb.Block) {
	pdata := &pb.BlockData{Data: b.Data()}
	qc := b.Header().QC()
	var pqrefHeader = qc.QrefBlock().GetMessage()
	pcert := &pb.QuorumCertificate{Header: pqrefHeader, SignatureAggregate: qc.SignatureAggregate()}

	return &pb.Block{Header: b.Header().GetMessage(), Data: pdata, Cert: pcert}
}

func (b *Block) Serialize() []byte {
	msg := b.GetMessage()

	bytes, e := proto.Marshal(msg)
	if e != nil {
		log.Error("Can't marshall message", e)
	}

	return bytes
}
