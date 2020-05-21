package test

import (
	"bytes"
	"context"
	"github.com/davecgh/go-spew/spew"
	net "github.com/gagarinchain/network"
	"github.com/gagarinchain/network/blockchain"
	"github.com/gagarinchain/network/blockchain/state"
	"github.com/gagarinchain/network/blockchain/tx"
	"github.com/gagarinchain/network/common"
	"github.com/gagarinchain/network/common/api"
	common2 "github.com/gagarinchain/network/common/eth/common"
	"github.com/gagarinchain/network/common/eth/crypto"
	msg "github.com/gagarinchain/network/common/message"
	"github.com/gagarinchain/network/common/protobuff"
	"github.com/gagarinchain/network/hotstuff"
	"github.com/gagarinchain/network/mocks"
	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"math/big"
	"testing"
	"time"
)

// Scenario 1a:
// Start new epoch,
// Replica
// Get proposal
// Proposer,
// Collect 2f + 1 votes
// Propose block with new QC
func TestScenario1a(t *testing.T) {
	ctx := initContext(t)
	timeout, f := context.WithTimeout(context.Background(), 10*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hotstuffChan, ctx.epochChan)
	defer f()

	ctx.StartFirstEpoch()
	ctx.setMe(2)

	newBlock := ctx.bc.NewBlock(ctx.bc.GetHead(), ctx.bc.GetGenesisCert(), []byte("wonderful block"))

	p := ctx.createProposal(newBlock, 1)
	ctx.hotstuffChan <- p

	votes := ctx.makeVotes(2*ctx.cfg.F/3+1, newBlock)
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

// Scenario 1b:
// Start new epoch,
// Replica, Next proposer
// Collect N votes,
// Receive proposal,
// Proposer,
// Collect 2*f + 1 - N votes
// Propose block with new QC
func TestScenario1b(t *testing.T) {
	ctx := initContext(t)
	timeout, f := context.WithTimeout(context.Background(), 10*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hotstuffChan, ctx.epochChan)
	defer f()

	ctx.StartFirstEpoch()
	ctx.setMe(2)

	newBlock := ctx.bc.NewBlock(ctx.bc.GetHead(), ctx.bc.GetGenesisCert(), []byte("wonderful block"))

	votes := ctx.makeVotes(2*ctx.cfg.F/3+1, newBlock)
	for _, v := range votes[:2*ctx.cfg.F/3-2] {
		ctx.hotstuffChan <- v
	}

	p := ctx.createProposal(newBlock, 1)
	ctx.hotstuffChan <- p

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

// Scenario 1c: Here we receive proposal after we collected votes
// Start new epoch,
// Replica, Next proposer
// Collect 2*f + 1 votes,
// Receive proposal,
// Proposer,
// Propose block with new QC
func TestScenario1c(t *testing.T) {
	ctx := initContext(t)

	timeout, f := context.WithTimeout(context.Background(), 10*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hotstuffChan, ctx.epochChan)
	defer f()

	defer f()
	//defer ctx.protocol.Stop()

	ctx.StartFirstEpoch()
	ctx.setMe(2)

	newBlock := ctx.bc.NewBlock(ctx.bc.GetHead(), ctx.bc.GetGenesisCert(), []byte("wonderful block"))

	votes := ctx.makeVotes(2*ctx.cfg.F/3+1, newBlock)
	for _, v := range votes {
		ctx.hotstuffChan <- v
	}

	go func() {
		ctx.SendBlocks([]api.Block{newBlock})
	}()

	p := ctx.createProposal(newBlock, 1)
	ctx.hotstuffChan <- p

	proposal := <-ctx.proposalChan

	payload := &pb.ProposalPayload{}
	if err := ptypes.UnmarshalAny(proposal.Payload, payload); err != nil {
		log.Error(err)
	}

	ctx.waitRounds(timeout, 2)

	assert.Equal(t, int32(2), payload.Block.GetHeader().GetHeight())
	assert.Equal(t, int32(1), payload.Block.GetCert().GetHeader().GetHeight())
}

//Scenario 1d:
//Start new epoch
//Replica
//Receive proposal, vote
//Next proposer don't propose
//Send last vote again and again and again
func TestScenario1d(t *testing.T) {
	ctx := initContext(t)

	timeout, f := context.WithTimeout(context.Background(), 10*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hotstuffChan, ctx.epochChan)
	defer f()

	ctx.StartFirstEpoch()
	ctx.setMe(6)

	newBlock := ctx.bc.NewBlock(ctx.bc.GetHead(), ctx.bc.GetGenesisCert(), []byte("wonderful block"))

	p := ctx.createProposal(newBlock, 1)
	ctx.hotstuffChan <- p

	vote1 := <-ctx.voteChan
	vote2 := <-ctx.voteChan
	vote3 := <-ctx.voteChan

	assert.Equal(t, vote1.Message, vote2.Message)
	assert.Equal(t, vote2.Message, vote3.Message)
	assert.Equal(t, ctx.peers[2], vote1.to)
	assert.Equal(t, ctx.peers[3], vote2.to)
	assert.Equal(t, ctx.peers[4], vote3.to)

}

//Scenario 2:
//Start new epoch
//Replica
//Get proposal
//Vote
func TestScenario2(t *testing.T) {
	ctx := initContext(t)
	timeout, f := context.WithTimeout(context.Background(), 10*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hotstuffChan, ctx.epochChan)
	defer f()

	ctx.StartFirstEpoch()
	ctx.setMe(0)

	newBlock := ctx.bc.NewBlock(ctx.bc.GetHead(), ctx.bc.GetGenesisCert(), []byte("wonderful block"))
	p := ctx.createProposal(newBlock, 1)
	ctx.hotstuffChan <- p

	vote := <-ctx.voteChan

	assert.Equal(t, pb.Message_VOTE, vote.Type)

}

//Scenario 2b:
//Start new epoch 0
//Receive messages for starting epoch 2
//Start epoch 2
func TestScenario2b(t *testing.T) {
	ctx := initContext(t)

	timeout, f := context.WithTimeout(context.Background(), 10*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hotstuffChan, ctx.epochChan)
	defer f()

	epochChanged := make(chan api.Event)
	ctx.pacer.SubscribeEpochChange(timeout, func(event api.Event) {
		epochChanged <- event
	})
	viewChanged := make(chan api.Event)
	ctx.pacer.SubscribeViewChange(timeout, func(event api.Event) {
		viewChanged <- event
	})
	ctx.sendStartEpochMessages(2, 2*ctx.cfg.F/3+1, nil)
	event := <-epochChanged
	assert.Equal(t, int32(2), event.Payload.(int32))

	event = <-viewChanged
	assert.Equal(t, int32(21), event.Payload.(int32))

}

//Scenario 3:
//Start new epoch
//Proposer
//Propose block with previous QC after Delta
func TestScenario3(t *testing.T) {
	ctx := initContext(t)

	timeout, f := context.WithTimeout(context.Background(), 10*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hotstuffChan, ctx.epochChan)
	defer f()

	ctx.StartFirstEpoch()
	ctx.setMe(2)

	newBlock := ctx.bc.NewBlock(ctx.bc.GetHead(), ctx.bc.GetGenesisCert(), []byte("wonderful block"))
	p := ctx.createProposal(newBlock, 1)
	ctx.hotstuffChan <- p
	<-ctx.voteChan

	proposal := <-ctx.proposalChan

	payload := &pb.ProposalPayload{}
	if err := ptypes.UnmarshalAny(proposal.Payload, payload); err != nil {
		log.Error(err)
	}

	assert.Equal(t, int32(2), payload.Block.GetHeader().GetHeight())
	assert.Equal(t, int32(0), payload.Block.GetCert().GetHeader().GetHeight())
}

//Scenario 4:
//Start new epoch
//Next Proposer
//Proposer equivocates, no proposal sent
//Propose block with previous QC (2) after 2*Delta
func TestScenario4(t *testing.T) {
	ctx := initContext(t)

	timeout, f := context.WithTimeout(context.Background(), 10*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hotstuffChan, ctx.epochChan)
	defer f()

	ctx.StartFirstEpoch()
	ctx.setMe(2)

	proposal := <-ctx.proposalChan

	payload := &pb.ProposalPayload{}
	if err := ptypes.UnmarshalAny(proposal.Payload, payload); err != nil {
		log.Error(err)
	}

	assert.Equal(t, int32(2), payload.Block.GetHeader().GetHeight())
	assert.Equal(t, int32(0), payload.Block.GetCert().GetHeader().GetHeight())
}

//Scenario 5a:
//Start new epoch
//Receive no messages
//Receive 2f + 1 start messages
//Start new epoch
//Propose
func TestScenario5a(t *testing.T) {
	ctx := initContext(t)

	timeout, f := context.WithTimeout(context.Background(), 12*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hotstuffChan, ctx.epochChan)
	defer f()

	ctx.StartFirstEpoch()

	ctx.setMe(1)

	log.Infof("me %v", ctx.pacer.Committee()[0].GetAddress().Hex())

	proposal := <-ctx.proposalChan
	time.Sleep(8 * ctx.cfg.Delta)
	ctx.sendStartEpochMessages(2, 2*ctx.cfg.F/3+1, ctx.protocol.HQC())
	<-ctx.startChan //mine start message

	proposal = <-ctx.proposalChan

	payload := &pb.ProposalPayload{}
	if err := ptypes.UnmarshalAny(proposal.Payload, payload); err != nil {
		log.Error(err)
	}

	assert.Equal(t, int32(21), payload.Block.GetHeader().GetHeight())
	assert.Equal(t, int32(0), payload.Block.GetCert().GetHeader().GetHeight())
}

//Scenario 5b:
//Start new epoch
//Receive no messages
//Start new epoch after epoch end
//Propose
func TestScenario5b(t *testing.T) {
	ctx := initContext(t)

	timeout, f := context.WithTimeout(context.Background(), 10*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hotstuffChan, ctx.epochChan)
	defer f()

	ctx.StartFirstEpoch()
	ctx.setMe(1)

	log.Infof("me %v", ctx.pacer.Committee()[0].GetAddress().Hex())

	<-ctx.startChan
	ctx.sendStartEpochMessages(1, 2*ctx.cfg.F/3+1, ctx.protocol.HQC())

	proposal := <-ctx.proposalChan

	payload := &pb.ProposalPayload{}
	if err := ptypes.UnmarshalAny(proposal.Payload, payload); err != nil {
		log.Error(err)
	}

	assert.Equal(t, int32(11), payload.Block.GetHeader().GetHeight())
	assert.Equal(t, int32(0), payload.Block.GetCert().GetHeader().GetHeight())
}

//Scenario 5c:
//Start new epoch
//Receive no messages
//Receive no messages on epoch start twice
//Receive 2f + 1 start messages
//Start new epoch
//Propose
func TestScenario5c(t *testing.T) {
	ctx := initContext(t)

	timeout, f := context.WithTimeout(context.Background(), 40*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hotstuffChan, ctx.epochChan)
	defer f()

	ctx.StartFirstEpoch()
	ctx.setMe(1)

	log.Infof("me %v", ctx.pacer.Committee()[0].GetAddress().Hex())
	proposal := <-ctx.proposalChan

	time.Sleep(26 * ctx.cfg.Delta)
	ctx.sendStartEpochMessages(2, 2*ctx.cfg.F/3+1, ctx.protocol.HQC())
	<-ctx.startChan //mine start message

	proposal = <-ctx.proposalChan

	payload := &pb.ProposalPayload{}
	if err := ptypes.UnmarshalAny(proposal.Payload, payload); err != nil {
		log.Error(err)
	}

	assert.Equal(t, int32(21), payload.Block.GetHeader().GetHeight())
	assert.Equal(t, int32(0), payload.Block.GetCert().GetHeader().GetHeight())
}

//Scenario 5d:
//Replica
//Start new epoch
//Receive f + 1 start messages
//Wait 4 * D
//Receive f start messages
//Start new epoch
func TestScenario5d(t *testing.T) {
	ctx := initContext(t)

	timeout, f := context.WithTimeout(context.Background(), 10*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hotstuffChan, ctx.epochChan)
	defer f()

	ctx.StartFirstEpoch()
	ctx.setMe(2)

	log.Infof("me %v", ctx.pacer.Committee()[2].GetAddress().Hex())

	ctx.sendStartEpochMessages(2, ctx.cfg.F/3+1, ctx.protocol.HQC())
	<-ctx.startChan //mine start message

	time.Sleep(4 * ctx.cfg.Delta)

	ctx.sendMoreStartEpochMessages(2, ctx.cfg.F/3+1, 2*ctx.cfg.F/3+1, ctx.protocol.HQC())

	ctx.waitRounds(timeout, 2)

	assert.Equal(t, int32(21), ctx.pacer.GetCurrentView())
}

//Scenario 5e:
//Replica
//Start new epoch
//Receive f + 1 start messages
//Receive Proposal
//Receive f start messages
//Start new epoch
//Process proposal and vote for it
func TestScenario5e(t *testing.T) {
	ctx := initContext(t)

	timeout, f := context.WithTimeout(context.Background(), 10*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hotstuffChan, ctx.epochChan)
	defer f()

	ctx.StartFirstEpoch()
	ctx.setMe(2)

	log.Infof("me %v", ctx.pacer.Committee()[2].GetAddress().Hex())

	ctx.sendStartEpochMessages(2, ctx.cfg.F/3+1, ctx.protocol.HQC())
	<-ctx.startChan //mine start message

	block := ctx.bc.GetHead()
	for i := 0; i < 20; i++ {
		block = ctx.bc.PadEmptyBlock(block, ctx.protocol.HQC())
	}
	newBlock := ctx.bc.NewBlock(block, ctx.bc.GetGenesisCert(), []byte("wonderful block"))
	p := ctx.createProposal(newBlock, 1)

	go func() {
		ctx.hotstuffChan <- p
	}()

	ctx.sendMoreStartEpochMessages(2, ctx.cfg.F/3+1, 2*ctx.cfg.F/3+1, ctx.protocol.HQC())

	v := <-ctx.voteChan

	vote := &pb.VotePayload{}
	if err := ptypes.UnmarshalAny(v.Payload, vote); err != nil {
		log.Error(err)
	}

	assert.Equal(t, int32(22), ctx.pacer.GetCurrentView())

	assert.Equal(t, int32(21), vote.GetHeader().GetHeight())
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
	go ctx.pacer.Run(timeout, ctx.hotstuffChan, ctx.epochChan)
	defer f()

	ctx.StartFirstEpoch()
	ctx.setMe(8)

	block1 := ctx.bc.NewBlock(ctx.bc.GetGenesisBlock(), ctx.bc.GetGenesisCert(), []byte("block 1"))
	proposal1 := ctx.createProposal(block1, 1)
	ctx.hotstuffChan <- proposal1
	block2 := ctx.bc.NewBlock(block1, ctx.bc.GetGenesisCert(), []byte("block 2"))
	proposal2 := ctx.createProposal(block2, 2)
	ctx.hotstuffChan <- proposal2
	block3 := ctx.bc.NewBlock(block2, ctx.bc.GetGenesisCert(), []byte("block 3"))
	proposal3 := ctx.createProposal(block3, 3)
	ctx.hotstuffChan <- proposal3

	block21 := ctx.bc.NewBlock(block1, ctx.bc.GetGenesisCert(), []byte("block 21"))
	block31 := ctx.bc.NewBlock(block21, ctx.bc.GetGenesisCert(), []byte("block 31"))
	block4 := ctx.bc.NewBlock(block31, ctx.bc.GetGenesisCert(), []byte("block 41"))
	proposal := ctx.createProposal(block4, 4)
	ctx.hotstuffChan <- proposal

	go func() {
		ctx.SendBlocks([]api.Block{block4, block31, block2})
	}()

	ctx.waitRounds(timeout, 5)

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
	go ctx.pacer.Run(timeout, ctx.hotstuffChan, ctx.epochChan)
	defer f()

	ctx.StartFirstEpoch()
	ctx.setMe(8)

	block1 := ctx.bc.NewBlock(ctx.bc.GetGenesisBlock(), ctx.bc.GetGenesisCert(), []byte("block 1"))
	proposal1 := ctx.createProposal(block1, 1)
	ctx.hotstuffChan <- proposal1
	qcb1 := ctx.createQC(block1)
	block2 := ctx.bc.NewBlock(block1, qcb1, []byte("block 2"))
	proposal2 := ctx.createProposal(block2, 2)
	ctx.hotstuffChan <- proposal2
	block3 := ctx.bc.NewBlock(block2, ctx.createQC(block2), []byte("block 3"))
	proposal3 := ctx.createProposal(block3, 3)
	ctx.hotstuffChan <- proposal3
	block4 := ctx.bc.NewBlock(block3, ctx.createQC(block3), []byte("block 4"))
	proposal4 := ctx.createProposal(block4, 4)
	ctx.hotstuffChan <- proposal4

	block22 := ctx.bc.NewBlock(block1, qcb1, []byte("block 22"))
	block32 := ctx.bc.NewBlock(block22, qcb1, []byte("block 32"))
	block42 := ctx.bc.NewBlock(block32, qcb1, []byte("block 42"))
	block52 := ctx.bc.NewBlock(block42, qcb1, []byte("block 52"))
	proposal := ctx.createProposal(block52, 5)
	ctx.hotstuffChan <- proposal

	ctx.SendBlocks([]api.Block{block22, block32, block42})

	ctx.waitRounds(timeout, 6)

	assert.Equal(t, int32(4), ctx.protocol.Vheight())
	spew.Dump(block1)
	spew.Dump(ctx.bc.GetTopCommittedBlock())
	assert.Equal(t, block1.Header(), ctx.bc.GetTopCommittedBlock().Header())
	assert.Equal(t, block52.Header(), ctx.bc.GetBlockByHeight(5)[0].Header())
}

//Scenario 6c:
//Start new epoch
//Replica
//Receive fork Gc<-B1c<-B2<-B3<-B4
//Receive proposal fork B2<-B32<-(B2)B42-B52 (vote for b52 as it extends QREF)
func TestScenario6c(t *testing.T) {
	ctx := initContext(t)

	timeout, f := context.WithTimeout(context.Background(), 10*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hotstuffChan, ctx.epochChan)
	defer f()

	ctx.StartFirstEpoch()
	ctx.setMe(8)

	block1 := ctx.bc.NewBlock(ctx.bc.GetGenesisBlock(), ctx.bc.GetGenesisCert(), []byte("block 1"))
	proposal1 := ctx.createProposal(block1, 1)
	ctx.hotstuffChan <- proposal1
	qcb1 := ctx.createQC(block1)
	block2 := ctx.bc.NewBlock(block1, qcb1, []byte("block 2"))
	proposal2 := ctx.createProposal(block2, 2)
	ctx.hotstuffChan <- proposal2
	qcb2 := ctx.createQC(block2)
	block3 := ctx.bc.NewBlock(block2, qcb2, []byte("block 3"))
	proposal3 := ctx.createProposal(block3, 3)
	ctx.hotstuffChan <- proposal3
	block4 := ctx.bc.NewBlock(block3, ctx.createQC(block3), []byte("block 4"))
	proposal4 := ctx.createProposal(block4, 4)
	ctx.hotstuffChan <- proposal4

	block32 := ctx.bc.NewBlock(block2, qcb2, []byte("block 32"))
	block42 := ctx.bc.NewBlock(block32, qcb2, []byte("block 42"))
	block52 := ctx.bc.NewBlock(block42, qcb2, []byte("block 52"))
	proposal := ctx.createProposal(block52, 5)
	ctx.hotstuffChan <- proposal

	ctx.SendBlocks([]api.Block{block2, block32, block42})

	ctx.waitRounds(timeout, 6)

	assert.Equal(t, int32(5), ctx.protocol.Vheight())
	assert.Equal(t, block1.Header(), ctx.bc.GetTopCommittedBlock().Header())
	assert.Equal(t, block52.Header(), ctx.bc.GetBlockByHeight(5)[0].Header())
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
	go ctx.pacer.Run(timeout, ctx.hotstuffChan, ctx.epochChan)
	defer f()

	ctx.StartFirstEpoch()
	ctx.setMe(8)

	block1 := ctx.bc.NewBlock(ctx.bc.GetGenesisBlock(), ctx.bc.GetGenesisCert(), []byte("block 1"))
	qcb1 := ctx.createQC(block1)
	block2 := ctx.bc.NewBlock(block1, qcb1, []byte("block 2"))
	qcb2 := ctx.createQC(block2)
	block3 := ctx.bc.NewBlock(block2, qcb2, []byte("block 3"))
	block4 := ctx.bc.NewBlock(block3, ctx.createQC(block3), []byte("block 4"))
	block32 := ctx.bc.NewBlock(block2, qcb2, []byte("block 32"))
	block42 := ctx.bc.NewBlock(block32, qcb2, []byte("block 42"))
	block52 := ctx.bc.NewBlock(block42, qcb2, []byte("block 52"))

	_ = ctx.bc.AddBlock(block1)
	_ = ctx.bc.AddBlock(block2)
	_ = ctx.bc.AddBlock(block3)
	_ = ctx.bc.AddBlock(block4)
	_ = ctx.bc.AddBlock(block32)
	_ = ctx.bc.AddBlock(block42)
	_ = ctx.bc.AddBlock(block52)
	ctx.bc.OnCommit(block1)

	ctx.pacer.OnNextView()
	ctx.pacer.OnNextView()
	ctx.pacer.OnNextView()
	ctx.pacer.OnNextView()
	ctx.pacer.OnNextView()

	qcb4 := ctx.createQC(block4)
	block51 := ctx.bc.NewBlock(block4, qcb4, []byte("block 51"))
	block6 := ctx.bc.NewBlock(block51, qcb4, []byte("block 6"))
	proposal6 := ctx.createProposal(block6, 6)
	ctx.hotstuffChan <- proposal6

	ctx.SendBlocks([]api.Block{block2, block3, block4, block51})

	ctx.waitRounds(timeout, 7)

	assert.Equal(t, block2.Header(), ctx.bc.GetTopCommittedBlock().Header())
	assert.Equal(t, int32(6), ctx.protocol.Vheight())
	assert.Equal(t, block6.Header(), ctx.bc.GetBlockByHeight(6)[0].Header())
}

//Scenario 6e: Fork Cleanup
//Faulty proposer 5 can send bad proposal for us only, trying to isolate us.
// After that 6-th proposer send us good proposal 6 and good proposal 5 (valiant sent different proposals)
// We commit block 2,3 after that and reject B21<-B32<-B42-B52
//Start new epoch
//Replica
//Receive fork Gc<-B1c<-B2<-B3<-B4
//Receive proposal fork B21<-B32<-B42-B52
//Receive proposal fork B4-B5-B6
func TestScenario6e(t *testing.T) {
	ctx := initContext(t)

	timeout, f := context.WithTimeout(context.Background(), 14*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hotstuffChan, ctx.epochChan)
	defer f()

	ctx.StartFirstEpoch()
	ctx.setMe(8)

	block1 := ctx.bc.NewBlock(ctx.bc.GetGenesisBlock(), ctx.bc.GetGenesisCert(), []byte("block 1"))
	proposal1 := ctx.createProposal(block1, 1)
	ctx.hotstuffChan <- proposal1
	qcb1 := ctx.createQC(block1)
	block2 := ctx.bc.NewBlock(block1, qcb1, []byte("block 2"))
	proposal2 := ctx.createProposal(block2, 2)
	ctx.hotstuffChan <- proposal2
	qcb2 := ctx.createQC(block2)
	block3 := ctx.bc.NewBlock(block2, qcb2, []byte("block 3"))
	proposal3 := ctx.createProposal(block3, 3)
	ctx.hotstuffChan <- proposal3
	block4 := ctx.bc.NewBlock(block3, ctx.createQC(block3), []byte("block 4"))
	proposal4 := ctx.createProposal(block4, 4)
	ctx.hotstuffChan <- proposal4

	block21 := ctx.bc.NewBlock(block1, qcb2, []byte("block 21"))
	block32 := ctx.bc.NewBlock(block21, qcb2, []byte("block 32"))
	block42 := ctx.bc.NewBlock(block32, qcb2, []byte("block 42"))
	block52 := ctx.bc.NewBlock(block42, qcb2, []byte("block 52"))
	proposal := ctx.createProposal(block52, 5)
	ctx.hotstuffChan <- proposal

	ctx.SendBlocks([]api.Block{block21, block32, block42})
	ctx.waitRounds(timeout, 6)

	block5 := ctx.bc.NewBlock(block4, ctx.createQC(block4), []byte("block 5"))
	block6 := ctx.bc.NewBlock(block5, ctx.createQC(block5), []byte("block 6"))
	proposal6 := ctx.createProposal(block6, 6)
	ctx.hotstuffChan <- proposal6

	ctx.SendBlocks([]api.Block{block2, block3, block4, block5})

	ctx.waitRounds(timeout, 2)

	assert.Equal(t, int32(6), ctx.protocol.Vheight())
	assert.Equal(t, block3.Header(), ctx.bc.GetTopCommittedBlock().Header())
	_ = ctx.bc.GetBlockByHash(block21.Header().Hash())
}

//Scenario 7a: Propose block with transaction
func TestScenario7a(t *testing.T) {
	ctx := initContext(t)

	timeout, f := context.WithTimeout(context.Background(), 10*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hotstuffChan, ctx.epochChan)
	defer f()

	ctx.setMe(1)
	ctx.StartFirstEpoch()
	ctx.generatePayment(ctx.peers[1].GetAddress(), ctx.peers[2].GetAddress(), big.NewInt(100), big.NewInt(1))

	message := <-ctx.proposalChan
	assert.Equal(t, pb.Message_PROPOSAL, message.Type)
	proposal, _ := hotstuff.CreateProposalFromMessage(message)
	assert.Equal(t, 1, proposal.NewBlock().TxsCount())
	assert.Equal(t, "0x573e917b1e96825378ab508d557e0cd8762b80c0b39b242d43cb420457c1df4f", proposal.NewBlock().Header().TxHash().Hex())
	assert.Equal(t, "0x7509e1371b6d9293bac6ed2e32661978a23d9ff4352f661bdfcb31c3370922b5", proposal.NewBlock().Header().StateHash().Hex())

}

//Scenario 7aa: Propose block with transactions
func TestScenario7aa(t *testing.T) {
	ctx := initContext(t)

	timeout, f := context.WithTimeout(context.Background(), 10*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hotstuffChan, ctx.epochChan)
	defer f()

	ctx.setMe(1)
	ctx.StartFirstEpoch()

	ctx.generatePayment(ctx.peers[2].GetAddress(), ctx.peers[3].GetAddress(), big.NewInt(100), big.NewInt(1))
	ctx.generatePayment(ctx.peers[1].GetAddress(), ctx.peers[3].GetAddress(), big.NewInt(100), big.NewInt(1))

	message := <-ctx.proposalChan
	assert.Equal(t, pb.Message_PROPOSAL, message.Type)
	proposal, _ := hotstuff.CreateProposalFromMessage(message)
	assert.Equal(t, "0xd7871f8f064f68ebb2b8e092efcf8212d12c3584c4ab9d91b5b1a47d421bbe8d", proposal.NewBlock().Header().TxHash().Hex())
	assert.Equal(t, "0x51de6155ccb435024a7874b4f933efacad07e49cf0776738d706ff3b52211df7", proposal.NewBlock().Header().StateHash().Hex())
	assert.Equal(t, 2, proposal.NewBlock().TxsCount())

}

//Scenario 7b: Vote for block with transactions
func TestScenario7b(t *testing.T) {
	ctx := initContext(t)

	timeout, f := context.WithTimeout(context.Background(), 10*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hotstuffChan, ctx.epochChan)
	defer f()

	ctx.StartFirstEpoch()
	ctx.setMe(0)

	ctx.generatePayment(ctx.peers[2].GetAddress(), ctx.peers[3].GetAddress(), big.NewInt(100), big.NewInt(1))
	ctx.generatePayment(ctx.peers[1].GetAddress(), ctx.peers[3].GetAddress(), big.NewInt(123), big.NewInt(1))

	block := ctx.bc.NewBlock(ctx.bc.GetHead(), ctx.bc.GetGenesisCert(), []byte("Wunder blok"))
	if e := ctx.stateDB.Release(block.Header().Hash()); e != nil {
		t.Error(e)
	}

	p := ctx.createProposal(block, 1)
	ctx.hotstuffChan <- p

	vote := <-ctx.voteChan

	assert.Equal(t, pb.Message_VOTE, vote.Type)

}

//Scenario 7c: Commit block with transactions
func TestScenario7c(t *testing.T) {
	ctx := initContext(t)

	timeout, f := context.WithTimeout(context.Background(), 10*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hotstuffChan, ctx.epochChan)
	defer f()

	ctx.StartFirstEpoch()
	ctx.setMe(8)

	ctx.generatePayment(ctx.peers[2].GetAddress(), ctx.peers[3].GetAddress(), big.NewInt(100), big.NewInt(1))
	ctx.generatePayment(ctx.peers[1].GetAddress(), ctx.peers[3].GetAddress(), big.NewInt(123), big.NewInt(1))
	block1 := ctx.bc.NewBlock(ctx.bc.GetGenesisBlock(), ctx.bc.GetGenesisCert(), []byte("block 1"))
	qcb1 := ctx.createQC(block1)

	block2 := ctx.bc.NewBlock(block1, qcb1, []byte("block 2"))
	qcb2 := ctx.createQC(block2)

	block3 := ctx.bc.NewBlock(block2, qcb2, []byte("block 3"))
	qc3 := ctx.createQC(block3)
	block4 := ctx.bc.NewBlock(block3, qc3, []byte("block 4"))
	block32 := ctx.bc.NewBlock(block2, qcb2, []byte("block 32"))
	block42 := ctx.bc.NewBlock(block32, qcb2, []byte("block 42"))
	block52 := ctx.bc.NewBlock(block42, qcb2, []byte("block 52"))

	_ = ctx.bc.AddBlock(block1)
	_ = ctx.bc.AddBlock(block2)
	_ = ctx.bc.AddBlock(block3)
	_ = ctx.bc.AddBlock(block4)
	_ = ctx.bc.AddBlock(block32)
	_ = ctx.bc.AddBlock(block42)
	_ = ctx.bc.AddBlock(block52)
	_, _, err := ctx.bc.OnCommit(block1)

	assert.NoError(t, err)
	assert.Equal(t, block3.Signature(), qc3.SignatureAggregate())
}

//Scenario 7d: Forks switching and blocks with transactions (release states etc)
//Test state db cleaning and fork transaction release
func TestScenario7d(t *testing.T) {
	ctx := initContext(t)

	timeout, f := context.WithTimeout(context.Background(), 10*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hotstuffChan, ctx.epochChan)
	defer f()

	ctx.StartFirstEpoch()
	ctx.setMe(8)

	ctx.generatePayment(ctx.peers[2].GetAddress(), ctx.peers[3].GetAddress(), big.NewInt(10), big.NewInt(1))
	ctx.generatePayment(ctx.peers[1].GetAddress(), ctx.peers[3].GetAddress(), big.NewInt(123), big.NewInt(1))
	block1 := ctx.bc.NewBlock(ctx.bc.GetGenesisBlock(), ctx.bc.GetGenesisCert(), []byte("block 1"))
	qcb1 := ctx.createQC(block1)
	_ = ctx.bc.AddBlock(block1)

	block2 := ctx.bc.NewBlock(block1, qcb1, []byte("block 2"))
	qcb2 := ctx.createQC(block2)
	ctx.generatePayment(ctx.peers[2].GetAddress(), ctx.peers[3].GetAddress(), big.NewInt(1), big.NewInt(1))
	_ = ctx.bc.AddBlock(block2)

	block3 := ctx.bc.NewBlock(block2, qcb2, []byte("block 3"))
	ctx.generatePayment(ctx.peers[2].GetAddress(), ctx.peers[3].GetAddress(), big.NewInt(2), big.NewInt(1))
	_ = ctx.bc.AddBlock(block3)

	block4 := ctx.bc.NewBlock(block3, ctx.createQC(block3), []byte("block 4"))
	ctx.generatePayment(ctx.peers[2].GetAddress(), ctx.peers[3].GetAddress(), big.NewInt(3), big.NewInt(1))
	_ = ctx.bc.AddBlock(block4)

	block32 := ctx.bc.NewBlock(block2, qcb2, []byte("block 32"))
	ctx.generatePayment(ctx.peers[2].GetAddress(), ctx.peers[3].GetAddress(), big.NewInt(4), big.NewInt(1))
	_ = ctx.bc.AddBlock(block32)

	block33 := ctx.bc.NewBlock(block2, qcb2, []byte("block 33"))
	ctx.generatePayment(ctx.peers[2].GetAddress(), ctx.peers[3].GetAddress(), big.NewInt(5), big.NewInt(1))
	_ = ctx.bc.AddBlock(block33)

	block42 := ctx.bc.NewBlock(block32, qcb2, []byte("block 42"))
	ctx.generatePayment(ctx.peers[2].GetAddress(), ctx.peers[3].GetAddress(), big.NewInt(6), big.NewInt(1))
	_ = ctx.bc.AddBlock(block42)

	_, ok := ctx.stateDB.Get(block42.Header().Hash())
	assert.True(t, ok)

	_, ok = ctx.stateDB.Get(block3.Header().Hash())
	assert.True(t, ok)

	_, _, err := ctx.bc.OnCommit(block3)
	assert.NoError(t, err)

	_, ok = ctx.stateDB.Get(block42.Header().Hash())
	assert.False(t, ok)

	_, ok = ctx.stateDB.Get(block3.Header().Hash())
	assert.True(t, ok)

}

//Scenario 8a: Settlement
//Send settlement transaction
//Receive it
//Send agreement transaction
//Receive it
func TestScenario8a(t *testing.T) {
	ctx := initContext(t)

	timeout, f := context.WithTimeout(context.Background(), 10*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hotstuffChan, ctx.epochChan)
	go ctx.txService.Run(timeout, ctx.txChan)
	defer f()

	ctx.StartFirstEpoch()

	s := ctx.generateSettlement(ctx.me.GetAddress(), big.NewInt(553), big.NewInt(api.DefaultSettlementReward+3))
	any, _ := ptypes.MarshalAny(s.GetMessage())
	m := msg.CreateMessage(pb.Message_TRANSACTION, any, ctx.me)
	ctx.txChan <- m

	sentTx := <-ctx.sentTxChan
	sentPb := &pb.Transaction{}
	_ = ptypes.UnmarshalAny(sentTx.GetPayload(), sentPb)

	assert.Equal(t, common2.BytesToAddress(s.Hash().Bytes()), common2.BytesToAddress(sentPb.To))

	ctx.txChan <- sentTx

	fromPool := poolToChan(timeout, ctx)

	t1 := <-fromPool
	t2 := <-fromPool

	assert.Equal(t, api.Settlement, t1.TxType())
	assert.Equal(t, api.Agreement, t2.TxType())
}

//Scenario 8b:
//Settler
//Receive 2f + 1 agreements
//Send proof
//Check reward and settle
//S:0x0F1 0    -> 553(settle) + 10(settle_reward) -> 563 -> -10(settle_reward) - 563(settle) -> 0
//0:0xA60 1000 -> 1000(init) - 13(3fee + 10reward) - 2(agreement_fee) -> 985 -> + 553(settle) - 10(proof_fee) + 4(3 change 1 self reward) = 1532
//1:0x54b 2000 -> 7 * 2(agreement_fee) + 3(proposer_reward) -2(agreement_fee) -> 2015 -> + 1(agreement_reward) = 2016
//2:0xCFC 9000 -> - 2(agreement_fee) -> 8998 -> + 10(proof_fee) + 1(agreement_reward) = 9009
//3:0xdA6 4000 -> -2(agreement_fee) -> 3998 -> + 1(agreement_reward) = 3999
//4:0x07E 3500 -> -2(agreement_fee) -> 3498 -> + 1(agreement_reward) = 3499
//5:0xCa2 7500 -> -2(agreement_fee) -> 7498 -> + 1(agreement_reward) = 7499
//6:0x660 9000 -> -2(agreement_fee) -> 8998 -> + 1(agreement_reward) = 8999
//7
//8
//9
func TestScenario8b(t *testing.T) {
	ctx := initContext(t)

	timeout, f := context.WithTimeout(context.Background(), 10*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hotstuffChan, ctx.epochChan)
	go ctx.txService.Run(timeout, ctx.txChan)
	defer f()

	ctx.StartFirstEpoch()

	s := ctx.generateSettlement(ctx.me.GetAddress(), big.NewInt(553), big.NewInt(api.DefaultSettlementReward+3))
	any, _ := ptypes.MarshalAny(s.GetMessage())
	sm := msg.CreateMessage(pb.Message_TRANSACTION, any, ctx.me)
	ctx.txChan <- sm
	var proofs [][]byte
	for i := 0; i < 2*len(ctx.peers)/3+1; i++ {
		nonce := 1
		if bytes.Equal(ctx.peers[i].GetAddress().Bytes(), ctx.me.GetAddress().Bytes()) {
			nonce = 2
		}

		agreement := tx.CreateAgreement(s, uint64(nonce), nil)
		agreement.SetFrom(ctx.peers[i].GetAddress())
		if err := agreement.CreateProof(ctx.peers[i].GetPrivateKey()); err != nil {
			t.Error(err)
		}
		agreement.Sign(ctx.peers[i].GetPrivateKey())
		any, _ := ptypes.MarshalAny(agreement.GetMessage())
		m := msg.CreateMessage(pb.Message_TRANSACTION, any, ctx.me)
		proofs = append(proofs, agreement.Data())
		ctx.txChan <- m
	}

	fromPool := waitPoolTransactions(timeout, ctx, len(ctx.peers)/2+1)
	<-fromPool

	block1 := ctx.bc.NewBlock(ctx.bc.GetGenesisBlock(), ctx.bc.GetGenesisCert(), []byte("block 1"))
	_ = ctx.bc.AddBlock(block1)
	to := common2.BytesToAddress(s.Hash().Bytes()[12:])
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
	spew.Dump(mappedProofs)
	aggregate := crypto.AggregateSignatures(bitmap, signs)
	prAggr := aggregate.ToProto()
	aggrBytes, _ := proto.Marshal(prAggr)

	proofTran := tx.CreateTransaction(api.Proof, to, ctx.me.GetAddress(), 3, big.NewInt(0), big.NewInt(10), aggrBytes)
	proofTran.Sign(ctx.me.GetPrivateKey())
	pany, _ := ptypes.MarshalAny(proofTran.GetMessage())
	pm := msg.CreateMessage(pb.Message_TRANSACTION, pany, ctx.me)
	ctx.txChan <- pm

	fromPool = waitPoolTransactions(timeout, ctx, len(ctx.peers)/2+2)
	<-fromPool

	block2 := ctx.bc.NewBlock(block1, ctx.bc.GetGenesisCert(), []byte("block 2"))
	_ = ctx.bc.AddBlock(block2)

	head := ctx.bc.GetHeadRecord()

	a1, _ := head.Get(ctx.peers[0].GetAddress())
	a2, _ := head.Get(ctx.peers[1].GetAddress())
	a3, _ := head.Get(ctx.peers[2].GetAddress())
	a4, _ := head.Get(ctx.peers[3].GetAddress())
	a5, _ := head.Get(ctx.peers[4].GetAddress())
	a6, _ := head.Get(ctx.peers[5].GetAddress())
	a7, _ := head.Get(ctx.peers[6].GetAddress())
	as, _ := head.Get(common2.HexToAddress("0x0bfa0e22be2feC85Cd04dA94b7b1fd8563F4Cb07"))

	assert.Equal(t, big.NewInt(1532), a1.Balance())
	assert.Equal(t, big.NewInt(2016), a2.Balance())
	assert.Equal(t, big.NewInt(9009), a3.Balance())
	assert.Equal(t, big.NewInt(3999), a4.Balance())
	assert.Equal(t, big.NewInt(3499), a5.Balance())
	assert.Equal(t, big.NewInt(7499), a6.Balance())
	assert.Equal(t, big.NewInt(8999), a7.Balance())
	assert.Equal(t, big.NewInt(0), as.Balance())
}

func poolToChan(timeout context.Context, ctx *TestContext) chan api.Transaction {
	fromPool := make(chan api.Transaction)
	ticker := time.NewTicker(100 * time.Millisecond)
	go func() {
		for {
			select {
			case <-ticker.C:
				it := ctx.pool.Iterator()
				next := it.Next()
				if next != nil {
					ctx.pool.Remove(next)
					fromPool <- next

				}
			case <-timeout.Done():
				close(fromPool)
				return
			}
		}
	}()
	return fromPool
}

func waitPoolTransactions(timeout context.Context, ctx *TestContext, count int) chan struct{} {
	fromPool := make(chan struct{})
	ticker := time.NewTicker(100 * time.Millisecond)
	go func() {
		for {
			select {
			case <-ticker.C:
				it := ctx.pool.Iterator()
				i := 0
				for it.HasNext() {
					it.Next()
					i++
				}
				if i >= count {
					fromPool <- struct{}{}
				}

			case <-timeout.Done():
				close(fromPool)
				return
			}
		}
	}()
	return fromPool
}

func (ctx *TestContext) generateTransaction(from, to common2.Address, amount, fee *big.Int, txType api.Type) api.Transaction {
	trans := tx.CreateTransaction(txType, to, from, 1, amount, fee, []byte(""))
	for _, peer := range ctx.peers {
		if bytes.Equal(peer.GetAddress().Bytes(), from.Bytes()) {
			trans.Sign(peer.GetPrivateKey())
		}
	}
	ctx.pool.Add(trans)

	return trans
}

func (ctx *TestContext) generatePayment(from, to common2.Address, amount, fee *big.Int) api.Transaction {
	return ctx.generateTransaction(from, to, amount, fee, api.Payment)
}

func (ctx *TestContext) generateSettlement(from common2.Address, amount, fee *big.Int) api.Transaction {
	trans := tx.CreateTransaction(api.Settlement, common2.HexToAddress(api.SettlementAddressHex), from, 1, amount, fee, []byte(""))
	for _, peer := range ctx.peers {
		if bytes.Equal(peer.GetAddress().Bytes(), from.Bytes()) {
			trans.Sign(peer.GetPrivateKey())
		}
	}
	return trans
}

type TestContext struct {
	peers        []*common.Peer
	pacer        *hotstuff.StaticPacer
	protocol     *hotstuff.Protocol
	cfg          *hotstuff.ProtocolConfig
	bc           api.Blockchain
	bsrv         blockchain.BlockService
	pool         tx.TransactionPool
	eventChan    chan api.Event
	voteChan     chan *DirectMessage
	startChan    chan *msg.Message
	proposalChan chan *msg.Message
	me           *common.Peer
	hotstuffChan chan *msg.Message
	epochChan    chan *msg.Message
	txChan       chan *msg.Message
	sentTxChan   chan *msg.Message
	headersChan  chan *msg.Message
	blocksToSend map[common2.Hash]*msg.Message
	blocksChan   chan *msg.Message
	seed         map[common2.Address]api.Account
	stateDB      state.DB
	txService    *tx.TxService
}

type DirectMessage struct {
	*msg.Message
	to *common.Peer
}

func (ctx *TestContext) makeVotes(count int, newBlock api.Block) []*msg.Message {
	votes := make([]*msg.Message, count)

	for i := 0; i < count; i++ {
		vote := makeVote(ctx.bc, newBlock, ctx.peers[i])
		vote.Sign(ctx.peers[i].GetPrivateKey())
		any, _ := ptypes.MarshalAny(vote.GetMessage())
		votes[i] = msg.CreateMessage(pb.Message_VOTE, any, ctx.peers[i])
	}

	return votes
}

func makeVote(bc api.Blockchain, newBlock api.Block, peer *common.Peer) api.Vote {
	vote := hotstuff.CreateVote(newBlock.Header(), bc.GetGenesisCert(), peer)
	vote.Sign(peer.GetPrivateKey())
	return vote
}

func (ctx *TestContext) StartFirstEpoch() {
	trigger := make(chan interface{})
	ctx.pacer.SubscribeEpochChange(context.Background(), func(event api.Event) {
		trigger <- event
	})
	ctx.sendStartEpochMessages(0, 2*ctx.cfg.F/3+1, nil)
	<-trigger
}

func (ctx *TestContext) sendMoreStartEpochMessages(index int32, start int, amount int, hqc api.QuorumCertificate) {
	for i := start; i < amount; i++ {
		var epoch *hotstuff.Epoch
		p := ctx.pacer.Committee()[i]
		if hqc == nil {
			hash := ctx.bc.GetGenesisBlockSignedHash(p.GetPrivateKey())
			epoch = hotstuff.CreateEpoch(p, index, nil, hash)
		} else {
			epoch = hotstuff.CreateEpoch(p, index, hqc, crypto.EmptySignature())
		}
		message, _ := epoch.GetMessage()
		ctx.epochChan <- message
	}
}
func (ctx *TestContext) sendStartEpochMessages(index int32, amount int, hqc api.QuorumCertificate) {
	ctx.sendMoreStartEpochMessages(index, 0, amount, hqc)
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

func (ctx *TestContext) createProposal(newBlock api.Block, peerNumber int) *msg.Message {
	proposal := hotstuff.CreateProposal(newBlock, newBlock.QC(), ctx.peers[peerNumber])
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

	return blockchain.CreateQuorumCertificate(aggregate, block.Header())
}

func (ctx *TestContext) SendBlocks(blocks []api.Block) {
	hs := headers(blocks)
	var pbHeaders []*pb.BlockHeader
	for _, h := range hs {
		pbHeaders = append(pbHeaders, h.GetMessage())
	}

	payload := &pb.HeadersResponse{Response: &pb.HeadersResponse_Headers{Headers: &pb.Headers{Headers: pbHeaders}}}
	any, e := ptypes.MarshalAny(payload)
	if e != nil {
		log.Error("can't send blocks")
		return
	}
	ctx.headersChan <- msg.CreateMessage(pb.Message_HEADERS_RESPONSE, any, ctx.me)

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
	srv := &mocks.Service{}
	storage := SoftStorageMock()

	bpersister := &blockchain.BlockPersister{storage}
	cpersister := &blockchain.BlockchainPersister{storage}

	peers := make([]*common.Peer, 10)
	for i := 0; i < 10; i++ {
		peers[i] = generateIdentity(t, i)
	}

	validators := []net.Validator{
		hotstuff.NewEpochStartValidator(peers),
		hotstuff.NewProposalValidator(peers),
		hotstuff.NewVoteValidator(peers),
	}

	pool := tx.NewTransactionPool()
	seed := blockchain.SeedFromFile("../static/seed.json")

	stateDb := state.NewStateDB(storage, &common.NullBus{})

	bc := blockchain.CreateBlockchainFromGenesisBlock(&blockchain.BlockchainConfig{
		Seed:           seed,
		BlockPerister:  bpersister,
		ChainPersister: cpersister,
		Pool:           pool,
		Db:             stateDb,
		Delta:          1 * time.Millisecond, //todo we have a lot of tests where blocks has no transactions, and we create several blocks per round. probably we must add instant block generation without transactions for tests
		Storage:        storage,
	})

	bsrv := blockchain.NewBlockService(srv, MockGoodBlockValidator(), MockGoodHeaderValidator())
	sync := blockchain.CreateSynchronizer(identity, bsrv, bc, -1, 20, 1, 3, -1, 2*10)
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

	startChan := make(chan *msg.Message)
	srv.On("Broadcast", mock.MatchedBy(func(ctx context.Context) bool { return true }), mock.MatchedBy(matcher(pb.Message_EPOCH_START))).Run(func(args mock.Arguments) {
		startChan <- (args[1]).(*msg.Message)
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
	blocksToSend := make(map[common2.Hash]*msg.Message)
	srv.On("SendRequest", mock.MatchedBy(func(ctx context.Context) bool { return true }), mock.AnythingOfType("*common.Peer"),
		mock.MatchedBy(matcher(pb.Message_HEADERS_REQUEST))).Return(headersChan, nil)
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
	epochChan := make(chan *msg.Message)
	txChan := make(chan *msg.Message)
	return &TestContext{
		voteChan:     voteChan,
		peers:        peers,
		pacer:        pacer,
		protocol:     p,
		bc:           bc,
		cfg:          config,
		seed:         seed,
		proposalChan: proposalChan,
		blocksChan:   blocksChan,
		blocksToSend: blocksToSend,
		headersChan:  headersChan,
		startChan:    startChan,
		eventChan:    eventChan,
		me:           identity,
		bsrv:         bsrv,
		pool:         pool,
		hotstuffChan: hottuffChan,
		epochChan:    epochChan,
		stateDB:      stateDb,
		txService:    txService,
		txChan:       txChan,
		sentTxChan:   sentTxChan,
	}
}

func headers(blocks []api.Block) (headers []api.Header) {
	for _, b := range blocks {
		headers = append(headers, b.Header())
	}

	return headers
}
