package hotstuff

import (
	"fmt"
	bc "github.com/gagarinchain/network/blockchain"
	comm "github.com/gagarinchain/network/common"
	"github.com/gagarinchain/network/common/eth/crypto"
	msg "github.com/gagarinchain/network/common/message"
	"github.com/gagarinchain/network/common/protobuff"
	"github.com/golang/protobuf/ptypes"
	"github.com/pkg/errors"
)

type Vote struct {
	Sender    *comm.Peer
	Header    *bc.Header
	Signature *crypto.Signature //We should not allow to change header if we want signature to be consistent with block
	HQC       *bc.QuorumCertificate
}

func CreateVote(newBlock *bc.Header, hqc *bc.QuorumCertificate, sender *comm.Peer) *Vote {
	return &Vote{Sender: sender, Header: newBlock, HQC: hqc}
}

func (v *Vote) Sign(key *crypto.PrivateKey) {
	v.Signature = v.Header.Sign(key)
}

func CreateVoteFromMessage(msg *msg.Message) (*Vote, error) {
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
	vote.Signature = sign
	return vote, nil
}

func (v *Vote) GetMessage() *pb.VotePayload {
	return &pb.VotePayload{Cert: v.HQC.GetMessage(), Header: v.Header.GetMessage(), Signature: v.Signature.ToProto()}
}
