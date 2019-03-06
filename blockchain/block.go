package blockchain

import (
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
	zeroHeader := createHeader(0, common.BytesToHash(make([]byte, common.HashLength)), common.BytesToHash(make([]byte, common.HashLength)), time.Now())
	zeroHeader.hash = crypto.Keccak256Hash([]byte("Block zero"))
	//We need block to calculate it's hash
	z := &Block{header: zeroHeader, data: []byte("Zero")}
	zeroCert := CreateQuorumCertificate([]byte("Valid"), z.header)

	firstHeader := createHeader(1, common.BytesToHash(make([]byte, common.HashLength)), zeroHeader.Hash(), time.Now())
	firstHeader.hash = crypto.Keccak256Hash([]byte("Block one"))
	first := &Block{header: firstHeader, data: []byte("First"), qc: zeroCert}
	firstCert := CreateQuorumCertificate([]byte("Valid"), firstHeader)

	secondHeader := createHeader(2, common.BytesToHash(make([]byte, common.HashLength)), firstHeader.Hash(), time.Now())
	secondHeader.hash = crypto.Keccak256Hash([]byte("Block two"))
	second := &Block{header: secondHeader, data: []byte("Second"), qc: firstCert}
	secondCert := CreateQuorumCertificate([]byte("Valid"), secondHeader)

	return z, first, second, secondCert
}

func createHeader(height int32, hash common.Hash, parent common.Hash, timestamp time.Time) *Header {
	return &Header{height: height, hash: hash, parent: parent, timestamp: timestamp}
}

func CreateHeader2(parent *Header) *Header {
	return createHeader(parent.height+1, common.BytesToHash(make([]byte, common.HashLength)), parent.Hash(), time.Now())
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
	return createHeader(header.Height, common.BytesToHash(header.DataHash), common.BytesToHash(header.ParentHash), time.Unix(header.Timestamp, 0))
}

func (h *Header) GetMessage() *pb.BlockHeader {
	return &pb.BlockHeader{ParentHash: h.Parent().Bytes(), DataHash: h.Hash().Bytes(), Height: h.Height(), Timestamp: h.Timestamp().Unix()}
}
