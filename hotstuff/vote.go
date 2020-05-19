package hotstuff

import (
	"fmt"
	bc "github.com/gagarinchain/network/blockchain"
	comm "github.com/gagarinchain/network/common"
	"github.com/gagarinchain/network/common/api"
	"github.com/gagarinchain/network/common/eth/crypto"
	msg "github.com/gagarinchain/network/common/message"
	"github.com/gagarinchain/network/common/protobuff"
	"github.com/golang/protobuf/ptypes"
	"github.com/pkg/errors"
)

type VoteImpl struct {
	sender    *comm.Peer
	header    api.Header
	signature *crypto.Signature //We should not allow to change header if we want signature to be consistent with block
	hqc       api.QuorumCertificate
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

func (v *VoteImpl) HQC() api.QuorumCertificate {
	return v.hqc
}

func CreateVote(newBlock api.Header, hqc api.QuorumCertificate, sender *comm.Peer) *VoteImpl {
	return &VoteImpl{sender: sender, header: newBlock, hqc: hqc}
}

func (v *VoteImpl) Sign(key *crypto.PrivateKey) {
	v.signature = v.Header().Sign(key)
}

func (v *VoteImpl) GetMessage() *pb.VotePayload {
	return &pb.VotePayload{Cert: v.HQC().GetMessage(), Header: v.Header().GetMessage(), Signature: v.Signature().ToProto()}
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
	qc := bc.CreateQuorumCertificateFromMessage(vp.Cert)
	header := bc.CreateBlockHeaderFromMessage(vp.Header)

	sign := crypto.SignatureFromProto(vp.Signature)
	res := crypto.Verify(header.Hash().Bytes(), sign)
	if !res {
		return nil, errors.New("bad signature")
	}

	a := crypto.PubkeyToAddress(crypto.NewPublicKey(sign.Pub()))
	msg.Source().SetAddress(a)
	msg.Source().SetPublicKey(crypto.NewPublicKey(sign.Pub()))

	vote := CreateVote(header, qc, msg.Source())
	vote.signature = sign
	return vote, nil
}
