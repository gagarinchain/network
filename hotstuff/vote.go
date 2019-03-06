package hotstuff

import (
	"fmt"
	"github.com/golang/protobuf/ptypes"
	"github.com/pkg/errors"
	bc "github.com/poslibp2p/blockchain"
	"github.com/poslibp2p/message"
	"github.com/poslibp2p/message/protobuff"
	"github.com/poslibp2p/network"
)

type Vote struct {
	Sender   *network.Peer
	NewBlock *bc.Block
	HQC      *bc.QuorumCertificate
}

func CreateVote(newBlock *bc.Block, hqc *bc.QuorumCertificate, me *network.Peer) *Vote {
	return &Vote{Sender: me, NewBlock: newBlock, HQC: hqc}
}

func CreateVoteFromMessage(msg *message.Message, p *network.Peer) (*Vote, error) {
	if msg.Type != pb.Message_VOTE {
		return nil, errors.New(fmt.Sprintf("wrong message type, expected [%v], but got [%v]",
			pb.Message_VOTE.String(), msg.Type))
	}

	vp := &pb.VotePayload{}
	if err := ptypes.UnmarshalAny(msg.Payload, vp); err != nil {
		log.Error("Couldn't unmarshal response", err)
	}
	qc := bc.CreateQuorumCertificateFromMessage(vp.Cert)
	return CreateVote(bc.CreateBlockFromMessage(vp.Block), qc, p), nil
}

func (v *Vote) GetMessage() *pb.VotePayload {
	return &pb.VotePayload{Cert: v.HQC.GetMessage(), Block: v.NewBlock.GetMessage()}
}
