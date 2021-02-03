package test

import (
	"bytes"
	"context"
	"github.com/gagarinchain/common"
	"github.com/gagarinchain/common/api"
	common2 "github.com/gagarinchain/common/eth/common"
	"github.com/gagarinchain/common/eth/crypto"
	msg "github.com/gagarinchain/common/message"
	"github.com/gagarinchain/common/protobuff"
	tx2 "github.com/gagarinchain/common/tx"
	"github.com/gagarinchain/network/blockchain"
	"github.com/gagarinchain/network/blockchain/state"
	"github.com/gagarinchain/network/blockchain/tx"
	"github.com/gagarinchain/network/hotstuff"
	"github.com/gagarinchain/network/mocks"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"math/big"
	"testing"
	"time"
)

// Scenario 1a: Simple test on optimistic responsiveness
// Start as Replica next Proposer
// Get proposal
// Collect 2f + 1 votes
// Propose block with new QC
func TestScenario1a(t *testing.T) {
	ctx := initContext(t)
	timeout, f := context.WithTimeout(context.Background(), 10*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hotstuffChan)
	defer f()

	ctx.StartFirstView(timeout)
	ctx.setMe(2)

	p, m := ctx.bfactory.CreateProposalForParent(1, ctx.bc.GetHead(), ctx.bc.GetGenesisCert(), []byte("wonderful block"))

	ctx.hotstuffChan <- m

	votes := ctx.makeVotes(2*ctx.cfg.F/3+1, p.NewBlock())
	for _, v := range votes {
		ctx.hotstuffChan <- v
	}

	proposal := <-ctx.proposalChan

	payload := &pb.ProposalPayload{}
	if err := ptypes.UnmarshalAny(proposal.Payload, payload); err != nil {
		log.Error(err)
	}

	assert.Equal(t, int32(2), payload.Block.GetHeader().GetHeight())
	assert.Equal(t, int32(1), payload.Block.GetCert().GetHeader().GetHeight())

}

// Scenario 1b: Send some votes before proposal, due to network reordering
// Start as Replica, Next proposer
// Collect not enough votes,
// Receive proposal,
// Collect the rest of votes
// Propose block with new QC
func TestScenario1b(t *testing.T) {
	ctx := initContext(t)
	timeout, f := context.WithTimeout(context.Background(), 10*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hotstuffChan)
	defer f()

	ctx.StartFirstView(timeout)
	ctx.setMe(2)

	p, m := ctx.bfactory.CreateProposalForParent(1, ctx.bc.GetHead(), ctx.bc.GetGenesisCert(), []byte("wonderful block"))

	votes := ctx.makeVotes(2*ctx.cfg.F/3+1, p.NewBlock())
	for _, v := range votes[:2*ctx.cfg.F/3-2] {
		ctx.hotstuffChan <- v
	}

	ctx.hotstuffChan <- m

	for _, v := range votes[2*ctx.cfg.F/3-2:] {
		ctx.hotstuffChan <- v
	}

	proposal := <-ctx.proposalChan

	payload := &pb.ProposalPayload{}
	if err := ptypes.UnmarshalAny(proposal.Payload, payload); err != nil {
		log.Error(err)
	}

	assert.Equal(t, int32(2), payload.Block.GetHeader().GetHeight())
	assert.Equal(t, int32(1), payload.Block.GetCert().GetHeader().GetHeight())

}

// Scenario 1c: Receive proposal after we collected votes
// Start as Replica, Next proposer
// Collect 2*f + 1 votes,
// Receive proposal,
// Propose block with new QC
func TestScenario1c(t *testing.T) {
	ctx := initContext(t)

	timeout, f := context.WithTimeout(context.Background(), 10*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hotstuffChan)
	defer f()

	defer f()
	//defer ctx.protocol.Stop()

	ctx.StartFirstView(timeout)
	ctx.setMe(2)

	p, m := ctx.bfactory.CreateProposalForParent(1, ctx.bc.GetHead(), ctx.bc.GetGenesisCert(), []byte("wonderful block"))

	votes := ctx.makeVotes(2*ctx.cfg.F/3+1, p.NewBlock())
	for _, v := range votes {
		ctx.hotstuffChan <- v
	}

	go func() {
		ctx.SendBlocks([]api.Block{p.NewBlock()})
	}()

	ctx.hotstuffChan <- m

	proposal := <-ctx.proposalChan

	payload := &pb.ProposalPayload{}
	if err := ptypes.UnmarshalAny(proposal.Payload, payload); err != nil {
		log.Error(err)
	}

	ctx.waitRounds(timeout, 2)

	assert.Equal(t, int32(2), payload.Block.GetHeader().GetHeight())
	assert.Equal(t, int32(1), payload.Block.GetCert().GetHeader().GetHeight())
}

// Scenario 1d: Do not vote for proposal that do not extend PREF block
// Start as Replica, Next proposer
// Collect 2*f + 1 votes,
// Receive proposal,
// Propose block with new QC
func TestScenario1d(t *testing.T) {
	ctx := initContext(t)

	timeout, f := context.WithTimeout(context.Background(), 10*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hotstuffChan)
	defer f()

	ctx.StartFirstView(timeout)
	ctx.setMe(6)

	b1 := ctx.bfactory.CreateBlock(ctx.bc.GetGenesisBlock(), ctx.bc.GetGenesisCert(), []byte(""))
	b2 := ctx.bfactory.CreateBlock(b1, ctx.bc.GetGenesisCert(), []byte(""))
	b21 := ctx.bfactory.CreateBlock(b1, ctx.bc.GetGenesisCert(), []byte("ыфвфыв"))
	qcBlock1 := ctx.bfactory.CreateQC(b2)
	b3 := ctx.bfactory.CreateBlock(b21, qcBlock1, []byte(""))
	_, m2 := ctx.bfactory.CreateProposalForBlock(3, b3, qcBlock1)
	ctx.hotstuffChan <- m2
	ctx.SendBlocks([]api.Block{b3, b2, b21, b1})

	ctx.waitTimeout(timeout)

	assert.Equal(t, int32(2), ctx.protocol.HC().Height())
}

//Scenario 2a: Synchronize when no proposal is received and form synchronize QC
//Start as Replica
//Next proposer don't propose
//Synchronize on view 1
//Receive (n - f) messages eventually
//Move to view 2 with SynchronizeQC
func TestScenario2a(t *testing.T) {
	ctx := initContext(t)

	timeout, f := context.WithTimeout(context.Background(), 10*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hotstuffChan)
	defer f()

	ctx.StartFirstView(timeout)
	ctx.setMe(6)

	ctx.waitTimeout(timeout)
	ctx.sendMoreSyncMessages(0, 2*ctx.cfg.F/3+1, 1, ctx.bc.GetGenesisCert(), false)

	ctx.waitRounds(timeout, 1)

	assert.Equal(t, int32(1), ctx.protocol.HC().Height())
	assert.Equal(t, api.SC, ctx.protocol.HC().Type())
	assert.Equal(t, int32(2), ctx.pacer.GetCurrentView())
}

//Scenario 2b: Interrupt synchronization with new proposal
//Start as Replica
//Receive proposal 1, vote
//Next proposer don't propose
//Synchronize on view 1
//Receive proposal with HC 3 during synchronize phase
//Move to view 4 with HC
func TestScenario2b(t *testing.T) {
	ctx := initContext(t)

	timeout, f := context.WithTimeout(context.Background(), 10*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hotstuffChan)
	defer f()

	ctx.StartFirstView(timeout)
	ctx.setMe(6)

	ctx.waitTimeout(timeout)
	ctx.sendMoreSyncMessages(0, 2*ctx.cfg.F/3+1, 1, ctx.bc.GetGenesisCert(), false)

	ctx.waitRounds(timeout, 1)

	blocks := ctx.bfactory.CreateChainNoGaps(4)
	block1 := blocks[len(blocks)-1]
	_, m := ctx.bfactory.CreateProposalForBlock(4, block1, block1.QC())
	ctx.hotstuffChan <- m
	ctx.SendBlocks(blocks)
	ctx.waitRounds(timeout, 1)

	assert.Equal(t, block1.QC().Height(), ctx.protocol.HC().Height())
	assert.Equal(t, api.QRef, ctx.protocol.HC().Type())
	assert.Equal(t, block1.QC().Height()+1, ctx.pacer.GetCurrentView())
}

