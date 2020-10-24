package blockchain

import (
	"github.com/gagarinchain/common/api"
	"github.com/gagarinchain/common/eth/common"
	"github.com/gagarinchain/common/eth/crypto"
	"github.com/gagarinchain/common/protobuff"
	"github.com/gagarinchain/common/trie"
	"github.com/gagarinchain/network/blockchain/state"
	"github.com/gagarinchain/network/blockchain/tx"
	"github.com/golang/protobuf/proto"
	"time"
)

type BlockImpl struct {
	header    *HeaderImpl
	qc        api.QuorumCertificate
	signature *crypto.SignatureAggregate
	txs       []api.Transaction
	receipts  []api.Receipt
	data      []byte
}

func (b *BlockImpl) Receipts() []api.Receipt {
	return b.receipts
}

func (b *BlockImpl) SetReceipts(receipts []api.Receipt) {
	b.receipts = receipts
}

func (b *BlockImpl) TxsCount() int {
	return len(b.txs)
}

func (b *BlockImpl) Txs() api.Iterator {
	return tx.NewIterator(b.txs)
}

func (b *BlockImpl) AddTransaction(t api.Transaction) {
	b.txs = append(b.txs, t)
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
	data := []byte(nil)
	zeroHeader := createHeader(0, common.BytesToHash(make([]byte, common.HashLength)), common.BytesToHash(make([]byte, common.HashLength)),
		common.BytesToHash(make([]byte, common.HashLength)), crypto.Keccak256Hash(),
		crypto.Keccak256Hash(data), common.BytesToHash(make([]byte, common.HashLength)),
		time.Date(2019, time.April, 12, 0, 0, 0, 0, time.UTC).Round(time.Millisecond))
	zeroHeader.SetHash()
	zero = &BlockImpl{header: zeroHeader, data: data, qc: CreateQuorumCertificate(crypto.EmptyAggregateSignatures(), zeroHeader), signature: crypto.EmptyAggregateSignatures()}

	return zero
}

func CreateBlockFromMessage(block *pb.Block) api.Block {
	header := CreateBlockHeaderFromMessage(block.Header)
	aggrPb := block.Cert.GetSignatureAggregate()
	cert := CreateQuorumCertificate(crypto.AggregateFromProto(aggrPb), CreateBlockHeaderFromMessage(block.Cert.Header))
	var txs []api.Transaction
	for _, tpb := range block.Txs {
		t, e := tx.CreateTransactionFromMessage(tpb, block.GetSignatureAggregate() != nil)
		if e != nil {
			log.Errorf("Bad transaction, %v", e)
			return nil
		}
		//spew.Dump(tpb)
		//spew.Dump(t)
		txs = append(txs, t)
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
	var txs []api.Transaction
	for _, tpb := range block.Txs {
		t, e := tx.CreateTransactionFromStorage(tpb)
		if e != nil {
			log.Errorf("Bad transaction, %v", e)
			return nil
		}
		txs = append(txs, t)
	}

	var signature *crypto.SignatureAggregate
	if block.SignatureAggregate != nil {
		signature = crypto.AggregateFromStorage(block.SignatureAggregate)
	}

	var receipts []api.Receipt
	for _, r := range block.Receipts {
		receipts = append(receipts, state.ReceiptFromStorage(r))
	}
	return &BlockImpl{header: header, signature: signature, qc: cert, data: block.Data.Data, txs: txs, receipts: receipts}
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

	var receipts []*pb.Receipt
	if len(b.Receipts()) > 0 {
		for _, r := range b.Receipts() {
			receipts = append(receipts, r.ToStorageProto())
		}
	}

	var sign *pb.SignatureAggregateS
	if b.signature != nil {
		sign = b.signature.ToStorageProto()
	}

	return &pb.BlockS{
		Header: b.Header().ToStorageProto(),
		Cert:   qc, SignatureAggregate: sign,
		Data:     &pb.BlockDataS{Data: b.Data()},
		Txs:      txs,
		Receipts: receipts,
	}
}

func (b *BlockImpl) pruneTxSignatures() {
	for _, next := range b.txs {
		next.DropSignature()
	}
}

func (b *BlockImpl) Serialize() ([]byte, error) {
	bytes, e := proto.Marshal(b.ToStorageProto())
	if e != nil {
		return nil, e
	}
	return bytes, nil
}

type BlockBuilderImpl struct {
	block *BlockImpl
}

func (b BlockBuilderImpl) AddTx(tx api.Transaction) api.BlockBuilder {
	b.block.AddTransaction(tx)
	return b
}

func NewBlockBuilderImpl() *BlockBuilderImpl {
	return &BlockBuilderImpl{block: &BlockImpl{}}
}

func NewBlockBuilderFromBlock(block api.Block) *BlockBuilderImpl {
	blockImpl, f := block.(*BlockImpl)
	if !f {
		log.Error("Can create block builder only with *BlockImpl implementation of api.Block")
	}
	return &BlockBuilderImpl{block: blockImpl}
}

func (b BlockBuilderImpl) SetHeader(header api.Header) api.BlockBuilder {
	impl, f := header.(*HeaderImpl)
	if !f {
		log.Critical("Can't set anything except *HeaderImpl as header for BlockImpl")
	}
	b.block.header = impl
	return b
}

func (b BlockBuilderImpl) SetQC(qc api.QuorumCertificate) api.BlockBuilder {
	b.block.qc = qc
	return b
}

func (b BlockBuilderImpl) SetTxs(txs []api.Transaction) api.BlockBuilder {
	b.block.txs = txs
	return b
}

func (b BlockBuilderImpl) SetData(data []byte) api.BlockBuilder {
	b.block.data = data
	return b
}

func (b BlockBuilderImpl) Header() api.Header {
	return b.block.header
}

func (b BlockBuilderImpl) QC() api.QuorumCertificate {
	return b.block.qc
}

func (b BlockBuilderImpl) Txs() []api.Transaction {
	return b.block.txs
}

func (b BlockBuilderImpl) Data() []byte {
	return b.block.data
}

func (b BlockBuilderImpl) Build() api.Block {
	if b.block.header == nil {
		panic("nil header")
	}
	if b.block.qc == nil {
		panic("nil qc")
	}
	//if b.block.data == nil {
	//	panic("nil data")
	//}
	//refresh hashes

	b.block.header.txHash = BuildProof(b.Txs())

	b.block.header.qcHash = b.QC().GetHash()
	b.block.header.dataHash = crypto.Keccak256Hash(b.Data())

	b.block.header.SetHash()

	return b.block
}

func BuildProof(txs []api.Transaction) common.Hash {
	txTrie := trie.New()
	for _, t := range txs {
		txTrie.InsertOrUpdate([]byte(t.Hash().Hex()), t.Serialized())
	}
	return txTrie.Proof()
}
