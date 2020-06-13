package rpc

import (
	"context"
	"github.com/davecgh/go-spew/spew"
	"github.com/emirpasic/gods/maps/treemap"
	"github.com/gagarinchain/common/api"
	pb "github.com/gagarinchain/common/protobuff"
	comm_rpc "github.com/gagarinchain/common/rpc"
	"github.com/gagarinchain/network/blockchain"
)

type OnBlockCommitAdapter struct {
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
		Block:   block.ToStorageProto(),
		Orphans: pbOrphans,
	})
	return err
}

type OnNextEpochAdapter struct {
	client *comm_rpc.OnNextEpochClient
}

func (f *OnNextEpochAdapter) OnNewEpoch(ctx context.Context, newEpoch int32) error {
	_, err := f.client.Pbc().OnNextEpoch(ctx, &pb.OnNextEpochRequest{
		NewEpoch: newEpoch,
	})
	return err
}

type OnNextViewAdapter struct {
	client *comm_rpc.OnNextViewClient
}

func (f *OnNextViewAdapter) OnNewView(ctx context.Context, newView int32) error {
	_, err := f.client.Pbc().OnNextView(ctx, &pb.OnNextViewRequest{
		NewView: newView,
	})
	return err
}

type OnNewBlockCreatedAdapter struct {
	client *comm_rpc.OnNewBlockCreatedClient
}

//todo remove builder and return block
func (f *OnNewBlockCreatedAdapter) OnNewBlockCreated(ctx context.Context, builder api.BlockBuilder, receipts []api.Receipt) (api.Block, error) {
	var pbr []*pb.Receipt
	for _, r := range receipts {
		pbr = append(pbr, r.ToStorageProto())
	}
	resp, err := f.client.Pbc().OnNewBlockCreated(ctx, &pb.OnNewBlockCreatedRequest{
		Block:    builder.Build().ToStorageProto(),
		Receipts: nil,
	})
	if err != nil {
		return nil, err
	}

	storage := blockchain.CreateBlockFromStorage(resp.Block)
	return storage, nil
}

type OnVoteReceivedAdapter struct {
	client *comm_rpc.OnVoteReceivedClient
}

func (f *OnVoteReceivedAdapter) OnVoteReceived(ctx context.Context, vote api.Vote) error {
	_, err := f.client.Pbc().OnVoteReceived(ctx, &pb.OnVoteReceivedRequest{
		Vote: vote.GetMessage(),
	})
	return err
}

func (f *OnVoteReceivedAdapter) OnQCFinished(ctx context.Context, qc api.QuorumCertificate) error {
	_, err := f.client.Pbc().OnQCFinished(ctx, &pb.OnQCFinishedRequest{
		Qc: qc.ToStorageProto(),
	})
	return err
}

type OnProposalAdapter struct {
	client *comm_rpc.OnProposalClient
}

func (f OnProposalAdapter) OnProposal(ctx context.Context, proposal api.Proposal) error {
	_, err := f.client.Pbc().OnProposal(ctx, &pb.OnProposalRequest{
		Proposal: proposal.GetMessage(),
	})
	return err
}

type OnReceiveProposalAdapter struct {
	client *comm_rpc.OnReceiveProposalClient
}

func (f *OnReceiveProposalAdapter) BeforeProposedBlockAdded(ctx context.Context, proposal api.Proposal) error {
	_, err := f.client.Pbc().BeforeProposedBlockAdded(ctx, &pb.BeforeProposedBlockAddedRequest{
		Proposal: proposal.GetMessage(),
	})
	return err
}

func (f *OnReceiveProposalAdapter) AfterProposedBlockAdded(ctx context.Context, proposal api.Proposal, receipts []api.Receipt) error {
	_, err := f.client.Pbc().AfterProposedBlockAdded(ctx, &pb.AfterProposedBlockAddedRequest{
		Proposal: proposal.GetMessage(),
	})
	return err
}

func (f *OnReceiveProposalAdapter) BeforeVoted(ctx context.Context, vote api.Vote) error {
	_, err := f.client.Pbc().BeforeVoted(ctx, &pb.BeforeVotedRequest{
		Vote: vote.GetMessage(),
	})
	return err
}

func (f *OnReceiveProposalAdapter) AfterVoted(ctx context.Context, vote api.Vote) error {
	_, err := f.client.Pbc().AfterVoted(ctx, &pb.AfterVotedRequest{
		Vote: vote.GetMessage(),
	})
	return err
}
