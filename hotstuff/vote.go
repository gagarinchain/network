package hotstuff

import (
	"fmt"
	comm "github.com/gagarinchain/common"
	"github.com/gagarinchain/common/api"
	"github.com/gagarinchain/common/eth/crypto"
	msg "github.com/gagarinchain/common/message"
	"github.com/gagarinchain/common/protobuff"
	bc "github.com/gagarinchain/network/blockchain"
	"github.com/golang/protobuf/ptypes"
	"github.com/pkg/errors"
)

type VoteImpl struct {
	sender    *comm.Peer
	header    api.Header
	signature *crypto.Signature //We should not allow to change header if we want signature to be consistent with block
	cert      api.Certificate
}

func (v *VoteImpl) Sender() *comm.Peer {
	return v.sender
}

func (v *VoteImpl) Header() api.Header {
	return v.header
}

func (v *VoteImpl) Signature() *crypto.Signature {
	return v.signature
}

func (v *VoteImpl) Cert() api.Certificate {
	return v.cert
}

func CreateVote(newBlock api.Header, cert api.Certificate, sender *comm.Peer) *VoteImpl {
	return &VoteImpl{sender: sender, header: newBlock, cert: cert}
}

func (v *VoteImpl) Sign(key *crypto.PrivateKey) {
	v.signature = v.Header().Sign(key)
}

func (v *VoteImpl) GetMessage() *pb.VotePayload {
	v2 := &pb.VotePayload{Header: v.Header().GetMessage(), Signature: v.Signature().ToProto()}

	switch v.cert.Type() {
	case api.SC:
		sc := v.cert.(api.SynchronizeCertificate)
		v2.Cert = &pb.VotePayload_Sc{Sc: sc.GetMessage()}
	case api.QRef:
		fallthrough
	case api.Empty:
		sc := v.cert.(api.QuorumCertificate)
		v2.Cert = &pb.VotePayload_Qc{Qc: sc.GetMessage()}
	default:
		log.Error("unknown cert type")
		return nil
	}

	return v2
}

func CreateVoteFromMessage(msg *msg.Message) (api.Vote, error) {
	if msg.Type != pb.Message_VOTE {
		return nil, errors.New(fmt.Sprintf("wrong message type, expected [%v], but got [%v]",
			pb.Message_VOTE.String(), msg.Type))
	}

	vp := &pb.VotePayload{}
	if err := ptypes.UnmarshalAny(msg.Payload, vp); err != nil {
		log.Error("Couldn'T unmarshal response", err)
	}

	var cert api.Certificate
	if vp.GetQc() != nil {
		cert = bc.CreateQuorumCertificateFromMessage(vp.GetQc())

	}
	if vp.GetSc() != nil {
		cert = bc.CreateSynchronizeCertificateFromMessage(vp.GetSc())
	}

	header := bc.CreateBlockHeaderFromMessage(vp.Header)

	sign := crypto.SignatureFromProto(vp.Signature)
	res := crypto.Verify(header.Hash().Bytes(), sign)
	if !res {
		return nil, errors.New("bad signature")
	}

	a := crypto.PubkeyToAddress(crypto.NewPublicKey(sign.Pub()))
	msg.Source().SetAddress(a)
	msg.Source().SetPublicKey(crypto.NewPublicKey(sign.Pub()))

	vote := CreateVote(header, cert, msg.Source())
	vote.signature = sign
	return vote, nil
}
