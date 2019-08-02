package hotstuff

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	bc "github.com/gagarinchain/network/blockchain"
	comm "github.com/gagarinchain/network/common"
	"github.com/gagarinchain/network/common/eth/crypto"
	msg "github.com/gagarinchain/network/common/message"
	"github.com/gagarinchain/network/common/protobuff"
	"github.com/golang/protobuf/ptypes"
)

type Proposal struct {
	Sender   *comm.Peer
	NewBlock *bc.Block
	//We should not allow to change header if we want signature to be consistent with block
	Signature []byte
	HQC       *bc.QuorumCertificate
}

func (p *Proposal) GetMessage() *pb.ProposalPayload {
	return &pb.ProposalPayload{Cert: p.HQC.GetMessage(), Block: p.NewBlock.GetMessage(), Signature: p.Signature}

}

func CreateProposal(newBlock *bc.Block, hqc *bc.QuorumCertificate, peer *comm.Peer) *Proposal {
	return &Proposal{Sender: peer, NewBlock: newBlock, HQC: hqc}
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

	pub, e := crypto.SigToPub(bc.HashHeader(*block.Header()).Bytes(), pp.Signature)

	if e != nil {
		return nil, errors.New("bad signature")
	}
	a := crypto.PubkeyToAddress(*pub)

	msg.Source().SetAddress(a)

	return CreateProposal(block, qc, msg.Source()), nil
}
