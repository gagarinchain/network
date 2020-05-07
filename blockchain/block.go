package blockchain

import (
	"github.com/gagarinchain/network/common/eth/common"
	"github.com/gagarinchain/network/common/eth/crypto"
	"github.com/gagarinchain/network/common/protobuff"
	"github.com/gagarinchain/network/common/trie"
	"github.com/gagarinchain/network/common/tx"
	"github.com/gogo/protobuf/proto"
	"time"
)

type Block struct {
	header    *Header
	qc        *QuorumCertificate
	signature *crypto.SignatureAggregate
	txs       *trie.FixedLengthHexKeyMerkleTrie
	data      []byte
}

func (b *Block) TxsCount() int {
	return len(b.txs.Values())
}

func (b *Block) Txs() tx.Iterator {
	var transactions []*tx.Transaction
	for _, bytes := range b.txs.Values() {
		//todo be careful we unmarshal and recover key here, think about storing deserialized entities in the trie
		t, e := tx.Deserialize(bytes)
		if e != nil {
			log.Error(e)
			return nil
		}
		transactions = append(transactions, t)
	}

	return newIterator(transactions)
}

func (b *Block) AddTransaction(t *tx.Transaction) {
	key := []byte(t.Hash().Hex())
	b.txs.InsertOrUpdate(key, t.Serialized())
}

type Header struct {
	height    int32
	hash      common.Hash
	txHash    common.Hash
	stateHash common.Hash
	dataHash  common.Hash
	qcHash    common.Hash
	parent    common.Hash
	timestamp time.Time
}

func (h *Header) DataHash() common.Hash {
	return h.dataHash
}

func (h *Header) StateHash() common.Hash {
	return h.stateHash
}

type ByHeight []*Block

func (h ByHeight) Len() int           { return len(h) }
func (h ByHeight) Less(i, j int) bool { return h[i].Height() < h[j].Height() }
func (h ByHeight) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

type HeadersByHeight []*Header

func (h HeadersByHeight) Len() int           { return len(h) }
func (h HeadersByHeight) Less(i, j int) bool { return h[i].Height() < h[j].Height() }
func (h HeadersByHeight) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (b *Block) Header() *Header {
	return b.header
}
func (b *Block) Data() []byte {
	return b.data
}

