package hotstuff

import (
	"errors"
	"fmt"
	comm "github.com/gagarinchain/common"
	"github.com/gagarinchain/common/api"
	"github.com/gagarinchain/common/eth/crypto"
	msg "github.com/gagarinchain/common/message"
	"github.com/gagarinchain/common/protobuff"
	bc "github.com/gagarinchain/network/blockchain"
	"github.com/golang/protobuf/ptypes"
)

type ProposalImpl struct {
	sender   *comm.Peer
	newBlock api.Block
	//We should not allow to change header if we want signature to be consistent with block
	signature *crypto.Signature
	hqc       api.QuorumCertificate
}

func (p *ProposalImpl) Sender() *comm.Peer {
	return p.sender
}

func (p *ProposalImpl) NewBlock() api.Block {
	return p.newBlock
}

func (p *ProposalImpl) Signature() *crypto.Signature {
	return p.signature
}

func (p *ProposalImpl) HQC() api.QuorumCertificate {
	return p.hqc
}

func (p *ProposalImpl) GetMessage() *pb.ProposalPayload {
	return &pb.ProposalPayload{Cert: p.HQC().GetMessage(), Block: p.NewBlock().GetMessage(), Signature: p.Signature().ToProto()}
}

func CreateProposal(newBlock api.Block, hqc api.QuorumCertificate, peer *comm.Peer) *ProposalImpl {
	return &ProposalImpl{sender: peer, newBlock: newBlock, hqc: hqc}
}

func (p *ProposalImpl) Sign(key *crypto.PrivateKey) {
	p.signature = p.NewBlock().Header().Sign(key)
}

func CreateProposalFromMessage(msg *msg.Message) (*ProposalImpl, error) {
	if msg.Type != pb.Message_PROPOSAL {
		return nil, errors.New(fmt.Sprintf("wrong message type, expected [%v], but got [%v]",
			pb.Message_PROPOSAL.String(), msg.Type))
	}
	pp := &pb.ProposalPayload{}
	if err := ptypes.UnmarshalAny(msg.Payload, pp); err != nil {
		log.Error("Couldn'T unmarshal response", err)
	}
	block, err := bc.CreateBlockFromMessage(pp.Block)
	if err != nil {
		return nil, err
	}
	qc := bc.CreateQuorumCertificateFromMessage(pp.Cert)

	sign := crypto.SignatureFromProto(pp.Signature)
	res := crypto.Verify(bc.HashHeader(block.Header()).Bytes(), sign)
	if !res {
		return nil, errors.New("bad signature")
	}

	a := crypto.PubkeyToAddress(crypto.NewPublicKey(sign.Pub()))

	msg.Source().SetAddress(a)
	msg.Source().SetPublicKey(crypto.NewPublicKey(sign.Pub()))

	return CreateProposal(block, qc, msg.Source()), nil
}
