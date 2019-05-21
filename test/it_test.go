package test

import (
	"bytes"
	"context"
	"github.com/golang/protobuf/ptypes"
	"github.com/poslibp2p"
	"github.com/poslibp2p/blockchain"
	"github.com/poslibp2p/blockchain/state"
	"github.com/poslibp2p/common"
	common2 "github.com/poslibp2p/common/eth/common"
	msg "github.com/poslibp2p/common/message"
	"github.com/poslibp2p/common/protobuff"
	"github.com/poslibp2p/common/tx"
	"github.com/poslibp2p/hotstuff"
	"github.com/poslibp2p/mocks"
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
	go ctx.pacer.Run(timeout, ctx.hottuffChan, ctx.epochChan)
	defer f()

	ctx.StartFirstEpoch()
	ctx.setMe(2)

	newBlock := ctx.bc.NewBlock(ctx.bc.GetHead(), ctx.bc.GetGenesisCert(), []byte("wonderful block"))

	p := ctx.createProposal(newBlock, 1)
	ctx.hottuffChan <- p

	votes := ctx.makeVotes(2*ctx.cfg.F/3+1, newBlock)
	for _, v := range votes {
		ctx.hottuffChan <- v
	}

	proposal := <-ctx.proposalCHan

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
	go ctx.pacer.Run(timeout, ctx.hottuffChan, ctx.epochChan)
	defer f()

	ctx.StartFirstEpoch()
	ctx.setMe(2)

	newBlock := ctx.bc.NewBlock(ctx.bc.GetHead(), ctx.bc.GetGenesisCert(), []byte("wonderful block"))

	votes := ctx.makeVotes(2*ctx.cfg.F/3+1, newBlock)
	for _, v := range votes[:2*ctx.cfg.F/3-2] {
		ctx.hottuffChan <- v
	}

	p := ctx.createProposal(newBlock, 1)
	ctx.hottuffChan <- p

	for _, v := range votes[2*ctx.cfg.F/3-2:] {
		ctx.hottuffChan <- v
	}

	proposal := <-ctx.proposalCHan

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
	go ctx.pacer.Run(timeout, ctx.hottuffChan, ctx.epochChan)
	defer f()

	defer f()
	//defer ctx.protocol.Stop()

	ctx.StartFirstEpoch()
	ctx.setMe(2)

	newBlock := ctx.bc.NewBlock(ctx.bc.GetHead(), ctx.bc.GetGenesisCert(), []byte("wonderful block"))

	votes := ctx.makeVotes(2*ctx.cfg.F/3+1, newBlock)
	for _, v := range votes {
		ctx.hottuffChan <- v
	}

	go func() {
		ctx.blockChan <- newBlock
		close(ctx.blockChan)
		ctx.blockChan = make(chan *blockchain.Block)
	}()

	p := ctx.createProposal(newBlock, 1)
	ctx.hottuffChan <- p

	proposal := <-ctx.proposalCHan

	payload := &pb.ProposalPayload{}
	if err := ptypes.UnmarshalAny(proposal.Payload, payload); err != nil {
		log.Error(err)
	}

	ctx.waitRounds(timeout, 2)

	assert.Equal(t, int32(2), payload.Block.GetHeader().GetHeight())
	assert.Equal(t, int32(1), payload.Block.GetCert().GetHeader().GetHeight())
}

//Scenario 2:
//Start new epoch
//Replica
//Get proposal
//Vote
func TestScenario2(t *testing.T) {
	ctx := initContext(t)
	timeout, f := context.WithTimeout(context.Background(), 10*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hottuffChan, ctx.epochChan)
	defer f()

	ctx.StartFirstEpoch()
	ctx.setMe(0)

	newBlock := ctx.bc.NewBlock(ctx.bc.GetHead(), ctx.bc.GetGenesisCert(), []byte("wonderful block"))
	p := ctx.createProposal(newBlock, 1)
	ctx.hottuffChan <- p

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
	go ctx.pacer.Run(timeout, ctx.hottuffChan, ctx.epochChan)
	defer f()

	trigger := make(chan interface{})
	ctx.pacer.SubscribeEpochChange(timeout, trigger)
	ctx.sendStartEpochMessages(2, 2*ctx.cfg.F/3+1, nil)
	<-trigger

	assert.Equal(t, int32(21), ctx.pacer.GetCurrentView())

}

