package blockchain

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/gagarinchain/network/common/api"
	"github.com/gagarinchain/network/common/eth/common"
	"github.com/gagarinchain/network/common/eth/crypto"
	"github.com/gagarinchain/network/common/protobuff"
	"github.com/gagarinchain/network/common/trie"
	"github.com/gagarinchain/network/common/tx"
	"github.com/golang/protobuf/proto"
	"time"
)

type BlockImpl struct {
	header    api.Header
	qc        api.QuorumCertificate
	signature *crypto.SignatureAggregate
	txs       *trie.FixedLengthHexKeyMerkleTrie
	data      []byte
}

func (b *BlockImpl) TxsCount() int {
	return len(b.txs.Values())
}

func (b *BlockImpl) Txs() tx.Iterator {
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

func (b *BlockImpl) AddTransaction(t *tx.Transaction) {
	key := []byte(t.Hash().Hex())
	b.txs.InsertOrUpdate(key, t.Serialized())
}

type ByHeight []api.Block

func (h ByHeight) Len() int           { return len(h) }
func (h ByHeight) Less(i, j int) bool { return h[i].Height() < h[j].Height() }
func (h ByHeight) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (b *BlockImpl) Header() api.Header {
	return b.header
}
func (b *BlockImpl) Data() []byte {
	return b.data
}
func (b *BlockImpl) Height() int32 {
	return b.Header().Height()
}

func (b *BlockImpl) QC() api.QuorumCertificate {
	return b.qc
}

//recalculate hash, since we add new field to block
func (b *BlockImpl) SetQC(qc api.QuorumCertificate) {
	b.qc = qc
	b.header.SetQCHash(qc.GetHash())
	b.header.SetHash()
}

func (b *BlockImpl) Signature() *crypto.SignatureAggregate {
	return b.signature
}
func (b *BlockImpl) SetSignature(s *crypto.SignatureAggregate) {
	b.signature = s
	b.pruneTxSignatures()
}

func (b *BlockImpl) QRef() api.Header {
	return b.QC().QrefBlock()
}

func CreateGenesisBlock() (zero api.Block) {
	data := []byte("Zero")
	zeroHeader := createHeader(0, common.BytesToHash(make([]byte, common.HashLength)), common.BytesToHash(make([]byte, common.HashLength)),
		common.BytesToHash(make([]byte, common.HashLength)), crypto.Keccak256Hash(),
		crypto.Keccak256Hash(data), common.BytesToHash(make([]byte, common.HashLength)),
		time.Date(2019, time.April, 12, 0, 0, 0, 0, time.UTC).Round(time.Millisecond))
	zeroHeader.SetHash()
	zero = &BlockImpl{header: zeroHeader, data: data, qc: CreateQuorumCertificate(crypto.EmptyAggregateSignatures(), zeroHeader),
		txs: trie.New(), signature: crypto.EmptyAggregateSignatures()}

	return zero
}

func CreateBlockFromMessage(block *pb.Block) api.Block {
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
		spew.Dump(tpb)
		spew.Dump(t)
		txs.InsertOrUpdate([]byte(t.Hash().Hex()), t.Serialized())
	}

	var signature *crypto.SignatureAggregate
	if block.GetSignatureAggregate() != nil {
		signature = crypto.AggregateFromProto(block.SignatureAggregate)
	}
	return &BlockImpl{header: header, signature: signature, qc: cert, data: block.Data.Data, txs: txs}
}

func CreateBlockFromStorage(block *pb.BlockS) api.Block {
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
	return &BlockImpl{header: header, signature: signature, qc: cert, data: block.Data.Data, txs: txs}
}

func (b *BlockImpl) GetMessage() *pb.Block {
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

func (b *BlockImpl) ToStorageProto() *pb.BlockS {
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

func (b *BlockImpl) pruneTxSignatures() {
	var txs = trie.New()
	iterator := b.Txs()
	for iterator.HasNext() {
		next := iterator.Next()
		next.DropSignature()
		txs.InsertOrUpdate([]byte(next.Hash().Hex()), next.Serialized())

	}
	b.txs = txs
}

func (b *BlockImpl) Serialize() ([]byte, error) {
	bytes, e := proto.Marshal(b.ToStorageProto())
	if e != nil {
		return nil, e
	}
	return bytes, nil
}