//Scenario 2c: Synchronize with votes on current height
//Start as Replica
//Receive proposal 1, vote
//Synchronize on view 1
//Send Synchronize message without votingSignature
func TestScenario2c(t *testing.T) {
	ctx := initContext(t)

	timeout, f := context.WithTimeout(context.Background(), 10*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hotstuffChan)
	defer f()

	sync1 := ctx.StartFirstView(timeout)
	ctx.setMe(6)

	_, m := ctx.bfactory.CreateProposalForParent(1, ctx.bc.GetHead(), ctx.bc.GetGenesisCert(), []byte("wonderful block"))
	ctx.hotstuffChan <- m

	ctx.waitTimeout(timeout)

	s2 := <-ctx.syncChan
	sync2, _ := hotstuff.CreateSyncFromMessage(s2)

	assert.NotNil(t, sync1.VotingSignature())
	assert.NotNil(t, sync1.Signature())
	assert.Equal(t, int32(0), sync1.Height())

	assert.Nil(t, sync2.VotingSignature())
	assert.NotNil(t, sync2.Signature())
	assert.Equal(t, int32(1), sync2.Height())
}

//Scenario 2d: Make QC during synchronize
//Start as Replica, next proposer
//Receive some votes
//Synchronize
//Receive some sync messages
//Receive the rest of votes, make QC, progress to the next view
//Receive the rest of syncs do nothing
func TestScenario2d(t *testing.T) {
	ctx := initContext(t)
	timeout, f := context.WithTimeout(context.Background(), 10*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hotstuffChan)
	defer f()

	ctx.StartFirstView(timeout)
	ctx.setMe(2)

	p, _ := ctx.bfactory.CreateProposalForParent(1, ctx.bc.GetHead(), ctx.bc.GetGenesisCert(), []byte("wonderful block"))

	votes := ctx.makeVotes(2*ctx.cfg.F/3+1, p.NewBlock())
	ctx.SendBlocks([]api.Block{p.NewBlock()})

	for i, v := range votes {
		if i < ctx.cfg.F+1 {
			ctx.hotstuffChan <- v
			i++
		}
	}

	ctx.waitTimeout(timeout)
	ctx.sendMoreSyncMessages(0, ctx.cfg.F/3+1, 1, ctx.bc.GetGenesisCert(), false)

	for i, v := range votes {
		if i > ctx.cfg.F {
			ctx.hotstuffChan <- v
		}
	}

	//ctx.waitRounds(timeout, 1)
	ctx.sendMoreSyncMessages(ctx.cfg.F/3+1, 2*ctx.cfg.F/3+1, 1, ctx.bc.GetGenesisCert(), false)

	proposal := <-ctx.proposalChan

	payload := &pb.ProposalPayload{}
	if err := ptypes.UnmarshalAny(proposal.Payload, payload); err != nil {
		log.Error(err)
	}

	assert.Equal(t, int32(2), payload.Block.GetHeader().Height)

}

//Scenario 2e: Make QC during synchronize
//Start as Replica
//Receive some votes
//Synchronize
//Receive some sync messages
//Receive the rest of votes, make QC, progress to the next view
//TODO should resend new QC to other replicas probably via resynchronization, or resending it to next proposer
func TestScenario2e(t *testing.T) {
	ctx := initContext(t)
	timeout, f := context.WithTimeout(context.Background(), 10*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hotstuffChan)
	defer f()

	ctx.StartFirstView(timeout)
	ctx.setMe(2)

	p, _ := ctx.bfactory.CreateProposalForParent(1, ctx.bc.GetHead(), ctx.bc.GetGenesisCert(), []byte("wonderful block"))

	votes := ctx.makeVotes(2*ctx.cfg.F/3+1, p.NewBlock())
	ctx.SendBlocks([]api.Block{p.NewBlock()})

	for i, v := range votes {
		if i < ctx.cfg.F+1 {
			ctx.hotstuffChan <- v
			i++
		}
	}

	ctx.waitTimeout(timeout)
	ctx.sendMoreSyncMessages(0, ctx.cfg.F/3+1, 1, ctx.bc.GetGenesisCert(), false)

	for i, v := range votes {
		if i > ctx.cfg.F {
			ctx.hotstuffChan <- v
		}
	}

	ctx.waitRounds(timeout, 1)

}

//Scenario 3a: Check if we really vote
//Start as a Replica
//Get proposal
//Vote
func TestScenario3a(t *testing.T) {
	ctx := initContext(t)
	timeout, f := context.WithTimeout(context.Background(), 10*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hotstuffChan)
	defer f()

	ctx.StartFirstView(timeout)
	ctx.setMe(0)

	newBlock := ctx.bfactory.CreateBlock(ctx.bc.GetHead(), ctx.bc.GetGenesisCert(), []byte("wonderful block"))
	p := ctx.CreateProposal(newBlock, 1)
	ctx.hotstuffChan <- p

	vote := <-ctx.voteChan

	assert.Equal(t, pb.Message_VOTE, vote.Type)

}

//Scenario 3b: Do not vote twice on one height
//Start new epoch
//Vote for block
//Synchronize without voting
//Receive synchronization messages
//Don't vote again
func TestScenario3b(t *testing.T) {
	ctx := initContext(t)

	timeout, f := context.WithTimeout(context.Background(), 10*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hotstuffChan)
	defer f()

	ctx.StartFirstView(timeout)
	ctx.setMe(0)

	p, m := ctx.bfactory.CreateProposalForParent(1, ctx.bc.GetHead(), ctx.bc.GetGenesisCert(), []byte("wonderful block"))
	ctx.hotstuffChan <- m
	ctx.SendBlocks([]api.Block{p.NewBlock()})

	<-ctx.voteChan

	select {
	case <-ctx.voteChan:
		t.Error("Received second vote on current height")
	case <-ctx.syncChan:
		ctx.sendMoreSyncMessages(0, 2*ctx.cfg.F/3+1, 1, ctx.bc.GetGenesisCert(), false)
	}
	ctx.waitRounds(timeout, 2)

}

//Scenario 3c: Do not vote when timeout
//Start new epoch
//Proposer
//Synchronize on r=1 and create TC
//Receive proposal
//Do not vote for it
func TestScenario3c(t *testing.T) {
	ctx := initContext(t)

	timeout, f := context.WithTimeout(context.Background(), 10*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hotstuffChan)
	defer f()

	ctx.StartFirstView(timeout)
	ctx.setMe(6)

	ctx.waitTimeout(timeout)

	_, m := ctx.bfactory.CreateProposalForParent(1, ctx.bc.GetGenesisBlock(), ctx.bc.GetGenesisCert(), []byte("wonderful block"))
	ctx.hotstuffChan <- m

	select {
	case <-ctx.voteChan:
		t.Error("Voted after timeout on the same height")
	case s := <-ctx.syncChan:
		sync, _ := hotstuff.CreateSyncFromMessage(s)
		assert.True(t, sync.Voting())
		ctx.sendMoreSyncMessages(0, 2*ctx.cfg.F/3+1, 1, ctx.bc.GetGenesisCert(), false)
	}
	ctx.waitRounds(timeout, 1)
}

//Scenario 3d: Next sync after SC made do not trigger cert recreation
//Start new epoch
//Proposer
//Propose block with previous SC after Delta
func TestScenario3d(t *testing.T) {
	ctx := initContext(t)

	timeout, f := context.WithTimeout(context.Background(), 10*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hotstuffChan)
	defer f()

	ctx.StartFirstView(timeout)
	ctx.setMe(6)

	ctx.waitTimeout(timeout)

	ctx.sendMoreSyncMessages(0, 2*ctx.cfg.F/3+1, 1, ctx.bc.GetGenesisCert(), false)
	hc1 := ctx.protocol.HC()
	ctx.sendMoreSyncMessages(2*ctx.cfg.F/3+1, 4, 1, ctx.bc.GetGenesisCert(), false)
	hc2 := ctx.protocol.HC()

	ctx.waitTimeout(timeout)

	assert.Equal(t, hc1, hc2)
}

