package test

import (
	"context"
	"github.com/golang/protobuf/ptypes"
	"github.com/poslibp2p/blockchain"
	"github.com/poslibp2p/common"
	"github.com/poslibp2p/common/eth/crypto"
	msg "github.com/poslibp2p/common/message"
	"github.com/poslibp2p/common/protobuff"
	"github.com/poslibp2p/hotstuff"
	"github.com/poslibp2p/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
	timeout, _ := context.WithTimeout(context.Background(), 10*ctx.cfg.Delta)
	go ctx.pacer.Run(timeout)
	go ctx.protocol.Run(ctx.protocolChan)
	defer ctx.pacer.Stop()
	defer ctx.protocol.Stop()

	ctx.StartFirstEpoch()
	ctx.setMe(2)

	newBlock := ctx.bc.NewBlock(ctx.bc.GetHead(), ctx.bc.GetGenesisCert(), []byte("wonderful block"))

	p := ctx.createProposal(newBlock, 1)
	ctx.protocolChan <- p

	votes := ctx.makeVotes(2*ctx.cfg.F/3+1, newBlock)
	for _, v := range votes {
		ctx.protocolChan <- v
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
	go ctx.pacer.Run(context.Background())
	go ctx.protocol.Run(ctx.protocolChan)

	defer ctx.pacer.Stop()
	defer ctx.protocol.Stop()

	ctx.StartFirstEpoch()
	ctx.setMe(2)

	newBlock := ctx.bc.NewBlock(ctx.bc.GetHead(), ctx.bc.GetGenesisCert(), []byte("wonderful block"))

	votes := ctx.makeVotes(2*ctx.cfg.F/3+1, newBlock)
	for _, v := range votes[:2*ctx.cfg.F/3-2] {
		ctx.protocolChan <- v
	}

	p := ctx.createProposal(newBlock, 1)
	ctx.protocolChan <- p

	for _, v := range votes[2*ctx.cfg.F/3-2:] {
		ctx.protocolChan <- v
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

	go ctx.pacer.Run(context.Background())
	go ctx.protocol.Run(ctx.protocolChan)

	defer ctx.pacer.Stop()
	//todo find out why we hangs here
	//defer ctx.protocol.Stop()

	ctx.StartFirstEpoch()
	ctx.setMe(2)

	newBlock := ctx.bc.NewBlock(ctx.bc.GetHead(), ctx.bc.GetGenesisCert(), []byte("wonderful block"))

	votes := ctx.makeVotes(2*ctx.cfg.F/3+1, newBlock)
	for _, v := range votes {
		ctx.protocolChan <- v
	}

	go func() {
		ctx.blockChan <- newBlock
		close(ctx.blockChan)
		ctx.blockChan = make(chan *blockchain.Block)
	}()

	p := ctx.createProposal(newBlock, 1)
	ctx.protocolChan <- p

	proposal := <-ctx.proposalCHan

	payload := &pb.ProposalPayload{}
	if err := ptypes.UnmarshalAny(proposal.Payload, payload); err != nil {
		log.Error(err)
	}

	ctx.waitRounds(3)

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
	go ctx.pacer.Run(context.Background())
	go ctx.protocol.Run(ctx.protocolChan)

	defer ctx.pacer.Stop()
	defer ctx.protocol.Stop()

	ctx.StartFirstEpoch()
	ctx.setMe(0)

	newBlock := ctx.bc.NewBlock(ctx.bc.GetHead(), ctx.bc.GetGenesisCert(), []byte("wonderful block"))
	p := ctx.createProposal(newBlock, 1)
	ctx.protocolChan <- p

	vote := <-ctx.voteChan

	assert.Equal(t, pb.Message_VOTE, vote.Type)

}

//Scenario 3:
//Start new epoch
//Proposer
//Propose block with previous QC after Delta
func TestScenario3(t *testing.T) {
	ctx := initContext(t)

	go ctx.pacer.Run(context.Background())
	go ctx.protocol.Run(ctx.protocolChan)

	defer ctx.pacer.Stop()
	defer ctx.protocol.Stop()

	ctx.StartFirstEpoch()
	ctx.setMe(2)

	newBlock := ctx.bc.NewBlock(ctx.bc.GetHead(), ctx.bc.GetGenesisCert(), []byte("wonderful block"))
	p := ctx.createProposal(newBlock, 1)
	ctx.protocolChan <- p
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
	go ctx.pacer.Run(context.Background())
	go ctx.protocol.Run(ctx.protocolChan)

	defer ctx.pacer.Stop()
	defer ctx.protocol.Stop()
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
	go ctx.pacer.Run(context.Background())
	go ctx.protocol.Run(ctx.protocolChan)

	defer ctx.pacer.Stop()
	defer ctx.protocol.Stop()

	ctx.StartFirstEpoch()
	ctx.setMe(0)

	log.Infof("me %v", ctx.pacer.Committee()[0].GetAddress().Hex())

	time.Sleep(8 * ctx.cfg.Delta)
	ctx.sendStartEpochMessages(2, 2*ctx.cfg.F/3+1, ctx.protocol.HQC())
	<-ctx.startChan //mine start message

	proposal := <-ctx.proposalCHan

	payload := &pb.ProposalPayload{}
	if err := ptypes.UnmarshalAny(proposal.Payload, payload); err != nil {
		log.Error(err)
	}

	assert.Equal(t, int32(10), payload.Block.GetHeader().GetHeight())
	assert.Equal(t, int32(0), payload.Block.GetCert().GetHeader().GetHeight())
}

//Scenario 5b:
//Start new epoch
//Receive no messages
//Start new epoch after epoch end
//Propose
func TestScenario5b(t *testing.T) {
	ctx := initContext(t)
	go ctx.pacer.Run(context.Background())
	go ctx.protocol.Run(ctx.protocolChan)

	defer ctx.pacer.Stop()
	defer ctx.protocol.Stop()
	ctx.StartFirstEpoch()
	ctx.setMe(0)

	log.Infof("me %v", ctx.pacer.Committee()[0].GetAddress().Hex())

	<-ctx.startChan
	ctx.sendStartEpochMessages(2, 2*ctx.cfg.F/3+1, ctx.protocol.HQC())

	proposal := <-ctx.proposalCHan

	payload := &pb.ProposalPayload{}
	if err := ptypes.UnmarshalAny(proposal.Payload, payload); err != nil {
		log.Error(err)
	}

	assert.Equal(t, int32(10), payload.Block.GetHeader().GetHeight())
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
	go ctx.pacer.Run(context.Background())
	go ctx.protocol.Run(ctx.protocolChan)

	defer ctx.pacer.Stop()
	defer ctx.protocol.Stop()

	ctx.StartFirstEpoch()
	ctx.setMe(0)

	log.Infof("me %v", ctx.pacer.Committee()[0].GetAddress().Hex())

	time.Sleep(26 * ctx.cfg.Delta)
	ctx.sendStartEpochMessages(2, 2*ctx.cfg.F/3+1, ctx.protocol.HQC())
	<-ctx.startChan //mine start message

	proposal := <-ctx.proposalCHan

	payload := &pb.ProposalPayload{}
	if err := ptypes.UnmarshalAny(proposal.Payload, payload); err != nil {
		log.Error(err)
	}

	assert.Equal(t, int32(10), payload.Block.GetHeader().GetHeight())
	assert.Equal(t, int32(0), payload.Block.GetCert().GetHeader().GetHeight())
}

//Scenario 6a:
//Start new epoch
//Replica
//Receive proposal fork
//Get no block on arbitrary height
//Reject proposal and all fork
func TestScenario6a(t *testing.T) {
	ctx := initContext(t)
	go ctx.pacer.Run(context.Background())
	go ctx.protocol.Run(ctx.protocolChan)

	defer ctx.pacer.Stop()
	defer func() {
		go ctx.protocol.Stop()
	}()

	ctx.StartFirstEpoch()
	ctx.setMe(8)

	block1 := ctx.bc.NewBlock(ctx.bc.GetGenesisBlock(), ctx.bc.GetGenesisCert(), []byte("block 1"))
	proposal1 := ctx.createProposal(block1, 1)
	ctx.protocolChan <- proposal1
	block2 := ctx.bc.NewBlock(block1, ctx.bc.GetGenesisCert(), []byte("block 2"))
	proposal2 := ctx.createProposal(block2, 2)
	ctx.protocolChan <- proposal2
	block3 := ctx.bc.NewBlock(block2, ctx.bc.GetGenesisCert(), []byte("block 3"))
	proposal3 := ctx.createProposal(block3, 3)
	ctx.protocolChan <- proposal3

	block21 := ctx.bc.NewBlock(block1, ctx.bc.GetGenesisCert(), []byte("block 21"))
	block31 := ctx.bc.NewBlock(block21, ctx.bc.GetGenesisCert(), []byte("block 31"))
	block4 := ctx.bc.NewBlock(block31, ctx.bc.GetGenesisCert(), []byte("block 41"))
	proposal := ctx.createProposal(block4, 4)
	ctx.protocolChan <- proposal

	ctx.blockChan <- block4
	ctx.blockChan <- block31
	ctx.blockChan <- block2
	close(ctx.blockChan)

	ctx.waitRounds(5)

	assert.Equal(t, ctx.bc.GetHead().Header().Height(), int32(3))
}

//Scenario 6b:
//Start new epoch
//Replica
//Receive fork Gc<-B1c<-B2<-B3<-B4
//Receive proposal fork B1c-B22-B32-B42-B52 (reject cause don't extend QREF())
func TestScenario6b(t *testing.T) {
	ctx := initContext(t)
	go ctx.pacer.Run(context.Background())
	go ctx.protocol.Run(ctx.protocolChan)

	defer ctx.pacer.Stop()
	defer func() {
		go ctx.protocol.Stop()
	}()

	ctx.StartFirstEpoch()
	ctx.setMe(8)

	block1 := ctx.bc.NewBlock(ctx.bc.GetGenesisBlock(), ctx.bc.GetGenesisCert(), []byte("block 1"))
	proposal1 := ctx.createProposal(block1, 1)
	ctx.protocolChan <- proposal1
	qcb1 := ctx.createQC(block1)
	block2 := ctx.bc.NewBlock(block1, qcb1, []byte("block 2"))
	proposal2 := ctx.createProposal(block2, 2)
	ctx.protocolChan <- proposal2
	block3 := ctx.bc.NewBlock(block2, ctx.createQC(block2), []byte("block 3"))
	proposal3 := ctx.createProposal(block3, 3)
	ctx.protocolChan <- proposal3
	block4 := ctx.bc.NewBlock(block3, ctx.createQC(block3), []byte("block 4"))
	proposal4 := ctx.createProposal(block4, 4)
	ctx.protocolChan <- proposal4

	block22 := ctx.bc.NewBlock(block1, qcb1, []byte("block 22"))
	block32 := ctx.bc.NewBlock(block22, qcb1, []byte("block 32"))
	block42 := ctx.bc.NewBlock(block32, qcb1, []byte("block 42"))
	block52 := ctx.bc.NewBlock(block42, qcb1, []byte("block 52"))
	proposal := ctx.createProposal(block52, 5)
	ctx.protocolChan <- proposal

	ctx.blockChan <- block22
	ctx.blockChan <- block32
	ctx.blockChan <- block42
	close(ctx.blockChan)

	ctx.waitRounds(6)

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
	go ctx.pacer.Run(context.Background())
	go ctx.protocol.Run(ctx.protocolChan)

	defer ctx.pacer.Stop()
	defer func() {
		go ctx.protocol.Stop()
	}()

	ctx.StartFirstEpoch()
	ctx.setMe(8)

	block1 := ctx.bc.NewBlock(ctx.bc.GetGenesisBlock(), ctx.bc.GetGenesisCert(), []byte("block 1"))
	proposal1 := ctx.createProposal(block1, 1)
	ctx.protocolChan <- proposal1
	qcb1 := ctx.createQC(block1)
	block2 := ctx.bc.NewBlock(block1, qcb1, []byte("block 2"))
	proposal2 := ctx.createProposal(block2, 2)
	ctx.protocolChan <- proposal2
	qcb2 := ctx.createQC(block2)
	block3 := ctx.bc.NewBlock(block2, qcb2, []byte("block 3"))
	proposal3 := ctx.createProposal(block3, 3)
	ctx.protocolChan <- proposal3
	block4 := ctx.bc.NewBlock(block3, ctx.createQC(block3), []byte("block 4"))
	proposal4 := ctx.createProposal(block4, 4)
	ctx.protocolChan <- proposal4

	block32 := ctx.bc.NewBlock(block2, qcb2, []byte("block 32"))
	block42 := ctx.bc.NewBlock(block32, qcb2, []byte("block 42"))
	block52 := ctx.bc.NewBlock(block42, qcb2, []byte("block 52"))
	proposal := ctx.createProposal(block52, 5)
	ctx.protocolChan <- proposal

	ctx.blockChan <- block2
	ctx.blockChan <- block32
	ctx.blockChan <- block42
	close(ctx.blockChan)

	ctx.waitRounds(6)

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
	go ctx.pacer.Run(context.Background())
	go ctx.protocol.Run(ctx.protocolChan)

	defer ctx.pacer.Stop()
	defer func() {
		go ctx.protocol.Stop()
	}()

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
	ctx.cfg.ControlChan <- *hotstuff.NewCommand(context.Background(), hotstuff.NextView)
	ctx.cfg.ControlChan <- *hotstuff.NewCommand(context.Background(), hotstuff.NextView)
	ctx.cfg.ControlChan <- *hotstuff.NewCommand(context.Background(), hotstuff.NextView)
	ctx.cfg.ControlChan <- *hotstuff.NewCommand(context.Background(), hotstuff.NextView)
	ctx.cfg.ControlChan <- *hotstuff.NewCommand(context.Background(), hotstuff.NextView)

	qcb4 := ctx.createQC(block4)
	block51 := ctx.bc.NewBlock(block4, qcb4, []byte("block 51"))
	block6 := ctx.bc.NewBlock(block51, qcb4, []byte("block 6"))
	proposal6 := ctx.createProposal(block6, 6)
	ctx.protocolChan <- proposal6

	ctx.blockChan <- block2
	ctx.blockChan <- block3
	ctx.blockChan <- block4
	ctx.blockChan <- block51
	close(ctx.blockChan)

	ctx.waitRounds(7)

	assert.Equal(t, block2, ctx.bc.GetTopCommittedBlock())
	assert.Equal(t, int32(6), ctx.protocol.Vheight())
	assert.Equal(t, block6, ctx.bc.GetBlockByHeight(6)[0])
}

type TestContext struct {
	peers        []*common.Peer
	pacer        *hotstuff.StaticPacer
	protocol     *hotstuff.Protocol
	cfg          *hotstuff.ProtocolConfig
	bc           *blockchain.Blockchain
	bsrv         blockchain.BlockService
	eventChan    chan hotstuff.Event
	voteChan     chan *msg.Message
	startChan    chan *msg.Message
	proposalCHan chan *msg.Message
	me           *common.Peer
	protocolChan chan *msg.Message
	blockChan    chan *blockchain.Block
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
	ctx.protocol.SubscribeEpochChange(trigger)
	ctx.sendStartEpochMessages(1, 2*ctx.cfg.F/3+1, nil)
	<-trigger
}

func (ctx *TestContext) sendStartEpochMessages(index int32, amount int, hqc *blockchain.QuorumCertificate) {
	for i := 0; i < amount; i++ {
		var epoch *hotstuff.Epoch
		if hqc == nil {
			sig, _ := crypto.Sign(ctx.bc.GetGenesisBlock().Header().Hash().Bytes(), ctx.pacer.Committee()[i].GetPrivateKey())
			epoch = hotstuff.CreateEpoch(ctx.pacer.Committee()[i], index, nil, sig)
		} else {
			epoch = hotstuff.CreateEpoch(ctx.pacer.Committee()[i], index, hqc, nil)
		}
		message, _ := epoch.GetMessage()
		ctx.protocolChan <- message
	}
}

func (ctx *TestContext) setMe(peerNumber int) {
	ctx.pacer.Committee()[peerNumber] = ctx.me
}

func (ctx *TestContext) waitRounds(count int) {
	for i := 0; i < count; i++ {
		<-ctx.eventChan
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
	identity := generateIdentity(t)
	srv := &mocks.Service{}
	bsrv := &mocks.BlockService{}
	storage := &mocks.Storage{}
	storage.On("PutBlock", mock.AnythingOfType("*blockchain.Block")).Return(nil)
	storage.On("GetBlock", mock.AnythingOfType("common.Hash")).Return(nil, nil)
	storage.On("Contains", mock.AnythingOfType("common.Hash")).Return(false)
	storage.On("PutCurrentTopHeight", mock.AnythingOfType("int32")).Return(nil)
	storage.On("PutCurrentEpoch", mock.AnythingOfType("int32")).Return(nil)
	storage.On("GetCurrentEpoch").Return(int32(0), nil)
	storage.On("GetTopCommittedHeight").Return(0)
	storage.On("PutTopCommittedHeight", mock.AnythingOfType("int32")).Return(nil)

	peers := make([]*common.Peer, 10)

	loader := &mocks.CommitteeLoader{}
	bc := blockchain.CreateBlockchainFromGenesisBlock(storage, bsrv)
	sync := blockchain.CreateSynchronizer(identity, bsrv, bc)
	config := &hotstuff.ProtocolConfig{
		F:           10,
		Delta:       1 * time.Second,
		Blockchain:  bc,
		Me:          identity,
		Srv:         srv,
		Storage:     storage,
		Sync:        sync,
		Committee:   peers,
		ControlChan: make(chan hotstuff.Command),
	}

	for i := 0; i < 10; i++ {
		peers[i] = generateIdentity(t)
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
	loader.On("LoadFromFile").Return(peers)

	pacer := hotstuff.CreatePacer(config)
	config.Pacer = pacer
	p := hotstuff.CreateProtocol(config)
	pacer.SetViewGetter(p)
	pacer.SetEventNotifier(p)

	eventChan := make(chan hotstuff.Event)
	p.SubscribeProtocolEvents(eventChan)

	protocolChan := make(chan *msg.Message)
	return &TestContext{
		voteChan:     voteChan,
		peers:        peers,
		pacer:        pacer,
		protocol:     p,
		bc:           bc,
		cfg:          config,
		proposalCHan: proposalChan,
		blockChan:    blockChan,
		startChan:    startChan,
		eventChan:    eventChan,
		me:           identity,
		bsrv:         bsrv,
		protocolChan: protocolChan,
	}
}
