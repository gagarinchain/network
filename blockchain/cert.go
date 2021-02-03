package blockchain

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/gagarinchain/common/api"
	"github.com/gagarinchain/common/eth/common"
	"github.com/gagarinchain/common/eth/crypto"
	"github.com/gagarinchain/common/protobuff"
	"github.com/golang/protobuf/proto"
)

type QuorumCertificateImpl struct {
	certType           api.CertType
	signatureAggregate *crypto.SignatureAggregate
	qrefBlock          api.Header
}

func (qc *QuorumCertificateImpl) ToBytes() ([]byte, error) {
	p := qc.ToStorageProto()
	return proto.Marshal(p)
}

func (qc *QuorumCertificateImpl) FromBytes(b []byte) error {
	p := &pb.QuorumCertificateS{}
	if err := proto.Unmarshal(b, p); err != nil {
		return err
	}

	qc.FromStorage(p)

	return nil
}

func (qc *QuorumCertificateImpl) FromStorage(p *pb.QuorumCertificateS) {
	var t api.CertType
	switch p.Type {
	case pb.QuorumCertificateS_BLOCK:
		t = api.QRef
	case pb.QuorumCertificateS_EMPTY:
		t = api.Empty
	}

	qc.signatureAggregate = crypto.AggregateFromStorage(p.SignatureAggregate)
	qc.certType = t
	qc.qrefBlock = CreateBlockHeaderFromStorage(p.Header)
}

type SynchronizeCertificateImpl struct {
	signatureAggregate *crypto.SignatureAggregate
	certType           api.CertType
	height             int32
	voting             bool
}

func (sc *SynchronizeCertificateImpl) ToBytes() ([]byte, error) {
	p := sc.ToStorageProto()
	return proto.Marshal(p)
}

func (sc *SynchronizeCertificateImpl) FromBytes(b []byte) error {
	p := &pb.SynchronizeCertificateS{}
	if err := proto.Unmarshal(b, p); err != nil {
		return err
	}
	sc.certType = api.SC
	sc.height = p.Height
	sc.voting = false
	sc.signatureAggregate = crypto.AggregateFromStorage(p.SignatureAggregate)

	return nil
}

func (sc *SynchronizeCertificateImpl) SignatureAggregate() *crypto.SignatureAggregate {
	return sc.signatureAggregate
}

func (sc *SynchronizeCertificateImpl) Height() int32 {
	return sc.height
}
func (sc *SynchronizeCertificateImpl) Type() api.CertType {
	return sc.certType
}

func (sc *SynchronizeCertificateImpl) GetMessage() *pb.SynchronizeCertificate {
	return &pb.SynchronizeCertificate{Height: sc.height, SignatureAggregate: sc.SignatureAggregate().ToProto()}
}

func (sc *SynchronizeCertificateImpl) GetHash() common.Hash {
	return CalculateSyncHash(sc.height, sc.voting)
}
func CalculateSyncHash(height int32, voting bool) common.Hash {
	buf := make([]byte, binary.MaxVarintLen64)
	binary.PutVarint(buf, int64(height))

	var v byte
	if voting {
		v = 1
	}
	hash := crypto.Keccak256Hash(append(buf, v))
	return hash
}

func (sc *SynchronizeCertificateImpl) IsValid(hash common.Hash, committee []*crypto.PublicKey) (bool, error) {
	calculated := sc.GetHash()

	if !bytes.Equal(calculated.Bytes(), hash.Bytes()) {
		return false, errors.New("SC hash is not valid")
	}

	if sc.signatureAggregate.N() < 2*len(committee)/3+1 {
		return false, errors.New("SC contains less than 2f + 1 signatures")
	}

	if sc.SignatureAggregate().IsValid(sc.GetHash().Bytes(), committee) {
		return true, nil
	}
	return false, errors.New("SC is not valid")
}

