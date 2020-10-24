package rpc

import (
	"context"
	"github.com/emirpasic/gods/maps/treemap"
	"github.com/gagarinchain/common"
	"github.com/gagarinchain/common/api"
	rpc2 "github.com/gagarinchain/common/rpc"
	"github.com/gagarinchain/network/blockchain"
)

//TODO think about organizing plugin chains, there are several moments to think about, e.g. how to transform input/output and plugin order
//Now we operate only with one plugin in chain
type PluginAdapter struct {
	me   *common.Peer
	orp  []api.OnReceiveProposal
	op   []api.OnProposal
	ovr  []api.OnVoteReceived
	onbc []api.OnNewBlockCreated
	onv  []api.OnNextView
	one  []api.OnNextEpoch
	obc  []api.OnBlockCommit
}

func NewPluginAdapter(me *common.Peer) *PluginAdapter {
	return &PluginAdapter{me: me}
}

func (p *PluginAdapter) AddPlugin(rpc common.Plugin) {
	for _, i := range rpc.Interfaces {
		switch i {
		case "OnReceiveProposal":
			p.AddOnReceiveProposal(&OnReceiveProposalAdapter{me: p.me, client: rpc2.InitOnReceiveProposalClient(rpc.Address)})
		case "OnProposal":
			p.AddOnProposal(&OnProposalAdapter{me: p.me, client: rpc2.InitOnProposalClient(rpc.Address)})
		case "OnVoteReceived":
			p.AddOnVoteReceived(&OnVoteReceivedAdapter{me: p.me, client: rpc2.InitOnVoteReceivedClient(rpc.Address)})
		case "OnNewBlockCreated":
			p.AddOnNewBlockCreated(&OnNewBlockCreatedAdapter{me: p.me, client: rpc2.InitOnNewBlockCreatedClient(rpc.Address)})
		case "OnNextView":
			p.AddOnNextView(&OnNextViewAdapter{me: p.me, client: rpc2.InitOnNextViewClient(rpc.Address)})
		case "OnNextEpoch":
			p.AddOnNextEpoch(&OnNextEpochAdapter{me: p.me, client: rpc2.InitOnNextEpochClient(rpc.Address)})
		case "OnBlockCommit":
			p.AddOnBlockCommit(&OnBlockCommitAdapter{me: p.me, client: rpc2.InitOnBlockCommitClient(rpc.Address)})
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
	var errs []error
	for _, obc := range p.obc {
		if err := obc.OnBlockCommit(ctx, block, orphans); err != nil {
			log.Error(err)
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs[0]
	} else {
		return nil
	}
}

func (p *PluginAdapter) OnNewEpoch(ctx context.Context, newEpoch int32) error {
	var errs []error
	for _, one := range p.one {
		if err := one.OnNewEpoch(ctx, newEpoch); err != nil {
			log.Error(err)
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs[0]
	} else {
		return nil
	}
}

func (p *PluginAdapter) OnNewView(ctx context.Context, newView int32) error {
	var errs []error
	for _, onv := range p.onv {
		if err := onv.OnNewView(ctx, newView); err != nil {
			log.Error(err)
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs[0]
	} else {
		return nil
	}
}

func (p *PluginAdapter) OnNewBlockCreated(ctx context.Context, builder api.BlockBuilder, receipts []api.Receipt) (api.Block, error) {
	var errs []error
	var block api.Block
	for _, onbc := range p.onbc {
		var err error
		block, err = onbc.OnNewBlockCreated(ctx, builder, receipts)
		if err != nil {
			log.Error(err)
			errs = append(errs, err)
		} else {
			builder = blockchain.NewBlockBuilderFromBlock(block)
		}
	}

	if len(errs) > 0 {
		return nil, errs[0]
	} else {
		return block, nil
	}

}

func (p *PluginAdapter) OnVoteReceived(ctx context.Context, vote api.Vote) error {
	var errs []error
	for _, ovr := range p.ovr {
		if err := ovr.OnVoteReceived(ctx, vote); err != nil {
			log.Error(err)
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs[0]
	} else {
		return nil
	}
}

func (p *PluginAdapter) OnQCFinished(ctx context.Context, qc api.QuorumCertificate) error {
	var errs []error
	for _, ovr := range p.ovr {
		if err := ovr.OnQCFinished(ctx, qc); err != nil {
			log.Error(err)
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs[0]
	} else {
		return nil
	}
}

func (p *PluginAdapter) OnProposal(ctx context.Context, proposal api.Proposal) error {
	var errs []error
	for _, op := range p.op {
		if err := op.OnProposal(ctx, proposal); err != nil {
			log.Error(err)
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs[0]
	} else {
		return nil
	}
}

func (p *PluginAdapter) BeforeProposedBlockAdded(ctx context.Context, proposal api.Proposal) error {
	var errs []error
	for _, orp := range p.orp {
		if err := orp.BeforeProposedBlockAdded(ctx, proposal); err != nil {
			log.Error(err)
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs[0]
	} else {
		return nil
	}
}

func (p *PluginAdapter) AfterProposedBlockAdded(ctx context.Context, proposal api.Proposal, receipts []api.Receipt) error {
	var errs []error
	for _, orp := range p.orp {
		if err := orp.AfterProposedBlockAdded(ctx, proposal, receipts); err != nil {
			log.Error(err)
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs[0]
	} else {
		return nil
	}
}

func (p *PluginAdapter) BeforeVoted(ctx context.Context, vote api.Vote) error {
	var errs []error
	for _, orp := range p.orp {
		if err := orp.BeforeVoted(ctx, vote); err != nil {
			log.Error(err)
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs[0]
	} else {
		return nil
	}
}

func (p *PluginAdapter) AfterVoted(ctx context.Context, vote api.Vote) error {
	var errs []error
	for _, orp := range p.orp {
		if err := orp.AfterVoted(ctx, vote); err != nil {
			log.Error(err)
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs[0]
	} else {
		return nil
	}
}