//Scenario 3e: Fail to synchronize, then synchronize successfully and propose
//Start new epoch
//Proposer
//Propose block with previous SC after Delta
func TestScenario3e(t *testing.T) {
	ctx := initContext(t)

	timeout, f := context.WithTimeout(context.Background(), 10*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hotstuffChan)
	defer f()

	ctx.StartFirstView(timeout)
	ctx.setMe(6)

	ctx.waitTimeout(timeout)

	ctx.waitTimeout(timeout)
	ctx.sendMoreSyncMessages(0, 2*ctx.cfg.F/3+1, 1, ctx.bc.GetGenesisCert(), false)

	ctx.waitRounds(timeout, 1)
	hc1 := ctx.protocol.HC()

	assert.Equal(t, int32(1), hc1.Height())
}

//Scenario 4a: Propose with synchronize QC
//Start new epoch
//Next Proposer
//Proposer equivocates, no proposal sent
//Propose block with previous QC (2) after 2*Delta
func TestScenario4a(t *testing.T) {
	ctx := initContext(t)
	timeout, f := context.WithTimeout(context.Background(), 10*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hotstuffChan)
	defer f()

	ctx.StartFirstView(timeout)
	ctx.setMe(2)

	ctx.waitTimeout(timeout)
	ctx.sendMoreSyncMessages(0, 2*ctx.cfg.F/3+1, 1, ctx.bc.GetGenesisCert(), true)

	proposal := <-ctx.proposalChan

	p, _ := hotstuff.CreateProposalFromMessage(proposal)

	assert.Equal(t, int32(2), p.NewBlock().Header().Height())
	assert.Equal(t, int32(1), p.NewBlock().QC().Height())
	var committee []*crypto.PublicKey
	for _, peer := range ctx.peers {
		committee = append(committee, peer.PublicKey())
	}
	valid, e := p.NewBlock().QC().IsValid(p.NewBlock().Header().QCHash(), committee)
	assert.Nil(t, e)
	assert.True(t, valid)
}

//Scenario 6a:
//Start new epoch
//Replica
//Receive proposal fork
//Get no block on arbitrary height
//Reject proposal and all fork
func TestScenario6a(t *testing.T) {
	ctx := initContext(t)

	timeout, f := context.WithTimeout(context.Background(), 10*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hotstuffChan)
	defer f()

	ctx.StartFirstView(timeout)
	ctx.setMe(8)

	p1, m1 := ctx.bfactory.CreateProposalForParent(1, ctx.bc.GetGenesisBlock(), ctx.bc.GetGenesisCert(), []byte("block 1"))
	qc1 := ctx.bfactory.CreateQC(p1.NewBlock())
	p2, m2 := ctx.bfactory.CreateProposalForParent(2, p1.NewBlock(), qc1, []byte("block 2"))
	qc2 := ctx.bfactory.CreateQC(p2.NewBlock())
	_, m3 := ctx.bfactory.CreateProposalForParent(3, p2.NewBlock(), qc2, []byte("block 3"))
	block21 := ctx.bfactory.CreateBlock(p1.NewBlock(), qc1, []byte("block 21"))
	block31 := ctx.bfactory.CreateBlock(block21, qc2, []byte("block 31"))
	qc3 := ctx.bfactory.CreateQC(block31)
	p4, m4 := ctx.bfactory.CreateProposalForParent(4, block31, qc3, []byte("block 41"))

	ctx.SendBlocks([]api.Block{p4.NewBlock(), block31, p2.NewBlock()})

	ctx.hotstuffChan <- m1
	ctx.hotstuffChan <- m2
	ctx.hotstuffChan <- m3
	ctx.hotstuffChan <- m4

	ctx.waitTimeout(timeout)

	assert.Equal(t, ctx.bc.GetHead().Header().Height(), int32(3))
}

//Scenario 6b:
//Start new epoch
//Replica
//Receive fork Gc<-B1c<-B2<-B3<-B4
//Receive proposal fork B1c-B22-B32-B42-B52 (reject cause don't extend QREF())
func TestScenario6b(t *testing.T) {
	ctx := initContext(t)

	timeout, f := context.WithTimeout(context.Background(), 10*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hotstuffChan)
	defer f()

	ctx.StartFirstView(timeout)
	ctx.setMe(8)

	p1, m1 := ctx.bfactory.CreateProposalForParent(1, ctx.bc.GetGenesisBlock(), ctx.bc.GetGenesisCert(), []byte("block 1"))
	qcb1 := ctx.bfactory.CreateQC(p1.NewBlock())
	p2, m2 := ctx.bfactory.CreateProposalForParent(2, p1.NewBlock(), qcb1, []byte("block 2"))
	p3, m3 := ctx.bfactory.CreateProposalForParent(3, p2.NewBlock(), ctx.bfactory.CreateQC(p2.NewBlock()), []byte("block 3"))
	_, m4 := ctx.bfactory.CreateProposalForParent(4, p3.NewBlock(), ctx.bfactory.CreateQC(p3.NewBlock()), []byte("block 4"))

	block22 := ctx.bfactory.CreateBlock(p1.NewBlock(), qcb1, []byte("block 22"))
	block32 := ctx.bfactory.CreateBlock(block22, qcb1, []byte("block 32"))
	block42 := ctx.bfactory.CreateBlock(block32, qcb1, []byte("block 42"))
	pblock52 := ctx.bfactory.CreateBlock(block42, qcb1, []byte("block 52"))

	_, m52 := ctx.bfactory.CreateProposalForBlock(5, pblock52, ctx.bfactory.CreateSC(4))

	ctx.SendBlocks([]api.Block{p1.NewBlock(), p2.NewBlock(), p3.NewBlock(), block22, block32, block42})

	ctx.hotstuffChan <- m1
	ctx.hotstuffChan <- m2
	ctx.hotstuffChan <- m3
	ctx.hotstuffChan <- m4
	ctx.hotstuffChan <- m52

	ctx.waitTimeout(timeout)

	assert.Equal(t, int32(4), ctx.protocol.Vheight())
	assert.Equal(t, p1.NewBlock().Header(), ctx.bc.GetTopCommittedBlock().Header())
	assert.Nil(t, ctx.bc.GetBlockByHash(pblock52.Header().Hash()))
}

//Scenario 6c:
//Start new epoch
//Replica
//Receive fork Gc<-B1c<-B2<-B3<-B4
//Receive proposal fork B2<-B32<-(B2)B42-B52 (vote for b52 as it extends QREF)
func TestScenario6c(t *testing.T) {
	ctx := initContext(t)

	timeout, f := context.WithTimeout(context.Background(), 10*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hotstuffChan)
	defer f()

	ctx.StartFirstView(timeout)
	ctx.setMe(8)

	p1, m1 := ctx.bfactory.CreateProposalForParent(1, ctx.bc.GetGenesisBlock(), ctx.bc.GetGenesisCert(), []byte("block 1"))
	qcb1 := ctx.createQC(p1.NewBlock())
	p2, m2 := ctx.bfactory.CreateProposalForParent(2, p1.NewBlock(), qcb1, []byte("block 2"))
	qcb2 := ctx.createQC(p2.NewBlock())
	p3, m3 := ctx.bfactory.CreateProposalForParent(3, p2.NewBlock(), qcb2, []byte("block 3"))
	_, m4 := ctx.bfactory.CreateProposalForParent(4, p3.NewBlock(), ctx.createQC(p3.NewBlock()), []byte("block 4"))
	block32 := ctx.bfactory.CreateBlock(p2.NewBlock(), qcb2, []byte("block 32"))
	block42 := ctx.bfactory.CreateBlock(block32, qcb2, []byte("block 42"))
	qc4 := ctx.bfactory.CreateEmptyQC(block42)
	p52, m52 := ctx.bfactory.CreateProposalForParent(5, block42, qc4, []byte("block 52"))

	ctx.SendBlocks([]api.Block{p2.NewBlock(), block32, block42})
	ctx.hotstuffChan <- m1
	ctx.hotstuffChan <- m2
	ctx.hotstuffChan <- m3
	ctx.hotstuffChan <- m4
	ctx.hotstuffChan <- m52

	ctx.waitTimeout(timeout)

	assert.Equal(t, int32(5), ctx.protocol.Vheight())
	assert.Equal(t, p1.NewBlock().Header(), ctx.bc.GetTopCommittedBlock().Header())
	assert.Equal(t, p52.NewBlock().Header(), ctx.bc.GetBlockByHeight(5)[0].Header())
}

