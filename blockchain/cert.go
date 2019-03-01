package blockchain

import "github.com/poslibp2p/eth/common"

type QuorumCertificate struct {
	signatureAggregate []byte
	qrefBlock          *Header
}

func CreateQuorumCertificate(aggregate []byte, qrefBlock *Header) *QuorumCertificate {
	return &QuorumCertificate{signatureAggregate: aggregate, qrefBlock: qrefBlock}
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
