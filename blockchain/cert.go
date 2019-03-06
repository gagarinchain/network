package blockchain

import (
	"github.com/poslibp2p/eth/common"
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

func (qc *QuorumCertificate) IsGenesisCert() bool {
	return qc.qrefBlock == nil && common.Bytes2Hex(qc.signatureAggregate) == "FFFFFFFFFFFFFFFF"
}

func (qc *QuorumCertificate) SignatureAggregate() []byte {
	return qc.signatureAggregate
}

func (qc *QuorumCertificate) QrefBlock() *Header {
	return qc.qrefBlock
}

func (qc *QuorumCertificate) GetMessage() *pb.QuorumCertificate {
	return &pb.QuorumCertificate{Header: qc.QrefBlock().GetMessage(), SignatureAggregate: qc.SignatureAggregate()}
}