//Scenario 6d:
//Start new epoch
//Replica
//Receive fork Gc<-B1c<-B2<-B3<-B4
//Receive fork B2<-B32<-(B2)B42-B52 (vote for B52 as it extends QREF)
//Receive fork <-(B4)B51-(B4)B6 (vote for B6 as it extends QREF)
func TestScenario6d(t *testing.T) {
	ctx := initContext(t)

	timeout, f := context.WithTimeout(context.Background(), 10*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hotstuffChan)
	defer f()

	ctx.StartFirstView(timeout)
	ctx.setMe(8)

	block1 := ctx.bfactory.CreateBlock(ctx.bc.GetGenesisBlock(), ctx.bc.GetGenesisCert(), []byte("block 1"))
	qcb1 := ctx.bfactory.CreateQC(block1)
	block2 := ctx.bfactory.CreateBlock(block1, qcb1, []byte("block 2"))
	qcb2 := ctx.bfactory.CreateQC(block2)
	block3 := ctx.bfactory.CreateBlock(block2, qcb2, []byte("block 3"))
	block4 := ctx.bfactory.CreateBlock(block3, ctx.createQC(block3), []byte("block 4"))
	block32 := ctx.bfactory.CreateBlock(block2, qcb2, []byte("block 32"))
	block42 := ctx.bfactory.CreateBlock(block32, qcb2, []byte("block 42"))
	block52 := ctx.bfactory.CreateBlock(block42, qcb2, []byte("block 52"))

	_, _ = ctx.bc.AddBlock(block1)
	_, _ = ctx.bc.AddBlock(block2)
	_, _ = ctx.bc.AddBlock(block3)
	_, _ = ctx.bc.AddBlock(block4)
	_, _ = ctx.bc.AddBlock(block32)
	_, _ = ctx.bc.AddBlock(block42)
	_, _ = ctx.bc.AddBlock(block52)
	ctx.bc.OnCommit(block1)

	//ctx.pacer.OnNextView()
	//ctx.pacer.OnNextView()
	//ctx.pacer.OnNextView()
	//ctx.pacer.OnNextView()
	//ctx.pacer.OnNextView()

	qcb4 := ctx.bfactory.CreateQC(block4)
	block51 := ctx.bfactory.CreateBlock(block4, qcb4, []byte("block 51"))
	p6, m6 := ctx.bfactory.CreateProposalForParentWithSC(6, block51, qcb4, []byte("block 6"))

	ctx.SendBlocks([]api.Block{block2, block3, block4, block51})
	ctx.hotstuffChan <- m6

	ctx.waitTimeout(timeout)

	assert.Equal(t, block2.Header(), ctx.bc.GetTopCommittedBlock().Header())
	assert.Equal(t, int32(6), ctx.protocol.Vheight())
	assert.Equal(t, p6.NewBlock().Header(), ctx.bc.GetBlockByHeight(6)[0].Header())
}

//Scenario 6e: Fork Cleanup
//Faulty proposer 5 can send bad proposal for us only, trying to isolate us.
//After that 6-th proposer send us good proposal 6 and good proposal 5 (villain sent different proposals)
//We commit block 2,3 after that and reject B21<-B32<-B42-B52
//Start new epoch
//Replica
//Receive fork Gc<-B1c<-B2<-B3<-B4
//Receive proposal fork B21<-B32<-B42-B52
//Receive proposal fork B4-B5-B6
func TestScenario6e(t *testing.T) {
	ctx := initContext(t)

	timeout, f := context.WithTimeout(context.Background(), 14*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hotstuffChan)
	defer f()

	ctx.StartFirstView(timeout)
	ctx.setMe(8)

	p1, m1 := ctx.bfactory.CreateProposalForParent(1, ctx.bc.GetGenesisBlock(), ctx.bc.GetGenesisCert(), []byte("block 1"))
	qcb1 := ctx.bfactory.CreateQC(p1.NewBlock())
	p2, m2 := ctx.bfactory.CreateProposalForParent(2, p1.NewBlock(), qcb1, []byte("block 2"))
	qcb2 := ctx.bfactory.CreateQC(p2.NewBlock())
	p3, m3 := ctx.bfactory.CreateProposalForParent(3, p2.NewBlock(), qcb2, []byte("block 3"))
	p4, m4 := ctx.bfactory.CreateProposalForParent(4, p3.NewBlock(), ctx.createQC(p3.NewBlock()), []byte("block 4"))

	block32 := ctx.bfactory.CreateBlock(p2.NewBlock(), qcb2, []byte("block 32"))
	block42 := ctx.bfactory.CreateBlock(block32, qcb2, []byte("block 42"))
	_, m52 := ctx.bfactory.CreateProposalForParentWithSC(5, block42, qcb2, []byte("block 52"))

	ctx.SendBlocks([]api.Block{p2.NewBlock(), block32, block42})
	ctx.hotstuffChan <- m1
	ctx.hotstuffChan <- m2
	ctx.hotstuffChan <- m3
	ctx.hotstuffChan <- m4
	ctx.hotstuffChan <- m52

	ctx.waitVoteOnHeight(timeout, 5)

	block5 := ctx.bfactory.CreateBlock(p4.NewBlock(), ctx.createQC(p4.NewBlock()), []byte("block 5"))
	_, m6 := ctx.bfactory.CreateProposalForParent(6, block5, ctx.createQC(block5), []byte("block 6"))
	ctx.hotstuffChan <- m6

	ctx.SendBlocks([]api.Block{p2.NewBlock(), p3.NewBlock(), p4.NewBlock(), block5})

	ctx.waitTimeout(timeout)

	assert.Equal(t, int32(6), ctx.protocol.Vheight())
	assert.Equal(t, p3.NewBlock().Header(), ctx.bc.GetTopCommittedBlock().Header())

	//check that state is released
	_, b := ctx.stateDB.Get(block32.Header().Hash())
	assert.False(t, b)

}

//Scenario 6f: Proposal block do not extend it's QC
func TestScenario6f(t *testing.T) {
	ctx := initContext(t)

	timeout, f := context.WithTimeout(context.Background(), 14*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hotstuffChan)
	defer f()

	ctx.StartFirstView(timeout)
	ctx.setMe(8)

	p1, m1 := ctx.bfactory.CreateProposalForParent(1, ctx.bc.GetGenesisBlock(), ctx.bc.GetGenesisCert(), []byte("block 1"))
	qcb1 := ctx.bfactory.CreateQC(p1.NewBlock())
	p2, m2 := ctx.bfactory.CreateProposalForParent(2, p1.NewBlock(), qcb1, []byte("block 2"))
	qcb2 := ctx.bfactory.CreateQC(p2.NewBlock())
	p3, m3 := ctx.bfactory.CreateProposalForParent(3, p2.NewBlock(), qcb2, []byte("block 3"))
	_, m4 := ctx.bfactory.CreateProposalForParent(4, p3.NewBlock(), ctx.createQC(p3.NewBlock()), []byte("block 4"))

	block21 := ctx.bfactory.CreateBlock(p1.NewBlock(), qcb2, []byte("block 21"))
	block32 := ctx.bfactory.CreateBlock(block21, qcb2, []byte("block 32"))
	block42 := ctx.bfactory.CreateBlock(block32, qcb2, []byte("block 42"))
	_, m52 := ctx.bfactory.CreateProposalForParentWithSC(5, block42, qcb2, []byte("block 52"))

	ctx.SendBlocks([]api.Block{block21, block32, block42})
	ctx.hotstuffChan <- m1
	ctx.hotstuffChan <- m2
	ctx.hotstuffChan <- m3
	ctx.hotstuffChan <- m4
	ctx.hotstuffChan <- m52

	ctx.waitTimeout(timeout)
	assert.Equal(t, int32(4), ctx.protocol.Vheight())
}