//Scenario 3:
//Start new epoch
//Proposer
//Propose block with previous QC after Delta
func TestScenario3(t *testing.T) {
	ctx := initContext(t)

	timeout, f := context.WithTimeout(context.Background(), 10*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hottuffChan, ctx.epochChan)
	defer f()

	ctx.StartFirstEpoch()
	ctx.setMe(2)

	newBlock := ctx.bc.NewBlock(ctx.bc.GetHead(), ctx.bc.GetGenesisCert(), []byte("wonderful block"))
	p := ctx.createProposal(newBlock, 1)
	ctx.hottuffChan <- p
	<-ctx.voteChan

	proposal := <-ctx.proposalCHan

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
	go ctx.pacer.Run(timeout, ctx.hottuffChan, ctx.epochChan)
	defer f()

	ctx.StartFirstEpoch()
	ctx.setMe(2)

	proposal := <-ctx.proposalCHan

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
	go ctx.pacer.Run(timeout, ctx.hottuffChan, ctx.epochChan)
	defer f()

	ctx.StartFirstEpoch()

	ctx.setMe(1)

	log.Infof("me %v", ctx.pacer.Committee()[0].GetAddress().Hex())

	proposal := <-ctx.proposalCHan
	time.Sleep(8 * ctx.cfg.Delta)
	ctx.sendStartEpochMessages(2, 2*ctx.cfg.F/3+1, ctx.protocol.HQC())
	<-ctx.startChan //mine start message

	proposal = <-ctx.proposalCHan

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
	go ctx.pacer.Run(timeout, ctx.hottuffChan, ctx.epochChan)
	defer f()

	ctx.StartFirstEpoch()
	ctx.setMe(1)

	log.Infof("me %v", ctx.pacer.Committee()[0].GetAddress().Hex())

	<-ctx.startChan
	ctx.sendStartEpochMessages(1, 2*ctx.cfg.F/3+1, ctx.protocol.HQC())

	proposal := <-ctx.proposalCHan

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
	go ctx.pacer.Run(timeout, ctx.hottuffChan, ctx.epochChan)
	defer f()

	ctx.StartFirstEpoch()
	ctx.setMe(1)

	log.Infof("me %v", ctx.pacer.Committee()[0].GetAddress().Hex())
	proposal := <-ctx.proposalCHan

	time.Sleep(26 * ctx.cfg.Delta)
	ctx.sendStartEpochMessages(2, 2*ctx.cfg.F/3+1, ctx.protocol.HQC())
	<-ctx.startChan //mine start message

	proposal = <-ctx.proposalCHan

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
	go ctx.pacer.Run(timeout, ctx.hottuffChan, ctx.epochChan)
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
	go ctx.pacer.Run(timeout, ctx.hottuffChan, ctx.epochChan)
	defer f()

	ctx.StartFirstEpoch()
	ctx.setMe(2)

	log.Infof("me %v", ctx.pacer.Committee()[2].GetAddress().Hex())

	ctx.sendStartEpochMessages(2, ctx.cfg.F/3+1, ctx.protocol.HQC())
	<-ctx.startChan //mine start message

	block := ctx.bc.GetHead()
	for i := 0; i < 20; i++ {
		block = ctx.bc.PadEmptyBlock(block)
	}
	newBlock := ctx.bc.NewBlock(block, ctx.bc.GetGenesisCert(), []byte("wonderful block"))
	p := ctx.createProposal(newBlock, 1)

	go func() {
		ctx.hottuffChan <- p
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
	go ctx.pacer.Run(timeout, ctx.hottuffChan, ctx.epochChan)
	defer f()

	ctx.StartFirstEpoch()
	ctx.setMe(8)

	block1 := ctx.bc.NewBlock(ctx.bc.GetGenesisBlock(), ctx.bc.GetGenesisCert(), []byte("block 1"))
	proposal1 := ctx.createProposal(block1, 1)
	ctx.hottuffChan <- proposal1
	block2 := ctx.bc.NewBlock(block1, ctx.bc.GetGenesisCert(), []byte("block 2"))
	proposal2 := ctx.createProposal(block2, 2)
	ctx.hottuffChan <- proposal2
	block3 := ctx.bc.NewBlock(block2, ctx.bc.GetGenesisCert(), []byte("block 3"))
	proposal3 := ctx.createProposal(block3, 3)
	ctx.hottuffChan <- proposal3

	block21 := ctx.bc.NewBlock(block1, ctx.bc.GetGenesisCert(), []byte("block 21"))
	block31 := ctx.bc.NewBlock(block21, ctx.bc.GetGenesisCert(), []byte("block 31"))
	block4 := ctx.bc.NewBlock(block31, ctx.bc.GetGenesisCert(), []byte("block 41"))
	proposal := ctx.createProposal(block4, 4)
	ctx.hottuffChan <- proposal

	ctx.blockChan <- block4
	ctx.blockChan <- block31
	ctx.blockChan <- block2
	close(ctx.blockChan)

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
	go ctx.pacer.Run(timeout, ctx.hottuffChan, ctx.epochChan)
	defer f()

	ctx.StartFirstEpoch()
	ctx.setMe(8)

	block1 := ctx.bc.NewBlock(ctx.bc.GetGenesisBlock(), ctx.bc.GetGenesisCert(), []byte("block 1"))
	proposal1 := ctx.createProposal(block1, 1)
	ctx.hottuffChan <- proposal1
	qcb1 := ctx.createQC(block1)
	block2 := ctx.bc.NewBlock(block1, qcb1, []byte("block 2"))
	proposal2 := ctx.createProposal(block2, 2)
	ctx.hottuffChan <- proposal2
	block3 := ctx.bc.NewBlock(block2, ctx.createQC(block2), []byte("block 3"))
	proposal3 := ctx.createProposal(block3, 3)
	ctx.hottuffChan <- proposal3
	block4 := ctx.bc.NewBlock(block3, ctx.createQC(block3), []byte("block 4"))
	proposal4 := ctx.createProposal(block4, 4)
	ctx.hottuffChan <- proposal4

	block22 := ctx.bc.NewBlock(block1, qcb1, []byte("block 22"))
	block32 := ctx.bc.NewBlock(block22, qcb1, []byte("block 32"))
	block42 := ctx.bc.NewBlock(block32, qcb1, []byte("block 42"))
	block52 := ctx.bc.NewBlock(block42, qcb1, []byte("block 52"))
	proposal := ctx.createProposal(block52, 5)
	ctx.hottuffChan <- proposal

	ctx.blockChan <- block22
	ctx.blockChan <- block32
	ctx.blockChan <- block42
	close(ctx.blockChan)

	ctx.waitRounds(timeout, 6)

	assert.Equal(t, int32(4), ctx.protocol.Vheight())
	assert.Equal(t, block1, ctx.bc.GetTopCommittedBlock())
	assert.Equal(t, block52, ctx.bc.GetBlockByHeight(5)[0])
}

//Scenario 6c:
//Start new epoch
//Replica
//Receive fork Gc<-B1c<-B2<-B3<-B4
//Receive proposal fork B2<-B32<-(B2)B42-B52 (vote for b52 as it extends QREF)
func TestScenario6c(t *testing.T) {
	ctx := initContext(t)

	timeout, f := context.WithTimeout(context.Background(), 10*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hottuffChan, ctx.epochChan)
	defer f()

	ctx.StartFirstEpoch()
	ctx.setMe(8)

	block1 := ctx.bc.NewBlock(ctx.bc.GetGenesisBlock(), ctx.bc.GetGenesisCert(), []byte("block 1"))
	proposal1 := ctx.createProposal(block1, 1)
	ctx.hottuffChan <- proposal1
	qcb1 := ctx.createQC(block1)
	block2 := ctx.bc.NewBlock(block1, qcb1, []byte("block 2"))
	proposal2 := ctx.createProposal(block2, 2)
	ctx.hottuffChan <- proposal2
	qcb2 := ctx.createQC(block2)
	block3 := ctx.bc.NewBlock(block2, qcb2, []byte("block 3"))
	proposal3 := ctx.createProposal(block3, 3)
	ctx.hottuffChan <- proposal3
	block4 := ctx.bc.NewBlock(block3, ctx.createQC(block3), []byte("block 4"))
	proposal4 := ctx.createProposal(block4, 4)
	ctx.hottuffChan <- proposal4

	block32 := ctx.bc.NewBlock(block2, qcb2, []byte("block 32"))
	block42 := ctx.bc.NewBlock(block32, qcb2, []byte("block 42"))
	block52 := ctx.bc.NewBlock(block42, qcb2, []byte("block 52"))
	proposal := ctx.createProposal(block52, 5)
	ctx.hottuffChan <- proposal

	ctx.blockChan <- block2
	ctx.blockChan <- block32
	ctx.blockChan <- block42
	close(ctx.blockChan)

	ctx.waitRounds(timeout, 6)

	assert.Equal(t, int32(5), ctx.protocol.Vheight())
	assert.Equal(t, block1, ctx.bc.GetTopCommittedBlock())
	assert.Equal(t, block52, ctx.bc.GetBlockByHeight(5)[0])
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
	go ctx.pacer.Run(timeout, ctx.hottuffChan, ctx.epochChan)
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
	//ctx.cfg.ControlChan <- *hotstuff.NewCommand(context.Background(), hotstuff.NextView)
	//ctx.cfg.ControlChan <- *hotstuff.NewCommand(context.Background(), hotstuff.NextView)
	//ctx.cfg.ControlChan <- *hotstuff.NewCommand(context.Background(), hotstuff.NextView)
	//ctx.cfg.ControlChan <- *hotstuff.NewCommand(context.Background(), hotstuff.NextView)
	//ctx.cfg.ControlChan <- *hotstuff.NewCommand(context.Background(), hotstuff.NextView)

	qcb4 := ctx.createQC(block4)
	block51 := ctx.bc.NewBlock(block4, qcb4, []byte("block 51"))
	block6 := ctx.bc.NewBlock(block51, qcb4, []byte("block 6"))
	proposal6 := ctx.createProposal(block6, 6)
	ctx.hottuffChan <- proposal6

	ctx.blockChan <- block2
	ctx.blockChan <- block3
	ctx.blockChan <- block4
	ctx.blockChan <- block51
	close(ctx.blockChan)

	ctx.waitRounds(timeout, 7)

	assert.Equal(t, block2, ctx.bc.GetTopCommittedBlock())
	assert.Equal(t, int32(6), ctx.protocol.Vheight())
	assert.Equal(t, block6, ctx.bc.GetBlockByHeight(6)[0])
}

//Scenario 7a: Propose block with transaction
func TestScenario7a(t *testing.T) {
	ctx := initContext(t)

	timeout, f := context.WithTimeout(context.Background(), 10*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hottuffChan, ctx.epochChan)
	defer f()

	ctx.setMe(1)
	ctx.StartFirstEpoch()
	ctx.generateTransaction(ctx.peers[1].GetAddress(), ctx.peers[2].GetAddress(), big.NewInt(100), big.NewInt(1))

	message := <-ctx.proposalCHan
	assert.Equal(t, pb.Message_PROPOSAL, message.Type)
	proposal, _ := hotstuff.CreateProposalFromMessage(message)
	assert.Equal(t, "0x7304f75a7c5e55bd0a6a5bc09f273fb56b080a89561a9a73e0c5d6e8dd2453b6", proposal.NewBlock.Header().TxHash().Hex())
	assert.Equal(t, "0xd494fdaa2e07ae7779a9de9ccc0f1ac8be8cfaca0e92a8639469881a2569e065", proposal.NewBlock.Header().StateHash().Hex())
	assert.Equal(t, 1, proposal.NewBlock.TxsCount())

} //Scenario 7aa: Propose block with transactions
func TestScenario7aa(t *testing.T) {
	ctx := initContext(t)

	timeout, f := context.WithTimeout(context.Background(), 10*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout, ctx.hottuffChan, ctx.epochChan)
	defer f()

	ctx.setMe(1)
	ctx.StartFirstEpoch()

	ctx.generateTransaction(ctx.peers[2].GetAddress(), ctx.peers[3].GetAddress(), big.NewInt(100), big.NewInt(1))
	ctx.generateTransaction(ctx.peers[1].GetAddress(), ctx.peers[3].GetAddress(), big.NewInt(100), big.NewInt(1))

	message := <-ctx.proposalCHan
	assert.Equal(t, pb.Message_PROPOSAL, message.Type)
	proposal, _ := hotstuff.CreateProposalFromMessage(message)
	assert.Equal(t, "0xbe85ed143375fb46f381c04332e3fe3bb50cec1a56fb69651e81d2e1d46e3c93", proposal.NewBlock.Header().TxHash().Hex())
	assert.Equal(t, "0xb4cbc1b6e4fdf47c3934fbd404fe3a99e9fc5d1ce04aae1c68b0eceb6d287945", proposal.NewBlock.Header().StateHash().Hex())
	assert.Equal(t, 2, proposal.NewBlock.TxsCount())

}

//Scenario 7b: Vote for block with transactions
func TestScenario7b(t *testing.T) {

}

//Scenario 7c: Commit block with transactions
func TestScenario7c(t *testing.T) {

}

//Scenario 7d: Forks and blocks with transactions (release states etc)
func TestScenario7d(t *testing.T) {

}

func (ctx *TestContext) generateTransaction(from, to common2.Address, amount, fee *big.Int) {
	trans := tx.CreateTransaction(tx.Payment, to, from, 1, amount, fee, []byte(""))
	for _, peer := range ctx.peers {
		if bytes.Equal(peer.GetAddress().Bytes(), from.Bytes()) {
			trans.Sign(peer.GetPrivateKey())
		}
	}
	ctx.pool.Add(trans)
}

type TestContext struct {
	peers        []*common.Peer
	pacer        *hotstuff.StaticPacer
	protocol     *hotstuff.Protocol
	cfg          *hotstuff.ProtocolConfig
	bc           *blockchain.Blockchain
	bsrv         blockchain.BlockService
	pool         blockchain.TransactionPool
	eventChan    chan hotstuff.Event
	voteChan     chan *msg.Message
	startChan    chan *msg.Message
	proposalCHan chan *msg.Message
	me           *common.Peer
	hottuffChan  chan *msg.Message
	epochChan    chan *msg.Message
	blockChan    chan *blockchain.Block
	seed         map[common2.Address]*state.Account
}

func (ctx *TestContext) makeVotes(count int, newBlock *blockchain.Block) []*msg.Message {
	votes := make([]*msg.Message, count)

	for i := 0; i < count; i++ {
		vote := makeVote(ctx.bc, newBlock, ctx.peers[i])
		any, _ := ptypes.MarshalAny(vote.GetMessage())
		votes[i] = msg.CreateMessage(pb.Message_VOTE, any, ctx.peers[i])
	}

	return votes
}

func makeVote(bc *blockchain.Blockchain, newBlock *blockchain.Block, peer *common.Peer) *hotstuff.Vote {
	vote := hotstuff.CreateVote(newBlock.Header(), bc.GetGenesisCert(), peer)
	vote.Sign(peer.GetPrivateKey())
	return vote
}

func (ctx *TestContext) StartFirstEpoch() {
	trigger := make(chan interface{})
	ctx.pacer.SubscribeEpochChange(context.Background(), trigger)
	ctx.sendStartEpochMessages(0, 2*ctx.cfg.F/3+1, nil)
	<-trigger
}

func (ctx *TestContext) sendMoreStartEpochMessages(index int32, start int, amount int, hqc *blockchain.QuorumCertificate) {
	for i := start; i < amount; i++ {
		var epoch *hotstuff.Epoch
		p := ctx.pacer.Committee()[i]
		if hqc == nil {
			hash := ctx.bc.GetGenesisBlockSignedHash(p.GetPrivateKey())
			peer := common.CreatePeer(nil, p.GetPrivateKey(), p.GetPeerInfo())
			epoch = hotstuff.CreateEpoch(peer, index, nil, hash)
		} else {
			peer := common.CreatePeer(nil, p.GetPrivateKey(), p.GetPeerInfo())
			epoch = hotstuff.CreateEpoch(peer, index, hqc, nil)
		}
		message, _ := epoch.GetMessage()
		ctx.epochChan <- message
	}
}
func (ctx *TestContext) sendStartEpochMessages(index int32, amount int, hqc *blockchain.QuorumCertificate) {
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
			if event == hotstuff.ChangedView {
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

func (ctx *TestContext) createProposal(newBlock *blockchain.Block, peerNumber int) *msg.Message {
	proposal := hotstuff.CreateProposal(newBlock, newBlock.QC(), ctx.peers[peerNumber])
	proposal.Sign(ctx.peers[peerNumber].GetPrivateKey())
	any, _ := ptypes.MarshalAny(proposal.GetMessage())
	return msg.CreateMessage(pb.Message_PROPOSAL, any, ctx.peers[peerNumber])
}

func (ctx *TestContext) createQC(block *blockchain.Block) *blockchain.QuorumCertificate {
	var sign []byte
	for i := 0; i < 2*ctx.cfg.F/3+1; i++ {
		sign = append(sign, block.Header().Sign(ctx.peers[i].GetPrivateKey())...)
	}

	return blockchain.CreateQuorumCertificate(sign, block.Header())
}

func initContext(t *testing.T) *TestContext {
	identity := generateIdentity(t, 0)
	log.Debugf("Current me %v", identity.GetAddress().Hex())
	srv := &mocks.Service{}
	bsrv := &mocks.BlockService{}
	storage := &mocks.Storage{}
	storage.On("PutBlock", mock.AnythingOfType("*blockchain.Block")).Return(nil)
	storage.On("GetBlock", mock.AnythingOfType("common.Hash")).Return(nil, nil)
	storage.On("Contains", mock.AnythingOfType("common.Hash")).Return(false)
	storage.On("PutCurrentTopHeight", mock.AnythingOfType("int32")).Return(nil)
	storage.On("PutCurrentEpoch", mock.AnythingOfType("int32")).Return(nil)
	storage.On("GetCurrentEpoch").Return(int32(-1), nil)
	storage.On("GetTopCommittedHeight").Return(0)
	storage.On("GetCurrentTopHeight").Return(int32(0), nil)
	storage.On("PutTopCommittedHeight", mock.AnythingOfType("int32")).Return(nil)

	peers := make([]*common.Peer, 10)
	for i := 0; i < 10; i++ {
		peers[i] = generateIdentity(t, i)
	}

	validators := []poslibp2p.Validator{
		hotstuff.NewEpochStartValidator(peers),
		hotstuff.NewProposalValidator(peers),
		hotstuff.NewVoteValidator(peers),
	}

	pool := blockchain.NewTransactionPool()
	seed := blockchain.SeedFromFile("../static/seed.json")
	bc := blockchain.CreateBlockchainFromGenesisBlock(&blockchain.Config{
		Seed:         seed,
		Storage:      storage,
		BlockService: bsrv,
		Pool:         pool,
		Db:           state.NewStateDB(),
	})

	sync := blockchain.CreateSynchronizer(identity, bsrv, bc)
	config := &hotstuff.ProtocolConfig{
		F:          10,
		Delta:      1 * time.Second,
		Blockchain: bc,
		Me:         identity,
		Srv:        srv,
		Storage:    storage,
		Sync:       sync,
		Validators: validators,
		Committee:  peers,
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

	voteChan := make(chan *msg.Message)
	srv.On("SendMessage", mock.MatchedBy(func(ctx context.Context) bool { return true }), mock.AnythingOfType("*common.Peer"), mock.MatchedBy(matcher(pb.Message_VOTE))).
		Return(make(chan *msg.Message), nil).
		Run(func(args mock.Arguments) {
			voteChan <- (args[2]).(*msg.Message)
		})

	blockChan := make(chan *blockchain.Block)
	bsrv.On("RequestBlock", mock.MatchedBy(func(ctx context.Context) bool { return true }), mock.AnythingOfType("common.Hash"),
		mock.AnythingOfType("*common.Peer")).Return(blockChan, nil)
	bsrv.On("RequestFork", mock.MatchedBy(func(ctx context.Context) bool { return true }), mock.AnythingOfType("int32"), mock.AnythingOfType("common.Hash"),
		mock.AnythingOfType("*common.Peer")).Return(blockChan, nil)

	pacer := hotstuff.CreatePacer(config)
	config.Pacer = pacer
	p := hotstuff.CreateProtocol(config)
	pacer.Bootstrap(context.Background(), p)

	eventChan := make(chan hotstuff.Event)
	pacer.SubscribeProtocolEvents(eventChan)

	hottuffChan := make(chan *msg.Message)
	epochChan := make(chan *msg.Message)
	return &TestContext{
		voteChan:     voteChan,
		peers:        peers,
		pacer:        pacer,
		protocol:     p,
		bc:           bc,
		cfg:          config,
		seed:         seed,
		proposalCHan: proposalChan,
		blockChan:    blockChan,
		startChan:    startChan,
		eventChan:    eventChan,
		me:           identity,
		bsrv:         bsrv,
		pool:         pool,
		hottuffChan:  hottuffChan,
		epochChan:    epochChan,
	}
}
