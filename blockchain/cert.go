package blockchain

import (
	"github.com/poslibp2p/eth/common"
	"github.com/poslibp2p/eth/crypto"
	"github.com/poslibp2p/message/protobuff"
)

type QuorumCertificate struct {
	signatureAggregate []byte
	qrefBlock          *Header
}

func CreateQuorumCertificate(aggregate []byte, qrefBlock *Header) *QuorumCertificate {
	return &QuorumCertificate{signatureAggregate: aggregate, qrefBlock: qrefBlock}
}
func CreateQuorumCertificateFromMessage(msg *pb.QuorumCertificate) *QuorumCertificate {
	return CreateQuorumCertificate(msg.SignatureAggregate, CreateBlockHeaderFromMessage(msg.Header))
}

func (qc *QuorumCertificate) SignatureAggregate() []byte {
	return qc.signatureAggregate
}

func (qc *QuorumCertificate) QrefBlock() *Header {
	return qc.qrefBlock
}

func (qc *QuorumCertificate) GetMessage() *pb.QuorumCertificate {
	var m *pb.BlockHeader
	if qc.QrefBlock() != nil {
		m = qc.QrefBlock().GetMessage()
	}
	return &pb.QuorumCertificate{Header: m, SignatureAggregate: qc.SignatureAggregate()}
}

func (qc *QuorumCertificate) GetHash() common.Hash {
	bytes := append(crypto.Keccak256(qc.signatureAggregate), qc.QrefBlock().Hash().Bytes()...)
	return common.BytesToHash(crypto.Keccak256(bytes))
}