//Scenario 7a: Propose block with transaction
func TestScenario7a(t *testing.T) {
	ctx := initContext(t)

	timeout, f := context.WithTimeout(context.Background(), 10*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hotstuffChan)
	defer f()

	ctx.setMe(1)
	ctx.generatePayment(ctx.peers[1].GetAddress(), ctx.peers[2].GetAddress(), big.NewInt(100), big.NewInt(1))

	ctx.StartFirstView(timeout)

	message := <-ctx.proposalChan
	assert.Equal(t, pb.Message_PROPOSAL, message.Type)
	proposal, _ := hotstuff.CreateProposalFromMessage(message)
	assert.Equal(t, 1, proposal.NewBlock().TxsCount())
	assert.Equal(t, "0xe656900a2d36a5e15ee19a4a6cfbd851563576d916682d72b1f686c4410728a0", proposal.NewBlock().Header().TxHash().Hex())
	assert.Equal(t, "0x7509e1371b6d9293bac6ed2e32661978a23d9ff4352f661bdfcb31c3370922b5", proposal.NewBlock().Header().StateHash().Hex())

}

//Scenario 7aa: Propose block with transactions
func TestScenario7aa(t *testing.T) {
	ctx := initContext(t)

	timeout, f := context.WithTimeout(context.Background(), 10*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hotstuffChan)
	defer f()

	ctx.setMe(1)

	ctx.generatePayment(ctx.peers[2].GetAddress(), ctx.peers[3].GetAddress(), big.NewInt(100), big.NewInt(1))
	ctx.generatePayment(ctx.peers[1].GetAddress(), ctx.peers[3].GetAddress(), big.NewInt(100), big.NewInt(1))

	ctx.StartFirstView(timeout)

	message := <-ctx.proposalChan
	assert.Equal(t, pb.Message_PROPOSAL, message.Type)
	proposal, _ := hotstuff.CreateProposalFromMessage(message)
	assert.Equal(t, "0x27abcbb0d0e698ef72621fba95a133d4337048acc0aa45e669b11c064de42ba3", proposal.NewBlock().Header().TxHash().Hex())
	assert.Equal(t, "0x51de6155ccb435024a7874b4f933efacad07e49cf0776738d706ff3b52211df7", proposal.NewBlock().Header().StateHash().Hex())
	assert.Equal(t, 2, proposal.NewBlock().TxsCount())

}

//Scenario 7b: Vote for block with transactions
func TestScenario7b(t *testing.T) {
	ctx := initContext(t)

	timeout, f := context.WithTimeout(context.Background(), 10*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hotstuffChan)
	defer f()

	ctx.setMe(0)

	p := []api.Transaction{
		ctx.generatePayment(ctx.peers[2].GetAddress(), ctx.peers[3].GetAddress(), big.NewInt(100), big.NewInt(1)),
		ctx.generatePayment(ctx.peers[1].GetAddress(), ctx.peers[3].GetAddress(), big.NewInt(123), big.NewInt(1)),
	}
	block := ctx.bfactory.CreateBlockWithTxs(ctx.bc.GetHead(), ctx.bc.GetGenesisCert(), []byte("Wunder blok"), p)

	ctx.StartFirstView(timeout)

	_, m := ctx.bfactory.CreateProposalForBlock(1, block, block.QC())
	ctx.hotstuffChan <- m

	vote := <-ctx.voteChan

	assert.Equal(t, pb.Message_VOTE, vote.Type)
	r, b := ctx.stateDB.Get(block.Header().Hash())
	assert.True(t, b)
	get, found := r.Get(ctx.peers[3].GetAddress())
	assert.True(t, found)
	assert.Equal(t, 0, get.Balance().Cmp(big.NewInt(4223)))
}

//Scenario 7c: Commit block with transactions
func TestScenario7c(t *testing.T) {
	ctx := initContext(t)

	timeout, f := context.WithTimeout(context.Background(), 10*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hotstuffChan)
	defer f()

	ctx.StartFirstView(timeout)
	ctx.setMe(8)

	p := []api.Transaction{
		ctx.generatePayment(ctx.peers[2].GetAddress(), ctx.peers[3].GetAddress(), big.NewInt(100), big.NewInt(1)),
		ctx.generatePayment(ctx.peers[1].GetAddress(), ctx.peers[3].GetAddress(), big.NewInt(123), big.NewInt(1)),
	}

	block1 := ctx.bfactory.CreateBlockWithTxs(ctx.bc.GetGenesisBlock(), ctx.bc.GetGenesisCert(), []byte("block 1"), p)
	_, m1 := ctx.bfactory.CreateProposalForBlock(1, block1, ctx.bc.GetGenesisCert())

	ctx.hotstuffChan <- m1

	qcb1 := ctx.bfactory.CreateQC(block1)
	p2, m2 := ctx.bfactory.CreateProposalForParent(2, block1, qcb1, []byte("block 2"))
	ctx.hotstuffChan <- m2

	qcb2 := ctx.bfactory.CreateQC(p2.NewBlock())
	p3, m3 := ctx.bfactory.CreateProposalForParent(3, p2.NewBlock(), qcb2, []byte("block 3"))
	ctx.hotstuffChan <- m3

	qc3 := ctx.bfactory.CreateQC(p3.NewBlock())
	_, m4 := ctx.bfactory.CreateProposalForParent(4, p3.NewBlock(), qc3, []byte("block 4"))
	ctx.hotstuffChan <- m4

	ctx.waitTimeout(timeout)

	topCommitted := ctx.bc.GetTopCommittedBlock()
	assert.Equal(t, int32(1), topCommitted.Height())
	r, _ := ctx.stateDB.Get(topCommitted.Header().Hash())
	get, _ := r.Get(ctx.peers[3].GetAddress())
	assert.Equal(t, 0, big.NewInt(4223).Cmp(get.Balance()))
}

