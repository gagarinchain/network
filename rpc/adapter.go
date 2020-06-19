package rpc

import (
	"context"
	"github.com/davecgh/go-spew/spew"
	"github.com/emirpasic/gods/maps/treemap"
	"github.com/gagarinchain/common"
	"github.com/gagarinchain/common/api"
	pb "github.com/gagarinchain/common/protobuff"
	comm_rpc "github.com/gagarinchain/common/rpc"
	"github.com/gagarinchain/network/blockchain"
)

type OnBlockCommitAdapter struct {
	me     *common.Peer
	client *comm_rpc.OnBlockCommitClient
}

func (f *OnBlockCommitAdapter) OnBlockCommit(ctx context.Context, block api.Block, orphans *treemap.Map) error {
	var pbOrphans []*pb.BlockS
	if orphans != nil && len(orphans.Values()) > 0 {
		for _, o := range orphans.Values() {
			spew.Dump(o)
			if o == nil {
				continue
			}
			orphan := o.([]api.Block)
			for _, b := range orphan {
				if b != nil {
					pbOrphans = append(pbOrphans, b.ToStorageProto())
				}
			}
		}
	}
	_, err := f.client.Pbc().OnBlockCommit(ctx, &pb.OnBlockCommitRequest{
		Me:      f.me.ToStorageProto(),
		Block:   block.ToStorageProto(),
		Orphans: pbOrphans,
	})
	return err
}

type OnNextEpochAdapter struct {
	me     *common.Peer
	client *comm_rpc.OnNextEpochClient
}

func (f *OnNextEpochAdapter) OnNewEpoch(ctx context.Context, newEpoch int32) error {
	_, err := f.client.Pbc().OnNextEpoch(ctx, &pb.OnNextEpochRequest{
		Me:       f.me.ToStorageProto(),
		NewEpoch: newEpoch,
	})
	return err
}

type OnNextViewAdapter struct {
	me     *common.Peer
	client *comm_rpc.OnNextViewClient
}

func (f *OnNextViewAdapter) OnNewView(ctx context.Context, newView int32) error {
	_, err := f.client.Pbc().OnNextView(ctx, &pb.OnNextViewRequest{
		Me:      f.me.ToStorageProto(),
		NewView: newView,
	})
	return err
}

type OnNewBlockCreatedAdapter struct {
	me     *common.Peer
	client *comm_rpc.OnNewBlockCreatedClient
}

//todo remove builder and return block
func (f *OnNewBlockCreatedAdapter) OnNewBlockCreated(ctx context.Context, builder api.BlockBuilder, receipts []api.Receipt) (api.Block, error) {
	var pbr []*pb.Receipt
	for _, r := range receipts {
		pbr = append(pbr, r.ToStorageProto())
	}

	builded := builder.Build().ToStorageProto()
	resp, err := f.client.Pbc().OnNewBlockCreated(ctx, &pb.OnNewBlockCreatedRequest{
		Me:       f.me.ToStorageProto(),
		Block:    builded,
		Receipts: pbr,
	})
	if err != nil {
		return nil, err
	}

	storage := blockchain.CreateBlockFromStorage(resp.Block)
	spew.Dump(builded)
	spew.Dump(storage)
	return storage, nil
}

type OnVoteReceivedAdapter struct {
	me     *common.Peer
	client *comm_rpc.OnVoteReceivedClient
}

func (f *OnVoteReceivedAdapter) OnVoteReceived(ctx context.Context, vote api.Vote) error {
	_, err := f.client.Pbc().OnVoteReceived(ctx, &pb.OnVoteReceivedRequest{
		Me:   f.me.ToStorageProto(),
		Vote: vote.GetMessage(),
	})
	return err
}

func (f *OnVoteReceivedAdapter) OnQCFinished(ctx context.Context, qc api.QuorumCertificate) error {
	_, err := f.client.Pbc().OnQCFinished(ctx, &pb.OnQCFinishedRequest{
		Me: f.me.ToStorageProto(),
		Qc: qc.ToStorageProto(),
	})
	return err
}

type OnProposalAdapter struct {
	me     *common.Peer
	client *comm_rpc.OnProposalClient
}

func (f OnProposalAdapter) OnProposal(ctx context.Context, proposal api.Proposal) error {
	_, err := f.client.Pbc().OnProposal(ctx, &pb.OnProposalRequest{
		Me:       f.me.ToStorageProto(),
		Proposal: proposal.GetMessage(),
	})
	return err
}

type OnReceiveProposalAdapter struct {
	me     *common.Peer
	client *comm_rpc.OnReceiveProposalClient
}

func (f *OnReceiveProposalAdapter) BeforeProposedBlockAdded(ctx context.Context, proposal api.Proposal) error {
	_, err := f.client.Pbc().BeforeProposedBlockAdded(ctx, &pb.BeforeProposedBlockAddedRequest{
		Me:       f.me.ToStorageProto(),
		Proposal: proposal.GetMessage(),
	})
	return err
}

func (f *OnReceiveProposalAdapter) AfterProposedBlockAdded(ctx context.Context, proposal api.Proposal, receipts []api.Receipt) error {
	var pbr []*pb.Receipt
	for _, r := range receipts {
		pbr = append(pbr, r.ToStorageProto())
	}
	_, err := f.client.Pbc().AfterProposedBlockAdded(ctx, &pb.AfterProposedBlockAddedRequest{
		Me:       f.me.ToStorageProto(),
		Proposal: proposal.GetMessage(),
		Receipts: pbr,
	})
	return err
}

func (f *OnReceiveProposalAdapter) BeforeVoted(ctx context.Context, vote api.Vote) error {
	_, err := f.client.Pbc().BeforeVoted(ctx, &pb.BeforeVotedRequest{
		Me:   f.me.ToStorageProto(),
		Vote: vote.GetMessage(),
	})
	return err
}

func (f *OnReceiveProposalAdapter) AfterVoted(ctx context.Context, vote api.Vote) error {
	_, err := f.client.Pbc().AfterVoted(ctx, &pb.AfterVotedRequest{
		Me:   f.me.ToStorageProto(),
		Vote: vote.GetMessage(),
	})
	return err
}
