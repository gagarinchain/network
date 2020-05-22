package hotstuff

import (
	"errors"
	"github.com/gagarinchain/common"
	"github.com/gagarinchain/common/api"
	"github.com/gagarinchain/common/protobuff"
	"github.com/gagarinchain/network/blockchain"
)

type EpochStartValidator struct {
	committee []*common.Peer
}

func NewEpochStartValidator(committee []*common.Peer) *EpochStartValidator {
	return &EpochStartValidator{committee: committee}
}

func (ev *EpochStartValidator) GetId() interface{} {
	return "EpochStart"
}

func (ev *EpochStartValidator) IsValid(entity interface{}) (bool, error) {
	if entity == nil {
		return false, errors.New("entity is nil")
	}
	epoch := entity.(*Epoch)

	var contains bool
	for _, c := range ev.committee {
		if c.GetAddress() == epoch.Sender().GetAddress() {
			contains = true
			break
		}
	}
	if !contains {
		return false, errors.New("signature is not valid, unknown peer")
	}

	return true, nil
}

func (ev *EpochStartValidator) Supported(mType pb.Message_MessageType) bool {
	return mType == pb.Message_EPOCH_START
}

type ProposalValidator struct {
	committee []*common.Peer
}

func NewProposalValidator(committee []*common.Peer) *ProposalValidator {
	return &ProposalValidator{committee: committee}
}

func (p *ProposalValidator) GetId() interface{} {
	return "Proposal"
}

func (p *ProposalValidator) IsValid(entity interface{}) (bool, error) {
	if entity == nil {
		return false, errors.New("entity is nil")
	}
	proposal := entity.(api.Proposal)

	var contains bool
	for _, c := range p.committee {
		if c.GetAddress() == proposal.Sender().GetAddress() {
			contains = true
			break
		}
	}
	if !contains {
		return false, errors.New("signature is not valid, unknown peer")
	}

	b, e := proposal.HQC().IsValid(proposal.HQC().GetHash(), common.PeersToPubs(p.committee))
	if !b || e != nil {
		return false, e
	}

	hash := blockchain.HashHeader(proposal.NewBlock().Header())
	if proposal.NewBlock().Header().Hash() != hash {
		return false, errors.New("block hash is not valid")
	}

	//todo validate block

	return true, nil

}

func (p *ProposalValidator) Supported(mType pb.Message_MessageType) bool {
	return mType == pb.Message_PROPOSAL
}

type VoteValidator struct {
	committee []*common.Peer
}

func NewVoteValidator(committee []*common.Peer) *VoteValidator {
	return &VoteValidator{committee: committee}
}

func (p *VoteValidator) GetId() interface{} {
	return "Proposal"
}

func (p *VoteValidator) IsValid(entity interface{}) (bool, error) {
	if entity == nil {
		return false, errors.New("entity is nil")
	}
	vote := entity.(api.Vote)

	var contains bool
	for _, c := range p.committee {
		if c.GetAddress() == vote.Sender().GetAddress() {
			contains = true
			break
		}
	}
	if !contains {
		return false, errors.New("signature is not valid, unknown peer")
	}

	b, e := vote.HQC().IsValid(vote.HQC().GetHash(), common.PeersToPubs(p.committee))
	if !b || e != nil {
		return false, e
	}

	return true, nil
}

func (p *VoteValidator) Supported(mType pb.Message_MessageType) bool {
	return mType == pb.Message_VOTE
}