//Scenario 7d: Forks switching and blocks with transactions (release states etc)
//Test state db cleaning and fork transaction release
// Fork1: b1 -> b2 -> b3 -> b4
//1:0x54b 2000 -> -123 -1 + 1 + 1-> 0 -> 0 -> 0 -> 1878
//2:0xCFC 9000 -> -10 -1 -> -1 + 1 -1 -> -2 -1 -> -3 -1 -> 8981
//3:0xdA6 4000 -> +10 + 123 -> + 1 -> +2 +1 -> +3 -> 4140
//4:0x07E 3500 -> 0 -> 0 -> 0 -> +1 -> 3501
//-----------------
// Fork2:         b1 -> b2 -> b32 -> b42 ->b5
//1:0x54b 2000 -> -123 -1 + 1 + 1-> 0 -> 0 -> 0 -> 0 ->1878
//2:0xCFC 9000 -> -10 -1 -> -1 + 1 -1 -> -4 -1 -> -6 -1 -> -3 - 1 ->8972
//3:0xdA6 4000 -> +10 + 123 -> + 1 -> +4 +1 -> +6 -> 3 ->4148
//4:0x07E 3500 -> 0 -> 0 -> 0 -> +1 -> 0 -> 3501
//5:0x07E 7500 -> 0 -> 0 -> 0 -> +0 -> +1 -> 7501
func TestScenario7d(t *testing.T) {
	ctx := initContext(t)

	timeout, f := context.WithTimeout(context.Background(), 10*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hotstuffChan)
	defer f()

	ctx.StartFirstView(timeout)
	ctx.setMe(8)

	p1 := []api.Transaction{
		ctx.generatePayment(ctx.peers[2].GetAddress(), ctx.peers[3].GetAddress(), big.NewInt(10), big.NewInt(1)),
		ctx.generatePayment(ctx.peers[1].GetAddress(), ctx.peers[3].GetAddress(), big.NewInt(123), big.NewInt(1)),
	}
	block1 := ctx.bfactory.CreateBlockWithTxs(ctx.bc.GetGenesisBlock(), ctx.bc.GetGenesisCert(), []byte("block 1"), p1)
	_, m1 := ctx.bfactory.CreateProposalForBlock(1, block1, ctx.bc.GetGenesisCert())
	ctx.hotstuffChan <- m1

	p2 := []api.Transaction{ctx.generatePaymentWithNonce(ctx.peers[2].GetAddress(), ctx.peers[3].GetAddress(), big.NewInt(1), big.NewInt(1), 2)}
	qcb1 := ctx.bfactory.CreateQC(block1)
	block2 := ctx.bfactory.CreateBlockWithTxs(block1, qcb1, []byte("block 2"), p2)
	_, m2 := ctx.bfactory.CreateProposalForBlock(2, block2, qcb1)
	ctx.hotstuffChan <- m2

	p3 := []api.Transaction{ctx.generatePaymentWithNonce(ctx.peers[2].GetAddress(), ctx.peers[3].GetAddress(), big.NewInt(2), big.NewInt(1), 3)}
	qcb2 := ctx.bfactory.CreateQC(block2)
	block3 := ctx.bfactory.CreateBlockWithTxs(block2, qcb2, []byte("block 3"), p3)
	_, m3 := ctx.bfactory.CreateProposalForBlock(3, block3, qcb2)
	ctx.hotstuffChan <- m3

	p4 := []api.Transaction{ctx.generatePaymentWithNonce(ctx.peers[2].GetAddress(), ctx.peers[3].GetAddress(), big.NewInt(3), big.NewInt(1), 4)}
	qcb3 := ctx.createQC(block3)
	block4 := ctx.bfactory.CreateBlockWithTxs(block3, qcb3, []byte("block 4"), p4)
	_, m4 := ctx.bfactory.CreateProposalForBlock(4, block4, qcb3)
	ctx.hotstuffChan <- m4

	p32 := []api.Transaction{ctx.generatePaymentWithNonce(ctx.peers[2].GetAddress(), ctx.peers[3].GetAddress(), big.NewInt(4), big.NewInt(1), 3)}
	block32 := ctx.bfactory.CreateBlockWithTxs(block2, qcb2, []byte("block 32"), p32)

	p42 := []api.Transaction{ctx.generatePaymentWithNonce(ctx.peers[2].GetAddress(), ctx.peers[3].GetAddress(), big.NewInt(6), big.NewInt(1), 4)}
	block42 := ctx.bfactory.CreateBlockWithTxs(block32, qcb2, []byte("block 42"), p42)

	p5 := []api.Transaction{ctx.generatePaymentWithNonce(ctx.peers[2].GetAddress(), ctx.peers[3].GetAddress(), big.NewInt(3), big.NewInt(1), 5)}
	sc4 := ctx.bfactory.CreateSC(block42.Height())
	block5 := ctx.bfactory.CreateBlockWithTxs(block42, qcb2, []byte("block 5"), p5)
	_, m5 := ctx.bfactory.CreateProposalForBlock(5, block5, sc4)

	ctx.SendBlocks([]api.Block{block2, block32, block42})
	ctx.hotstuffChan <- m5

	ctx.waitTimeout(timeout)

	r4, ok4 := ctx.stateDB.Get(block4.Header().Hash())
	assert.True(t, ok4)
	b14, _ := r4.Get(ctx.peers[1].GetAddress())
	b24, _ := r4.Get(ctx.peers[2].GetAddress())
	b34, _ := r4.Get(ctx.peers[3].GetAddress())
	b44, _ := r4.Get(ctx.peers[4].GetAddress())

	assert.ElementsMatch(t,
		[]*big.Int{big.NewInt(1878), big.NewInt(8981), big.NewInt(4140), big.NewInt(3501)},
		[]*big.Int{b14.Balance(), b24.Balance(), b34.Balance(), b44.Balance()},
	)

	r5, ok5 := ctx.stateDB.Get(block5.Header().Hash())
	assert.True(t, ok5)
	b15, _ := r5.Get(ctx.peers[1].GetAddress())
	b25, _ := r5.Get(ctx.peers[2].GetAddress())
	b35, _ := r5.Get(ctx.peers[3].GetAddress())
	b45, _ := r5.Get(ctx.peers[4].GetAddress())

	assert.ElementsMatch(t,
		[]*big.Int{big.NewInt(1878), big.NewInt(8972), big.NewInt(4148), big.NewInt(3501)},
		[]*big.Int{b15.Balance(), b25.Balance(), b35.Balance(), b45.Balance()},
	)

}

//Scenario 8a:
//Settler
//Receive 2f + 1 agreements
//Send proof
//Check reward and settle
//S:0x0F1 0    -> 553(settle) + 10(settle_reward) -> 563 -> -10(settle_reward) - 563(settle) -> 0
//0:0xA60 1000 -> 1000(init) - 13(3fee + 10reward) - 2(agreement_fee) -> 985 -> + 553(settle) - 10(proof_fee) + 4(3 change 1 self reward) = 1532
//1:0x54b 2000 -> 3(proposer_reward) -2(agreement_fee) -> 2001 -> + 1(agreement_reward) = 2002
//2:0xCFC 9000 -> 7 * 2(agreement_fee) - 2(agreement_fee) -> 8998 -> + 1(agreement_reward) = 9013
//3:0xdA6 4000 -> -2(agreement_fee) -> 3998 -> + 1(agreement_reward) -> + 10(proof_fee) = 4009
//4:0x07E 3500 -> -2(agreement_fee) -> 3498 -> + 1(agreement_reward) = 3499
//5:0xCa2 7500 -> -2(agreement_fee) -> 7498 -> + 1(agreement_reward) = 7499
//6:0x660 9000 -> -2(agreement_fee) -> 8998 -> + 1(agreement_reward) = 8999
//7
//8
//9
func TestScenario8a(t *testing.T) {
	ctx := initContext(t)

	timeout, f := context.WithTimeout(context.Background(), 10*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hotstuffChan)
	go ctx.txService.Run(timeout, ctx.txChan)
	defer f()

	ctx.StartFirstView(timeout)
	ctx.setMe(0)

	s := []api.Transaction{
		ctx.generateSettlement(ctx.me.GetAddress(), big.NewInt(553), big.NewInt(api.DefaultSettlementReward+3)),
	}

	block1 := ctx.bfactory.CreateBlockWithTxs(ctx.bc.GetGenesisBlock(), ctx.bc.GetGenesisCert(), []byte("block 1"), s)
	_, m := ctx.bfactory.CreateProposalForBlock(1, block1, ctx.bc.GetGenesisCert())
	ctx.hotstuffChan <- m

	var agreements []api.Transaction
	var proofs [][]byte
	for i := 0; i < 2*len(ctx.peers)/3+1; i++ {
		nonce := 1
		if bytes.Equal(ctx.peers[i].GetAddress().Bytes(), ctx.me.GetAddress().Bytes()) {
			nonce = 2
		}

		agreement := tx2.CreateAgreement(s[0], uint64(nonce), nil)
		agreement.SetFrom(ctx.peers[i].GetAddress())
		if err := agreement.CreateProof(ctx.peers[i].GetPrivateKey()); err != nil {
			t.Error(err)
		}
		agreement.Sign(ctx.peers[i].GetPrivateKey())
		proofs = append(proofs, agreement.Data())
		agreements = append(agreements, agreement)
	}

	block2 := ctx.bfactory.CreateBlockWithTxs(block1, ctx.bfactory.CreateQC(block1), []byte("block 1"), agreements)
	_, m2 := ctx.bfactory.CreateProposalForBlock(2, block2, block2.QC())
	ctx.hotstuffChan <- m2

	to := common2.BytesToAddress(s[0].Hash().Bytes()[12:])
	log.Debugf("Contract address %v", to.Hex())
	snap := ctx.bc.GetHeadRecord()
	_, found := snap.Get(to)
	if !found {
		t.Error("snapshot not found")
	}

	mappedProofs := make(map[common2.Address]*crypto.Signature)
	var signs []*crypto.Signature
	for _, proof := range proofs {
		pbSign := &pb.Signature{}
		if err := proto.Unmarshal(proof, pbSign); err != nil {
			t.Error(err)
			return
		}
		signFromProto := crypto.SignatureFromProto(pbSign)
		signs = append(signs, signFromProto)
		address := crypto.PubkeyToAddress(crypto.NewPublicKey(signFromProto.Pub()))
		mappedProofs[address] = signFromProto
	}
	bitmap := ctx.pacer.GetBitmap(mappedProofs)
	//spew.Dump(mappedProofs)
	aggregate := crypto.AggregateSignatures(bitmap, signs)
	prAggr := aggregate.ToProto()
	aggrBytes, _ := proto.Marshal(prAggr)

	proofTran := []api.Transaction{
		tx2.CreateTransaction(api.Proof, to, ctx.me.GetAddress(), 3, big.NewInt(0), big.NewInt(10), aggrBytes),
	}
	proofTran[0].Sign(ctx.me.GetPrivateKey())

	block3 := ctx.bfactory.CreateBlockWithTxs(block2, ctx.bfactory.CreateQC(block2), []byte("block 3"), proofTran)
	_, m3 := ctx.bfactory.CreateProposalForBlock(3, block3, block3.QC())
	ctx.hotstuffChan <- m3

	ctx.waitTimeout(timeout)

	head := ctx.bc.GetHeadRecord()

	a1, _ := head.Get(ctx.peers[0].GetAddress())
	a2, _ := head.Get(ctx.peers[1].GetAddress())
	a3, _ := head.Get(ctx.peers[2].GetAddress())
	a4, _ := head.Get(ctx.peers[3].GetAddress())
	a5, _ := head.Get(ctx.peers[4].GetAddress())
	a6, _ := head.Get(ctx.peers[5].GetAddress())
	a7, _ := head.Get(ctx.peers[6].GetAddress())
	//hash of settlement transaction is converted to address
	as, _ := head.Get(common2.HexToAddress("0x2A8d5E4695F170D042376c888aE1f8f4B9A64A9c"))

	assert.Equal(t, big.NewInt(1532), a1.Balance())
	assert.Equal(t, big.NewInt(2002), a2.Balance())
	assert.Equal(t, big.NewInt(9013), a3.Balance())
	assert.Equal(t, big.NewInt(4009), a4.Balance())
	assert.Equal(t, big.NewInt(3499), a5.Balance())
	assert.Equal(t, big.NewInt(7499), a6.Balance())
	assert.Equal(t, big.NewInt(8999), a7.Balance())
	assert.Equal(t, big.NewInt(0), as.Balance())
}

