package blockchain

import (
	"bytes"
	"errors"
	"github.com/gagarinchain/common/api"
	"github.com/gagarinchain/common/eth/common"
	"github.com/gagarinchain/common/eth/crypto"
	"github.com/gagarinchain/common/protobuff"
	"github.com/gogo/protobuf/proto"
)

type QuorumCertificateImpl struct {
	signatureAggregate *crypto.SignatureAggregate
	qrefBlock          api.Header
}

func CreateQuorumCertificate(aggregate *crypto.SignatureAggregate, qrefBlock api.Header) api.QuorumCertificate {
	return &QuorumCertificateImpl{signatureAggregate: aggregate, qrefBlock: qrefBlock}
}
func CreateQuorumCertificateFromMessage(msg *pb.QuorumCertificate) api.QuorumCertificate {
	return CreateQuorumCertificate(crypto.AggregateFromProto(msg.SignatureAggregate), CreateBlockHeaderFromMessage(msg.Header))
}

func (qc *QuorumCertificateImpl) SignatureAggregate() *crypto.SignatureAggregate {
	return qc.signatureAggregate
}

func (qc *QuorumCertificateImpl) QrefBlock() api.Header {
	return qc.qrefBlock
}

func (qc *QuorumCertificateImpl) GetMessage() *pb.QuorumCertificate {
	var m *pb.BlockHeader
	if qc.QrefBlock() != nil {
		m = qc.QrefBlock().GetMessage()
	}
	return &pb.QuorumCertificate{Header: m, SignatureAggregate: qc.SignatureAggregate().ToProto()}
}

//Calculates signature hash and concatenates it with QREF block hash
func (qc *QuorumCertificateImpl) GetHash() common.Hash {
	aggrPb := qc.signatureAggregate.ToProto()
	marshal, e := proto.Marshal(aggrPb)
	if e != nil {
		log.Error("can't marshal signature aggregate")
		return common.Hash{}
	}
	bytes := append(crypto.Keccak256(marshal), qc.QrefBlock().Hash().Bytes()...)
	return common.BytesToHash(crypto.Keccak256(bytes))
}

func (qc *QuorumCertificateImpl) IsValid(qcHash common.Hash, committee []*crypto.PublicKey) (bool, error) {
	calculated := qc.GetHash()

	//Skip qc checks for genesis QC
	if qc.QrefBlock().IsGenesisBlock() {
		return true, nil
	}

	if !bytes.Equal(calculated.Bytes(), qcHash.Bytes()) {
		return false, errors.New("QC hash is not valid")
	}

	if qc.signatureAggregate.N() < 2*len(committee)/3+1 {
		return false, errors.New("QC contains less than 2f + 1 signatures")
	}
	if qc.SignatureAggregate().IsValid(qc.QrefBlock().Hash().Bytes(), committee) {
		return true, nil
	}
	return false, errors.New("QC is not valid")
}

func (qc *QuorumCertificateImpl) ToStorageProto() *pb.QuorumCertificateS {
	var m *pb.BlockHeaderS
	if qc.QrefBlock() != nil {
		m = qc.QrefBlock().ToStorageProto()
	}
	return &pb.QuorumCertificateS{Header: m, SignatureAggregate: qc.SignatureAggregate().ToStorageProto()}
}
