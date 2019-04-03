package hotstuff

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"github.com/golang/protobuf/ptypes"
	bc "github.com/poslibp2p/blockchain"
	"github.com/poslibp2p/eth/common"
	"github.com/poslibp2p/eth/crypto"
	msg "github.com/poslibp2p/message"
	"github.com/poslibp2p/message/protobuff"
)

type Proposal struct {
	Sender   *msg.Peer
	NewBlock *bc.Block
	//We should not allow to change header if we want signature to be consistent with block
	Signature []byte
	HQC       *bc.QuorumCertificate
}

func (p *Proposal) GetMessage() *pb.ProposalPayload {
	return &pb.ProposalPayload{Cert: p.HQC.GetMessage(), Block: p.NewBlock.GetMessage(), Signature: p.Signature}

}

func CreateProposal(newBlock *bc.Block, hqc *bc.QuorumCertificate, me *msg.Peer) *Proposal {
	return &Proposal{Sender: me, NewBlock: newBlock, HQC: hqc}
}

func (p *Proposal) Sign(key *ecdsa.PrivateKey) {
	p.Signature = p.NewBlock.Header().Sign(key)
}

func CreateProposalFromMessage(msg *msg.Message) (*Proposal, error) {
	if msg.Type != pb.Message_PROPOSAL {
		return nil, errors.New(fmt.Sprintf("wrong message type, expected [%v], but got [%v]",
			pb.Message_PROPOSAL.String(), msg.Type))
	}
	pp := &pb.ProposalPayload{}
	if err := ptypes.UnmarshalAny(msg.Payload, pp); err != nil {
		log.Error("Couldn't unmarshal response", err)
	}

	block := bc.CreateBlockFromMessage(pp.Block)
	qc := bc.CreateQuorumCertificateFromMessage(pp.Cert)

	pub, e := crypto.SigToPub(block.Header().Hash().Bytes(), pp.Signature)
	if e != nil {
		return nil, errors.New("bad signature")
	}
	a := common.BytesToAddress(crypto.FromECDSAPub(pub))

	msg.Source().SetAddress(a)

	return CreateProposal(block, qc, msg.Source()), nil
}