func (ctx *TestContext) generateTransaction(from, to common2.Address, amount, fee *big.Int, nonce uint64, txType api.Type) api.Transaction {
	trans := tx2.CreateTransaction(txType, to, from, nonce, amount, fee, []byte(""))
	for _, peer := range ctx.peers {
		if bytes.Equal(peer.GetAddress().Bytes(), from.Bytes()) {
			trans.Sign(peer.GetPrivateKey())
		}
	}
	ctx.pool.Add(trans)

	return trans
}

func (ctx *TestContext) generatePayment(from, to common2.Address, amount, fee *big.Int) api.Transaction {
	return ctx.generateTransaction(from, to, amount, fee, 1, api.Payment)
}
func (ctx *TestContext) generatePaymentWithNonce(from, to common2.Address, amount, fee *big.Int, nonce uint64) api.Transaction {
	return ctx.generateTransaction(from, to, amount, fee, nonce, api.Payment)
}

func (ctx *TestContext) generateSettlement(from common2.Address, amount, fee *big.Int) api.Transaction {
	trans := tx2.CreateTransaction(api.Settlement, common2.HexToAddress(api.SettlementAddressHex), from, 1, amount, fee, []byte(""))
	for _, peer := range ctx.peers {
		if bytes.Equal(peer.GetAddress().Bytes(), from.Bytes()) {
			trans.Sign(peer.GetPrivateKey())
		}
	}
	trans.SetTo(common2.BytesToAddress(trans.Hash().Bytes()[12:]))

	return trans
}

type TestContext struct {
	peers         []*common.Peer
	pacer         *hotstuff.StaticPacer
	protocol      *hotstuff.Protocol
	cfg           *hotstuff.ProtocolConfig
	bc            api.Blockchain
	bsrv          blockchain.BlockService
	pool          tx.TransactionPool
	eventChan     chan api.Event
	voteChan      chan *DirectMessage
	syncChan      chan *msg.Message
	proposalChan  chan *msg.Message
	me            *common.Peer
	hotstuffChan  chan *msg.Message
	txChan        chan *msg.Message
	sentTxChan    chan *msg.Message
	headersChan   chan *msg.Message
	headersToSend map[int32][]*pb.BlockHeader
	blocksToSend  map[common2.Hash]*msg.Message
	blocksChan    chan *msg.Message
	seed          map[common2.Address]api.Account
	stateDB       state.DB
	txService     *tx.TxService
	bfactory      *BlockFactory
}

type DirectMessage struct {
	*msg.Message
	to *common.Peer
}

func (ctx *TestContext) makeVotes(count int, newBlock api.Block) []*msg.Message {
	votes := make([]*msg.Message, count)

	for i := 0; i < count; i++ {
		vote := makeVote(newBlock, ctx.peers[i])
		vote.Sign(ctx.peers[i].GetPrivateKey())
		any, _ := ptypes.MarshalAny(vote.GetMessage())
		votes[i] = msg.CreateMessage(pb.Message_VOTE, any, ctx.peers[i])
	}

	return votes
}

func makeVote(newBlock api.Block, peer *common.Peer) api.Vote {
	vote := hotstuff.CreateVote(newBlock.Header(), newBlock.QC(), peer)
	vote.Sign(peer.GetPrivateKey())
	return vote
}

func (ctx *TestContext) StartFirstView(c context.Context) api.Sync {
	ctx.waitTimeout(c)
	sm := <-ctx.syncChan
	sync, _ := hotstuff.CreateSyncFromMessage(sm)

	trigger := make(chan interface{})
	ctx.pacer.SubscribeViewChange(context.Background(), func(event api.Event) {
		trigger <- event.Payload
	})
	ctx.sendMoreSyncMessages(0, 2*ctx.cfg.F/3+1, 0, nil, true)
	for e := range trigger {
		if e == int32(1) {
			return sync
		}
	}
	return nil
}

func (ctx *TestContext) sendMoreSyncMessages(start int, amount int, height int32, cert api.Certificate, voting bool) {
	for i := start; i < amount; i++ {
		var sync api.Sync
		p := ctx.pacer.Committee()[i]
		if cert == nil {
			sync = hotstuff.CreateSync(height, voting, ctx.bc.GetGenesisCert(), p)
		} else {
			sync = hotstuff.CreateSync(height, voting, cert, p)
		}
		sync.Sign(p.GetPrivateKey())
		any, err := ptypes.MarshalAny(sync.GetMessage())
		if err != nil {
			log.Error(err)
			return
		}
		m := msg.CreateMessage(pb.Message_SYNCHRONIZE, any, p)
		ctx.hotstuffChan <- m
	}
}
func (ctx *TestContext) sendSyncMessages(amount int, hqc api.QuorumCertificate) {
	ctx.sendMoreSyncMessages(0, amount, hqc.Height()+1, hqc, true)
}

func (ctx *TestContext) setMe(peerNumber int) {
	peer := ctx.pacer.Committee()[peerNumber]
	ctx.pacer.Committee()[peerNumber] = ctx.me
	ctx.pacer.Committee()[0] = peer

}

func (ctx *TestContext) waitRounds(c context.Context, count int) {
	var i int
	for {
		select {
		case event := <-ctx.eventChan:
			if event.T == api.ChangedView {
				i++
				if i == count {
					return
				}
			}
		case <-c.Done():
			return
		}
	}
}
func (ctx *TestContext) waitTimeout(c context.Context) {
	for {
		select {
		case event := <-ctx.eventChan:
			if event.T == api.TimedOut {
				return
			}
		case <-c.Done():
			return
		}
	}
}

func (ctx *TestContext) waitVoteOnHeight(c context.Context, height int32) {
	for {
		select {
		case vote := <-ctx.voteChan:
			v, _ := hotstuff.CreateVoteFromMessage(vote.Message)
			if v.Header().Height() == height {
				return
			}
		case <-c.Done():
			return
		}
	}
}

func (ctx *TestContext) CreateProposal(newBlock api.Block, peerNumber int) *msg.Message {
	proposal := hotstuff.CreateProposal(newBlock, ctx.peers[peerNumber], newBlock.QC())
	proposal.Sign(ctx.peers[peerNumber].GetPrivateKey())
	any, _ := ptypes.MarshalAny(proposal.GetMessage())
	return msg.CreateMessage(pb.Message_PROPOSAL, any, ctx.peers[peerNumber])
}