func (sc *SynchronizeCertificateImpl) ToStorageProto() *pb.SynchronizeCertificateS {
	return &pb.SynchronizeCertificateS{
		Height:             sc.height,
		SignatureAggregate: sc.signatureAggregate.ToStorageProto(),
	}
}

func CreateSynchronizeCertificate(aggregate *crypto.SignatureAggregate, height int32) api.SynchronizeCertificate {
	return &SynchronizeCertificateImpl{certType: api.SC, signatureAggregate: aggregate, height: height}
}
func CreateSynchronizeCertificateFromMessage(msg *pb.SynchronizeCertificate) api.SynchronizeCertificate {
	return CreateSynchronizeCertificate(crypto.AggregateFromProto(msg.SignatureAggregate), msg.Height)
}

func (qc *QuorumCertificateImpl) Type() api.CertType {
	return qc.certType
}

func (qc *QuorumCertificateImpl) Height() int32 {
	return qc.QrefBlock().Height()
}

func CreateQuorumCertificate(aggregate *crypto.SignatureAggregate, qrefBlock api.Header, t api.CertType) api.QuorumCertificate {
	return &QuorumCertificateImpl{signatureAggregate: aggregate, qrefBlock: qrefBlock, certType: t}
}
func CreateQuorumCertificateFromMessage(msg *pb.QuorumCertificate) api.QuorumCertificate {
	var t api.CertType
	switch msg.Type {
	case pb.QuorumCertificate_QREF:
		t = api.QRef
	case pb.QuorumCertificate_EMPTY:
		t = api.Empty
	}

	return CreateQuorumCertificate(crypto.AggregateFromProto(msg.SignatureAggregate), CreateBlockHeaderFromMessage(msg.Header), t)
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

	var t pb.QuorumCertificate_CertType
	switch qc.certType {
	case api.QRef:
		t = pb.QuorumCertificate_QREF
	case api.Empty:
		t = pb.QuorumCertificate_EMPTY
	}

	return &pb.QuorumCertificate{Header: m, SignatureAggregate: qc.SignatureAggregate().ToProto(), Type: t}
}

//It is simply QREF hash
func (qc *QuorumCertificateImpl) GetHash() common.Hash {
	switch qc.certType {
	case api.QRef:
		return qc.QrefBlock().Hash()
	case api.Empty:
		return CalculateSyncHash(qc.Height(), true)
	default:
		return common.Hash{}
	}

}

func (qc *QuorumCertificateImpl) IsValid(qcHash common.Hash, committee []*crypto.PublicKey) (bool, error) {
	if !bytes.Equal(qcHash.Bytes(), qc.GetHash().Bytes()) {
		return false, errors.New("QC.Qref hash is not valid")
	}

	//Skip qc checks for genesis QC
	if qc.QrefBlock().IsGenesisBlock() {
		return true, nil
	}

	if qc.signatureAggregate.N() < 2*len(committee)/3+1 {
		return false, errors.New("QC contains less than 2f + 1 signatures")
	}

	if qc.certType == api.Empty && qc.QrefBlock().TxHash().Hex() != api.EmptyTxHashHex {
		return false, errors.New("empty qc is used for non empty block")
	}

	if qc.SignatureAggregate().IsValid(qc.GetHash().Bytes(), committee) {
		return true, nil
	}

	return false, errors.New("QC is not valid")
}

func (qc *QuorumCertificateImpl) ToStorageProto() *pb.QuorumCertificateS {
	var m *pb.BlockHeaderS
	if qc.QrefBlock() != nil {
		m = qc.QrefBlock().ToStorageProto()
	}

	var t pb.QuorumCertificateS_CertType
	switch qc.certType {
	case api.QRef:
		t = pb.QuorumCertificateS_BLOCK
	case api.Empty:
		t = pb.QuorumCertificateS_EMPTY
	}

	return &pb.QuorumCertificateS{Header: m, SignatureAggregate: qc.SignatureAggregate().ToStorageProto(), Type: t}
}
