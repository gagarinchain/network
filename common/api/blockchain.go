package api

import (
	"github.com/emirpasic/gods/maps/treemap"
	"github.com/gagarinchain/network/blockchain/state"
	"github.com/gagarinchain/network/common/eth/common"
	"github.com/gagarinchain/network/common/eth/crypto"
	pb "github.com/gagarinchain/network/common/protobuff"
	"github.com/gagarinchain/network/common/tx"
	"time"
)

type Blockchain interface {
	GetBlockByHash(hash common.Hash) (block Block)
	GetBlockByHeight(height int32) (res []Block)
	GetFork(height int32, headHash common.Hash) (res []Block)

	Contains(hash common.Hash) bool
	GetThreeChain(twoHash common.Hash) (zero Block, one Block, two Block)
	OnCommit(b Block) (toCommit []Block, orphans *treemap.Map, err error)
	GetHead() Block
	GetHeadRecord() *state.Record
	GetTopHeight() int32
	GetTopHeightBlocks() []Block
	AddBlock(block Block) error
	RemoveBlock(block Block) error
	GetGenesisBlock() Block
	GetGenesisCert() QuorumCertificate
	IsSibling(sibling Header, ancestor Header) bool
	NewBlock(parent Block, qc QuorumCertificate, data []byte) Block
	PadEmptyBlock(head Block, qc QuorumCertificate) Block
	GetGenesisBlockSignedHash(key *crypto.PrivateKey) *crypto.Signature
	ValidateGenesisBlockSignature(signature *crypto.Signature, address common.Address) bool
	GetTopCommittedBlock() Block
	UpdateGenesisBlockQC(certificate QuorumCertificate)
	SetProposerGetter(proposerGetter ProposerForHeight)
}

type Block interface {
	TxsCount() int
	Txs() tx.Iterator
	AddTransaction(t *tx.Transaction)
	Header() Header
	Data() []byte
	Height() int32
	QC() QuorumCertificate
	SetQC(qc QuorumCertificate)
	Signature() *crypto.SignatureAggregate
	SetSignature(s *crypto.SignatureAggregate)
	QRef() Header
	GetMessage() *pb.Block
	ToStorageProto() *pb.BlockS
	Serialize() ([]byte, error)
}

type Header interface {
	DataHash() common.Hash
	StateHash() common.Hash
	Height() int32
	Hash() common.Hash
	TxHash() common.Hash
	QCHash() common.Hash
	SetQCHash(hash common.Hash)
	Parent() common.Hash
	Timestamp() time.Time
	IsGenesisBlock() bool
	GetMessage() *pb.BlockHeader
	ToStorageProto() *pb.BlockHeaderS
	SetHash()
	Sign(key *crypto.PrivateKey) *crypto.Signature
}

type QuorumCertificate interface {
	SignatureAggregate() *crypto.SignatureAggregate
	QrefBlock() Header
	GetMessage() *pb.QuorumCertificate
	GetHash() common.Hash
	IsValid(qcHash common.Hash, committee []*crypto.PublicKey) (bool, error)
	ToStorageProto() *pb.QuorumCertificateS
}