func (ctx *TestContext) createQC(block api.Block) api.QuorumCertificate {
	var signs []*crypto.Signature
	signsByAddress := make(map[common2.Address]*crypto.Signature)

	for i := 0; i < 2*ctx.cfg.F/3+1; i++ {
		s := block.Header().Sign(ctx.peers[i].GetPrivateKey())
		signs = append(signs, s)
		signsByAddress[ctx.peers[i].GetAddress()] = s
	}
	bitmap := ctx.pacer.GetBitmap(signsByAddress)
	aggregate := crypto.AggregateSignatures(bitmap, signs)

	return blockchain.CreateQuorumCertificate(aggregate, block.Header(), api.QRef)
}

func (ctx *TestContext) SendBlocks(blocks []api.Block) {
	hs := headers(blocks)
	for _, h := range hs {
		blockHeaders := ctx.headersToSend[h.Height()]
		blockHeaders = append(blockHeaders, h.GetMessage())
		ctx.headersToSend[h.Height()] = blockHeaders
	}

	for _, b := range blocks {
		pblock := b.GetMessage()
		p := &pb.BlockResponsePayload_Block{Block: pblock}
		payload := &pb.BlockResponsePayload{Response: p}
		any, e := ptypes.MarshalAny(payload)
		if e != nil {
			log.Error("can't send blocks")
			return
		}
		ctx.blocksToSend[b.Header().Hash()] = msg.CreateMessage(pb.Message_BLOCK_RESPONSE, any, ctx.me)
	}
}

func initContext(t *testing.T) *TestContext {
	identity := generateIdentity(t, 0)
	log.Debugf("Current me %v", identity.GetAddress().Hex())
	peers := make([]*common.Peer, 10)
	var committee []common2.Address
	for i := 0; i < 10; i++ {
		peers[i] = generateIdentity(t, i)
		committee = append(committee, peers[i].GetAddress())
	}

	srv := &mocks.Service{}
	storage := SoftStorageMock()

	bpersister := &blockchain.BlockPersister{storage}
	cpersister := &blockchain.BlockchainPersister{storage}

	pool := tx.NewTransactionPool()
	seed := blockchain.SeedFromFile("../static/seed.json")

	stateDb := state.NewStateDB(storage, committee, &common.NullBus{})

	bc := blockchain.CreateBlockchainFromGenesisBlock(&blockchain.BlockchainConfig{
		Seed:           seed,
		BlockPerister:  bpersister,
		ChainPersister: cpersister,
		Pool:           pool,
		Db:             stateDb,
		Delta:          1 * time.Millisecond, //todo we have a lot of tests where blocks has no transactions, and we create several blocks per round. probably we must add instant block generation without transactions for tests
		Storage:        storage,
	})

	blockValidator := MockGoodBlockValidator()
	validators := []api.Validator{
		hotstuff.NewSyncValidator(peers),
		hotstuff.NewProposalValidator(peers, blockValidator),
		hotstuff.NewVoteValidator(peers),
	}
	bsrv := blockchain.NewBlockService(srv, blockValidator, MockGoodHeaderValidator())
	sync := blockchain.CreateSynchronizer(identity, bsrv, bc, blockValidator, -1, 20, 1, 3, -1, 2*10)

	config := &hotstuff.ProtocolConfig{
		F:            10,
		Delta:        1 * time.Second,
		Blockchain:   bc,
		Me:           identity,
		Srv:          srv,
		Sync:         sync,
		Validators:   validators,
		Committee:    peers,
		Storage:      storage,
		InitialState: hotstuff.DefaultState(bc),
	}

	matcher := func(msgType pb.Message_MessageType) func(m *msg.Message) bool {
		return func(m *msg.Message) bool {
			return m.Type == msgType
		}
	}

	syncChan := make(chan *msg.Message)
	srv.On("Broadcast", mock.MatchedBy(func(ctx context.Context) bool { return true }), mock.MatchedBy(matcher(pb.Message_SYNCHRONIZE))).Run(func(args mock.Arguments) {
		syncChan <- (args[1]).(*msg.Message)
	})
	proposalChan := make(chan *msg.Message)
	srv.On("Broadcast", mock.MatchedBy(func(ctx context.Context) bool { return true }), mock.MatchedBy(matcher(pb.Message_PROPOSAL))).Run(func(args mock.Arguments) {
		proposalChan <- (args[1]).(*msg.Message)
	})
	sentTxChan := make(chan *msg.Message)
	srv.On("BroadcastTransaction", mock.MatchedBy(func(ctx context.Context) bool { return true }), mock.MatchedBy(matcher(pb.Message_TRANSACTION))).Run(func(args mock.Arguments) {
		sentTxChan <- (args[1]).(*msg.Message)
	})

	voteChan := make(chan *DirectMessage)
	srv.On("SendMessage", mock.MatchedBy(func(ctx context.Context) bool { return true }), mock.AnythingOfType("*common.Peer"), mock.MatchedBy(matcher(pb.Message_VOTE))).
		Return(make(chan *msg.Message), nil).
		Run(func(args mock.Arguments) {
			voteChan <- &DirectMessage{Message: (args[2]).(*msg.Message), to: (args[1]).(*common.Peer)}
		})

	blocksChan := make(chan *msg.Message)
	headersChan := make(chan *msg.Message)
	headersToSend := make(map[int32][]*pb.BlockHeader)
	blocksToSend := make(map[common2.Hash]*msg.Message)
	srv.On("SendRequest", mock.MatchedBy(func(ctx context.Context) bool { return true }), mock.AnythingOfType("*common.Peer"),
		mock.MatchedBy(matcher(pb.Message_HEADERS_REQUEST))).Run(
		func(args mock.Arguments) {
			m := args.Get(2).(*msg.Message)
			req := &pb.HeadersRequest{}
			if err := ptypes.UnmarshalAny(m.Payload, req); err != nil {
				log.Error(err)
			}
			go func() {
				headers := &pb.Headers{}
				for i := req.Low + 1; i <= req.High; i++ {
					headers.Headers = append(headers.Headers, headersToSend[i]...)
				}

				payload := &pb.HeadersResponse{Response: &pb.HeadersResponse_Headers{Headers: headers}}
				any, e := ptypes.MarshalAny(payload)
				if e != nil {
					log.Error("can't send blocks")
					return
				}

				headersChan <- msg.CreateMessage(pb.Message_HEADERS_RESPONSE, any, identity)
			}()
		}).Return(headersChan, nil)
	srv.On("SendRequest", mock.MatchedBy(func(ctx context.Context) bool { return true }), mock.AnythingOfType("*common.Peer"),
		mock.MatchedBy(matcher(pb.Message_BLOCK_REQUEST))).Run(
		func(args mock.Arguments) {
			m := args.Get(2).(*msg.Message)
			req := &pb.BlockRequestPayload{}
			if err := ptypes.UnmarshalAny(m.Payload, req); err != nil {
				log.Error(err)
			}
			go func() {
				blocksChan <- blocksToSend[common2.BytesToHash(req.Hash)]
			}()
		}).Return(blocksChan, nil)

	txService := tx.NewService(blockchain.NewTransactionValidator(peers), pool, srv, bc, identity)

	pacer := hotstuff.CreatePacer(config)
	config.Pacer = pacer
	p := hotstuff.CreateProtocol(config)
	pacer.Bootstrap(context.Background(), p)

	eventChan := make(chan api.Event)
	pacer.SubscribeProtocolEvents(eventChan)
	bc.SetProposerGetter(pacer)

	hottuffChan := make(chan *msg.Message)
	txChan := make(chan *msg.Message)
	return &TestContext{
		voteChan:      voteChan,
		syncChan:      syncChan,
		peers:         peers,
		pacer:         pacer,
		protocol:      p,
		bc:            bc,
		cfg:           config,
		seed:          seed,
		proposalChan:  proposalChan,
		blocksChan:    blocksChan,
		blocksToSend:  blocksToSend,
		headersChan:   headersChan,
		headersToSend: headersToSend,
		eventChan:     eventChan,
		me:            identity,
		bsrv:          bsrv,
		pool:          pool,
		hotstuffChan:  hottuffChan,
		stateDB:       stateDb,
		txService:     txService,
		txChan:        txChan,
		sentTxChan:    sentTxChan,
		bfactory:      NewBlockFactory(peers),
	}
}

func headers(blocks []api.Block) (headers []api.Header) {
	for _, b := range blocks {
		headers = append(headers, b.Header())
	}

	return headers
}
