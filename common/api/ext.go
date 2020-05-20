package api

import (
	"github.com/emirpasic/gods/maps/treemap"
)

type OnReceiveProposal interface {
	BeforeProposedBlockAdded(pacer Pacer, bc Blockchain, proposal Proposal)
	AfterProposedBlockAdded(pacer Pacer, bc Blockchain, proposal Proposal)
	BeforeVoted(pacer Pacer, bc Blockchain, vote Vote)
	AfterVoted(pacer Pacer, bc Blockchain, vote Vote)
}

type OnProposal interface {
	CreateBlock(pacer Pacer, bc Blockchain) Block
}

type OnVoteReceived interface {
	OnVoteReceived(pacer Pacer, bc Blockchain, vote Vote)
	OnQCFinished(pacer Pacer, bc Blockchain, qc QuorumCertificate)
}

type OnNextView interface {
	OnNewView(pacer Pacer, bc Blockchain, newView int32)
}

type OnNextEpoch interface {
	OnNewEpoch(pacer Pacer, bc Blockchain, newEpoch int32)
}

type OnBlockCommit interface {
	OnBlockCommit(bc Blockchain, block Block, orphans *treemap.Map)
}

type NullOnReceiveProposal struct{}

func (NullOnReceiveProposal) BeforeProposedBlockAdded(pacer Pacer, bc Blockchain, proposal Proposal) {
}

func (NullOnReceiveProposal) AfterProposedBlockAdded(pacer Pacer, bc Blockchain, proposal Proposal) {
}

func (NullOnReceiveProposal) BeforeVoted(pacer Pacer, bc Blockchain, vote Vote) {
}

func (NullOnReceiveProposal) AfterVoted(pacer Pacer, bc Blockchain, vote Vote) {
}

type NullOnVoteReceived struct{}

func (NullOnVoteReceived) OnVoteReceived(pacer Pacer, bc Blockchain, vote Vote) {
}

func (NullOnVoteReceived) OnQCFinished(pacer Pacer, bc Blockchain, qc QuorumCertificate) {
}

type NullOnNextView struct{}

func (NullOnNextView) OnNewView(pacer Pacer, bc Blockchain, newView int32) {
}

type NullOnNextEpoch struct{}

func (NullOnNextEpoch) OnNewEpoch(pacer Pacer, bc Blockchain, newEpoch int32) {
}

type NullOnBlockCommit struct{}

func (NullOnBlockCommit) OnBlockCommit(bc Blockchain, block Block, orphans *treemap.Map) {
}

type NullOnProposal struct{}

func (NullOnProposal) CreateBlock(pacer Pacer, bc Blockchain) Block {
	return nil
}
