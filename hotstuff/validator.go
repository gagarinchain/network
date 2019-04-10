package hotstuff

import (
	"errors"
	"github.com/poslibp2p/blockchain"
	"github.com/poslibp2p/common"
	"github.com/poslibp2p/common/protobuff"
)

type EpochStartValidator struct {
	committee []*common.Peer
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

func (p *ProposalValidator) IsValid(entity interface{}) (bool, error) {
	if entity == nil {
		return false, errors.New("entity is nil")
	}
	proposal := entity.(*Proposal)

	var contains bool
	for _, c := range p.committee {
		if c.GetAddress() == proposal.Sender.GetAddress() {
			contains = true
			break
		}
	}
	if !contains {
		return false, errors.New("signature is not valid, unknown peer")
	}

	hash := blockchain.HashHeader(*proposal.NewBlock.Header())
	if proposal.NewBlock.Header().Hash() != hash {
		return false, errors.New("block hash is not valid")
	}

	return true, nil

}

func (p *ProposalValidator) Supported(mType pb.Message_MessageType) bool {
	return mType == pb.Message_PROPOSAL
}

type VoteValidator struct {
	committee []*common.Peer
}

func (p *VoteValidator) IsValid(entity interface{}) (bool, error) {
	if entity == nil {
		return false, errors.New("entity is nil")
	}
	vote := entity.(*Vote)

	var contains bool
	for _, c := range p.committee {
		if c.GetAddress() == vote.Sender.GetAddress() {
			contains = true
			break
		}
	}
	if !contains {
		return false, errors.New("signature is not valid, unknown peer")
	}

	return true, nil
}

func IsValid(block *blockchain.Block) (bool, error) {
	if block == nil {
		return false, errors.New("entity is nil")
	}

	hash := blockchain.HashHeader(*block.Header())
	if block.Header().Hash() != hash {
		return false, errors.New("block hash is not valid")
	}

	//todo check other hashes here
	return true, nil
}

func (p *VoteValidator) Supported(mType pb.Message_MessageType) bool {
	return mType == pb.Message_VOTE
}
