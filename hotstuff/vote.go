package hotstuff

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/golang/protobuf/ptypes"
	"github.com/pkg/errors"
	bc "github.com/poslibp2p/blockchain"
	comm "github.com/poslibp2p/common"
	"github.com/poslibp2p/common/eth/common"
	"github.com/poslibp2p/common/eth/crypto"
	msg "github.com/poslibp2p/common/message"
	"github.com/poslibp2p/common/protobuff"
)

type Vote struct {
	Sender    *comm.Peer
	Header    *bc.Header
	Signature []byte //We should not allow to change header if we want signature to be consistent with block
	HQC       *bc.QuorumCertificate
}

func CreateVote(newBlock *bc.Header, hqc *bc.QuorumCertificate, sender *comm.Peer) *Vote {
	return &Vote{Sender: sender, Header: newBlock, HQC: hqc}
}

func (v *Vote) Sign(key *ecdsa.PrivateKey) {
	v.Signature = v.Header.Sign(key)
}

func CreateVoteFromMessage(msg *msg.Message) (*Vote, error) {
	if msg.Type != pb.Message_VOTE {
		return nil, errors.New(fmt.Sprintf("wrong message type, expected [%v], but got [%v]",
			pb.Message_VOTE.String(), msg.Type))
	}

	vp := &pb.VotePayload{}
	if err := ptypes.UnmarshalAny(msg.Payload, vp); err != nil {
		log.Error("Couldn't unmarshal response", err)
	}
	qc := bc.CreateQuorumCertificateFromMessage(vp.Cert)
	header := bc.CreateBlockHeaderFromMessage(vp.Header)

	pub, e := crypto.SigToPub(header.Hash().Bytes(), vp.Signature)
	if e != nil {
		return nil, errors.New("bad signature")
	}
	a := common.BytesToAddress(crypto.FromECDSAPub(pub))
	msg.Source().SetAddress(a)

	return CreateVote(header, qc, msg.Source()), nil
}

func (v *Vote) GetMessage() *pb.VotePayload {
	return &pb.VotePayload{Cert: v.HQC.GetMessage(), Header: v.Header.GetMessage(), Signature: v.Signature}
}
