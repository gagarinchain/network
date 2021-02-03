package hotstuff

import (
	"errors"
	"github.com/gagarinchain/common"
	"github.com/gagarinchain/common/api"
	"github.com/gagarinchain/common/protobuff"
	"github.com/gagarinchain/network/blockchain"
)

type SyncValidator struct {
	committee      []*common.Peer
	blockValidator api.Validator
}

func NewSyncValidator(committee []*common.Peer) *SyncValidator {
	return &SyncValidator{committee: committee}
}

func (ev *SyncValidator) GetId() interface{} {
	return "Synchronize"
}

func (ev *SyncValidator) IsValid(entity interface{}) (bool, error) {
	if entity == nil {
		return false, errors.New("entity is nil")
	}
	sync := entity.(*SyncImpl)

	var contains bool
	for _, c := range ev.committee {
		if c.GetAddress() == sync.Sender().GetAddress() {
			contains = true
			break
		}
	}
	if !contains {
		log.Error("Bad source address %v", sync.Sender().GetAddress().Hex())
		return false, errors.New("signature is not valid, unknown peer")
	}

	if sync.Cert() != nil {
		b, e := sync.Cert().IsValid(sync.Cert().GetHash(), common.PeersToPubs(ev.committee))
		if !b || e != nil {
			return false, e
		}
	}

	return true, nil
}

func (ev *SyncValidator) Supported(mType pb.Message_MessageType) bool {
	return mType == pb.Message_SYNCHRONIZE
}

type ProposalValidator struct {
	committee      []*common.Peer
	blockValidator api.Validator
}

func NewProposalValidator(committee []*common.Peer, blockValidator api.Validator) *ProposalValidator {
	return &ProposalValidator{committee: committee, blockValidator: blockValidator}
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

	hash := blockchain.HashHeader(proposal.NewBlock().Header())
	if proposal.NewBlock().Header().Hash() != hash {
		return false, errors.New("block hash is not valid")
	}

	b, e := proposal.Cert().IsValid(proposal.Cert().GetHash(), common.PeersToPubs(p.committee))
	if !b || e != nil {
		return false, e
	}

	return p.blockValidator.IsValid(proposal.NewBlock())
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

	b, e := vote.Cert().IsValid(vote.Cert().GetHash(), common.PeersToPubs(p.committee))
	if !b || e != nil {
		return false, e
	}

	return true, nil
}

func (p *VoteValidator) Supported(mType pb.Message_MessageType) bool {
	return mType == pb.Message_VOTE
}