func (h *Header) Height() int32 {
	return h.height
}
func (b *Block) Height() int32 {
	return b.Header().Height()
}
func (h *Header) Hash() common.Hash {
	return h.hash
}
func (h *Header) TxHash() common.Hash {
	return h.txHash
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

//recalculate hash, since we add new field to block
func (b *Block) SetQC(qc *QuorumCertificate) {
	b.qc = qc
	b.header.qcHash = qc.GetHash()
	b.header.SetHash()
}

func (b *Block) Signature() *crypto.SignatureAggregate {
	return b.signature
}
func (b *Block) SetSignature(s *crypto.SignatureAggregate) {
	b.signature = s
	b.pruneTxSignatures()
}

func (b *Block) QRef() *Header {
	return b.QC().QrefBlock()
}

func CreateGenesisBlock() (zero *Block) {
	data := []byte("Zero")
	zeroHeader := createHeader(0, common.BytesToHash(make([]byte, common.HashLength)), common.BytesToHash(make([]byte, common.HashLength)),
		common.BytesToHash(make([]byte, common.HashLength)), crypto.Keccak256Hash(),
		crypto.Keccak256Hash(data), common.BytesToHash(make([]byte, common.HashLength)),
		time.Date(2019, time.April, 12, 0, 0, 0, 0, time.UTC).Round(time.Millisecond))
	zeroHeader.SetHash()
	zero = &Block{header: zeroHeader, data: data, qc: CreateQuorumCertificate(crypto.EmptyAggregateSignatures(), zeroHeader),
		txs: trie.New(), signature: crypto.EmptyAggregateSignatures()}

	return zero
}

func createHeader(height int32, hash common.Hash, qcHash common.Hash, txHash common.Hash,
	stateHash common.Hash, dataHash common.Hash, parent common.Hash, timestamp time.Time) *Header {
	return &Header{
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

func (h *Header) IsGenesisBlock() bool {
	return h.Height() == 0
}

func CreateBlockFromMessage(block *pb.Block) *Block {
	header := CreateBlockHeaderFromMessage(block.Header)
	aggrPb := block.Cert.GetSignatureAggregate()
	cert := CreateQuorumCertificate(crypto.AggregateFromProto(aggrPb), CreateBlockHeaderFromMessage(block.Cert.Header))
	var txs = trie.New()
	for _, tpb := range block.Txs {
		t, e := tx.CreateTransactionFromMessage(tpb, block.GetSignatureAggregate() != nil)
		if e != nil {
			log.Errorf("Bad transaction, %v", e)
			return nil
		}
		txs.InsertOrUpdate([]byte(t.Hash().Hex()), t.Serialized())
	}

	var signature *crypto.SignatureAggregate
	if block.SignatureAggregate != nil {
		signature = crypto.AggregateFromProto(block.SignatureAggregate)
	}
	return &Block{header: header, signature: signature, qc: cert, data: block.Data.Data, txs: txs}
}

func CreateBlockFromStorage(block *pb.BlockS) *Block {
	header := CreateBlockHeaderFromStorage(block.Header)
	aggrPb := block.Cert.GetSignatureAggregate()
	cert := CreateQuorumCertificate(crypto.AggregateFromStorage(aggrPb), CreateBlockHeaderFromStorage(block.Cert.Header))
	var txs = trie.New()
	for _, tpb := range block.Txs {
		t, e := tx.CreateTransactionFromStorage(tpb)
		if e != nil {
			log.Errorf("Bad transaction, %v", e)
			return nil
		}
		txs.InsertOrUpdate([]byte(t.Hash().Hex()), t.Serialized())
	}

	var signature *crypto.SignatureAggregate
	if block.SignatureAggregate != nil {
		signature = crypto.AggregateFromStorage(block.SignatureAggregate)
	}
	return &Block{header: header, signature: signature, qc: cert, data: block.Data.Data, txs: txs}
}

func (b *Block) GetMessage() *pb.Block {
	var qc *pb.QuorumCertificate
	if b.qc != nil {
		qc = b.QC().GetMessage()
	}

	var txs []*pb.Transaction
	if b.TxsCount() > 0 {
		it := b.Txs()
		for t := it.Next(); t != nil; t = it.Next() {
			txs = append(txs, t.GetMessage())
		}
	}

	var sign *pb.SignatureAggregate
	if b.signature != nil {
		sign = b.signature.ToProto()
	}

	return &pb.Block{Header: b.Header().GetMessage(), Cert: qc, SignatureAggregate: sign,
		Data: &pb.BlockData{Data: b.Data()}, Txs: txs}
}

func (b *Block) ToStorageProto() *pb.BlockS {
	var qc *pb.QuorumCertificateS
	if b.qc != nil {
		qc = b.QC().ToStorageProto()
	}

	var txs []*pb.TransactionS
	if b.TxsCount() > 0 {
		it := b.Txs()
		for t := it.Next(); t != nil; t = it.Next() {
			txs = append(txs, t.ToStorageProto())
		}
	}

	var sign *pb.SignatureAggregateS
	if b.signature != nil {
		sign = b.signature.ToStorageProto()
	}

	return &pb.BlockS{Header: b.Header().ToStorageProto(), Cert: qc, SignatureAggregate: sign,
		Data: &pb.BlockDataS{Data: b.Data()}, Txs: txs}
}

func (b *Block) pruneTxSignatures() {
	var txs = trie.New()
	iterator := b.Txs()
	for iterator.HasNext() {
		next := iterator.Next()
		next.DropSignature()
		txs.InsertOrUpdate([]byte(next.Hash().Hex()), next.Serialized())

	}
	b.txs = txs
}

func CreateBlockHeaderFromMessage(header *pb.BlockHeader) *Header {
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
func CreateBlockHeaderFromStorage(header *pb.BlockHeaderS) *Header {
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

func (h *Header) GetMessage() *pb.BlockHeader {
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

func (h *Header) ToStorageProto() *pb.BlockHeaderS {
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

func (h *Header) SetHash() {
	h.hash = HashHeader(*h)
}

func HashHeader(h Header) common.Hash {
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
}

//returns 96 byte of header signature
func (h *Header) Sign(key *crypto.PrivateKey) *crypto.Signature {
	sig := crypto.Sign(h.hash.Bytes(), key)
	if sig == nil {
		log.Error("Can't sign message")
	}

	return sig
}
