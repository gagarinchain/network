package hotstuff

import (
	"github.com/golang/protobuf/ptypes"
	"github.com/poslibp2p/blockchain"
	msg "github.com/poslibp2p/message"
	"github.com/poslibp2p/message/protobuff"
	"github.com/poslibp2p/network"
)

type Proposal struct {
	Sender   *network.Peer
	NewBlock *blockchain.Block
	HQC      *blockchain.QuorumCertificate
}

func (p *Proposal) GetMessage() (*msg.Message, error) {

	//TODO FILL ME
	payload := &pb.ProposalPayload{}

	any, e := ptypes.MarshalAny(payload)
	if e != nil {
		return nil, e
	}
	m := msg.CreateMessage(pb.Message_PROPOSAL, p.Sender.GetPrivateKey(), any)

	return m, nil
}
