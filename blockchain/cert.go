package blockchain

import (
	"github.com/gagarinchain/network/common/eth/common"
	"github.com/gagarinchain/network/common/eth/crypto"
	"github.com/gagarinchain/network/common/protobuff"
	"github.com/gogo/protobuf/proto"
)

type QuorumCertificate struct {
	signatureAggregate *crypto.SignatureAggregate
	qrefBlock          *Header
}

func CreateQuorumCertificate(aggregate *crypto.SignatureAggregate, qrefBlock *Header) *QuorumCertificate {
	return &QuorumCertificate{signatureAggregate: aggregate, qrefBlock: qrefBlock}
}
func CreateQuorumCertificateFromMessage(msg *pb.QuorumCertificate) *QuorumCertificate {
	return CreateQuorumCertificate(crypto.AggregateFromProto(msg.SignatureAggregate), CreateBlockHeaderFromMessage(msg.Header))
}

func (qc *QuorumCertificate) SignatureAggregate() *crypto.SignatureAggregate {
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
	return &pb.QuorumCertificate{Header: m, SignatureAggregate: qc.SignatureAggregate().ToProto()}
}

//Calculates signature hash and concatenates it with QREF block hash
func (qc *QuorumCertificate) GetHash() common.Hash {
	aggrPb := qc.signatureAggregate.ToProto()
	marshal, e := proto.Marshal(aggrPb)
	if e != nil {
		log.Error("can't marshal signature aggregate")
		return common.Hash{}
	}
	bytes := append(crypto.Keccak256(marshal), qc.QrefBlock().Hash().Bytes()...)
	return common.BytesToHash(crypto.Keccak256(bytes))
}
