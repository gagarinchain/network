package rpc

import (
	"context"
	"github.com/emirpasic/gods/maps/treemap"
	"github.com/gagarinchain/common"
	"github.com/gagarinchain/common/api"
	rpc2 "github.com/gagarinchain/common/rpc"
)

//TODO think about organizing plugin chains, there are several moments to think about, e.g. how to transform input/output and plugin order
//Now we operate only with one plugin in chain
type PluginAdapter struct {
	orp  []api.OnReceiveProposal
	op   []api.OnProposal
	ovr  []api.OnVoteReceived
	onbc []api.OnNewBlockCreated
	onv  []api.OnNextView
	one  []api.OnNextEpoch
	obc  []api.OnBlockCommit
}

func (p *PluginAdapter) AddPlugin(rpc common.Plugin) {
	for _, i := range rpc.Interfaces {
		switch i {
		case "OnReceiveProposal":
			p.AddOnReceiveProposal(&OnReceiveProposalAdapter{client: rpc2.InitOnReceiveProposalClient(rpc.Address)})
		case "OnProposal":
			p.AddOnProposal(&OnProposalAdapter{client: rpc2.InitOnProposalClient(rpc.Address)})
		case "OnVoteReceived":
			p.AddOnVoteReceived(&OnVoteReceivedAdapter{client: rpc2.InitOnVoteReceivedClient(rpc.Address)})
		case "OnNewBlockCreated":
			p.AddOnNewBlockCreated(&OnNewBlockCreatedAdapter{client: rpc2.InitOnNewBlockCreatedClient(rpc.Address)})
		case "OnNextView":
			p.AddOnNextView(&OnNextViewAdapter{client: rpc2.InitOnNextViewClient(rpc.Address)})
		case "OnNextEpoch":
			p.AddOnNextEpoch(&OnNextEpochAdapter{client: rpc2.InitOnNextEpochClient(rpc.Address)})
		case "OnBlockCommit":
			p.AddOnBlockCommit(&OnBlockCommitAdapter{client: rpc2.InitOnBlockCommitClient(rpc.Address)})
		default:
			log.Errorf("Interface %v is not supported", i)
		}
	}
}

func (p *PluginAdapter) AddOnReceiveProposal(plugin api.OnReceiveProposal) {
	p.orp = append(p.orp, plugin)
}
func (p *PluginAdapter) AddOnProposal(plugin api.OnProposal) {
	p.op = append(p.op, plugin)
}
func (p *PluginAdapter) AddOnVoteReceived(plugin api.OnVoteReceived) {
	p.ovr = append(p.ovr, plugin)
}
func (p *PluginAdapter) AddOnNewBlockCreated(plugin api.OnNewBlockCreated) {
	p.onbc = append(p.onbc, plugin)
}
func (p *PluginAdapter) AddOnNextView(plugin api.OnNextView) {
	p.onv = append(p.onv, plugin)
}
func (p *PluginAdapter) AddOnNextEpoch(plugin api.OnNextEpoch) {
	p.one = append(p.one, plugin)
}
func (p *PluginAdapter) AddOnBlockCommit(plugin api.OnBlockCommit) {
	p.obc = append(p.obc, plugin)
}

func (p *PluginAdapter) OnBlockCommit(ctx context.Context, block api.Block, orphans *treemap.Map) error {
	l := len(p.obc)
	if l > 0 {
		return p.obc[l-1].OnBlockCommit(ctx, block, orphans)
	}
	return nil
}

func (p *PluginAdapter) OnNewEpoch(ctx context.Context, newEpoch int32) error {
	l := len(p.one)
	if l > 0 {
		return p.one[l-1].OnNewEpoch(ctx, newEpoch)
	}
	return nil
}

func (p *PluginAdapter) OnNewView(ctx context.Context, newView int32) error {
	l := len(p.onv)
	if l > 0 {
		return p.onv[l-1].OnNewView(ctx, newView)
	}
	return nil
}

func (p *PluginAdapter) OnNewBlockCreated(ctx context.Context, builder api.BlockBuilder, receipts []api.Receipt) (api.Block, error) {
	l := len(p.onbc)
	if l > 0 {
		return p.onbc[l-1].OnNewBlockCreated(ctx, builder, receipts)
	}
	return builder.Build(), nil
}

func (p *PluginAdapter) OnVoteReceived(ctx context.Context, vote api.Vote) error {
	l := len(p.ovr)
	if l > 0 {
		return p.ovr[l-1].OnVoteReceived(ctx, vote)
	}
	return nil
}

func (p *PluginAdapter) OnQCFinished(ctx context.Context, qc api.QuorumCertificate) error {
	l := len(p.ovr)
	if l > 0 {
		return p.ovr[l-1].OnQCFinished(ctx, qc)
	}
	return nil
}

func (p *PluginAdapter) OnProposal(ctx context.Context, proposal api.Proposal) error {
	l := len(p.op)
	if l > 0 {
		return p.op[l-1].OnProposal(ctx, proposal)
	}
	return nil
}

func (p *PluginAdapter) BeforeProposedBlockAdded(ctx context.Context, proposal api.Proposal) error {
	l := len(p.orp)
	if l > 0 {
		return p.orp[l-1].BeforeProposedBlockAdded(ctx, proposal)
	}
	return nil
}

func (p *PluginAdapter) AfterProposedBlockAdded(ctx context.Context, proposal api.Proposal, receipts []api.Receipt) error {
	l := len(p.orp)
	if l > 0 {
		return p.orp[l-1].AfterProposedBlockAdded(ctx, proposal, receipts)
	}
	return nil
}

func (p *PluginAdapter) BeforeVoted(ctx context.Context, vote api.Vote) error {
	l := len(p.orp)
	if l > 0 {
		return p.orp[l-1].BeforeVoted(ctx, vote)
	}
	return nil
}

func (p *PluginAdapter) AfterVoted(ctx context.Context, vote api.Vote) error {
	l := len(p.orp)
	if l > 0 {
		return p.orp[l-1].AfterVoted(ctx, vote)
	}
	return nil
}
