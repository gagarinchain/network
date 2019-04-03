package test

import (
	"github.com/golang/protobuf/ptypes"
	"github.com/poslibp2p/blockchain"
	"github.com/poslibp2p/hotstuff"
	msg "github.com/poslibp2p/message"
	"github.com/poslibp2p/message/protobuff"
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
	go ctx.pacer.Run()
	go ctx.protocol.Run(ctx.protocolChan)
	defer ctx.pacer.Stop()
	defer ctx.protocol.Stop()

	ctx.StartFirstEpoch()
	ctx.setMe(4)

	newBlock := ctx.bc.NewBlock(ctx.bc.GetHead(), ctx.bc.GetGenesisCert(), []byte("wonderful block"))

	p := ctx.createProposal(newBlock, 3)
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

	assert.Equal(t, int32(4), payload.Block.GetHeader().GetHeight())
	assert.Equal(t, int32(3), payload.Block.GetCert().GetHeader().GetHeight())

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
	go ctx.pacer.Run()
	go ctx.protocol.Run(ctx.protocolChan)

	defer ctx.pacer.Stop()
	defer ctx.protocol.Stop()

	ctx.StartFirstEpoch()
	ctx.setMe(4)

	newBlock := ctx.bc.NewBlock(ctx.bc.GetHead(), ctx.bc.GetGenesisCert(), []byte("wonderful block"))

	votes := ctx.makeVotes(2*ctx.cfg.F/3+1, newBlock)
	for _, v := range votes[:2*ctx.cfg.F/3-2] {
		ctx.protocolChan <- v
	}

	p := ctx.createProposal(newBlock, 3)
	ctx.protocolChan <- p

	for _, v := range votes[2*ctx.cfg.F/3-2:] {
		ctx.protocolChan <- v
	}

	proposal := <-ctx.proposalCHan

	payload := &pb.ProposalPayload{}
	if err := ptypes.UnmarshalAny(proposal.Payload, payload); err != nil {
		log.Error(err)
	}

	assert.Equal(t, int32(4), payload.Block.GetHeader().GetHeight())
	assert.Equal(t, int32(3), payload.Block.GetCert().GetHeader().GetHeight())

}

// Scenario 1c:
// Start new epoch,
// Replica, Next proposer
// Collect 2*f + 1 votes,
// Receive proposal,
// Proposer,
// Propose block with new QC
func TestScenario1c(t *testing.T) {
	ctx := initContext(t)

	go ctx.pacer.Run()
	go ctx.protocol.Run(ctx.protocolChan)

	defer ctx.pacer.Stop()
	defer ctx.protocol.Stop()

	ctx.StartFirstEpoch()
	ctx.setMe(4)

	newBlock := ctx.bc.NewBlock(ctx.bc.GetHead(), ctx.bc.GetGenesisCert(), []byte("wonderful block"))

	votes := ctx.makeVotes(2*ctx.cfg.F/3+1, newBlock)
	for _, v := range votes {
		ctx.protocolChan <- v
	}

	go func() {
		ctx.blockChan <- newBlock
	}()

	p := ctx.createProposal(newBlock, 3)
	ctx.protocolChan <- p

	proposal := <-ctx.proposalCHan

	payload := &pb.ProposalPayload{}
	if err := ptypes.UnmarshalAny(proposal.Payload, payload); err != nil {
		log.Error(err)
	}

	assert.Equal(t, int32(4), payload.Block.GetHeader().GetHeight())
	assert.Equal(t, int32(3), payload.Block.GetCert().GetHeader().GetHeight())
}

//Scenario 2:
//Start new epoch
//Replica
//Get proposal
//Vote
func TestScenario2(t *testing.T) {
	ctx := initContext(t)
	go ctx.pacer.Run()
	go ctx.protocol.Run(ctx.protocolChan)

	defer ctx.pacer.Stop()
	defer ctx.protocol.Stop()

	ctx.StartFirstEpoch()
	ctx.setMe(0)

	newBlock := ctx.bc.NewBlock(ctx.bc.GetHead(), ctx.bc.GetGenesisCert(), []byte("wonderful block"))
	p := ctx.createProposal(newBlock, 3)
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

	go ctx.pacer.Run()
	go ctx.protocol.Run(ctx.protocolChan)

	defer ctx.pacer.Stop()
	defer ctx.protocol.Stop()

	ctx.StartFirstEpoch()
	ctx.setMe(4)

	newBlock := ctx.bc.NewBlock(ctx.bc.GetHead(), ctx.bc.GetGenesisCert(), []byte("wonderful block"))
	p := ctx.createProposal(newBlock, 3)
	ctx.protocolChan <- p
	<-ctx.voteChan

	proposal := <-ctx.proposalCHan

	payload := &pb.ProposalPayload{}
	if err := ptypes.UnmarshalAny(proposal.Payload, payload); err != nil {
		log.Error(err)
	}

	assert.Equal(t, int32(4), payload.Block.GetHeader().GetHeight())
	assert.Equal(t, int32(2), payload.Block.GetCert().GetHeader().GetHeight())
}

//Scenario 4:
//Start new epoch
//Next Proposer
//Proposer equivocates, no proposal sent
//Propose block with previous QC (2) after 2*Delta
func TestScenario4(t *testing.T) {
	ctx := initContext(t)
	go ctx.pacer.Run()
	go ctx.protocol.Run(ctx.protocolChan)

	defer ctx.pacer.Stop()
	defer ctx.protocol.Stop()
	ctx.StartFirstEpoch()
	ctx.setMe(4)

	proposal := <-ctx.proposalCHan

	payload := &pb.ProposalPayload{}
	if err := ptypes.UnmarshalAny(proposal.Payload, payload); err != nil {
		log.Error(err)
	}

	assert.Equal(t, int32(4), payload.Block.GetHeader().GetHeight())
	assert.Equal(t, int32(2), payload.Block.GetCert().GetHeader().GetHeight())
}

//Scenario 5a:
//Start new epoch
//Receive no messages
//Receive 2f + 1 start messages
//Start new epoch
//Propose
func TestScenario5a(t *testing.T) {
	ctx := initContext(t)
	go ctx.pacer.Run()
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
	assert.Equal(t, int32(2), payload.Block.GetCert().GetHeader().GetHeight())
}

//Scenario 5b:
//Start new epoch
//Receive no messages
//Start new epoch after epoch end
//Propose
func TestScenario5b(t *testing.T) {
	ctx := initContext(t)
	go ctx.pacer.Run()
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
	assert.Equal(t, int32(2), payload.Block.GetCert().GetHeader().GetHeight())
}

type TestContext struct {
	peers        []*msg.Peer
	pacer        *hotstuff.StaticPacer
	protocol     *hotstuff.Protocol
	cfg          *hotstuff.ProtocolConfig
	bc           *blockchain.Blockchain
	voteChan     chan *msg.Message
	startChan    chan *msg.Message
	proposalCHan chan *msg.Message
	me           *msg.Peer
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

func makeVote(bc *blockchain.Blockchain, newBlock *blockchain.Block, peer *msg.Peer) *hotstuff.Vote {
	vote := hotstuff.CreateVote(newBlock.Header(), bc.GetGenesisCert(), peer)
	vote.Sign(peer.GetPrivateKey())
	return vote
}

func (ctx *TestContext) StartFirstEpoch() {
	trigger := make(chan interface{})
	ctx.protocol.SubscribeEpochChange(trigger)
	ctx.sendStartEpochMessages(1, 2*ctx.cfg.F/3+1, ctx.protocol.HQC())
	<-trigger
}

func (ctx *TestContext) sendStartEpochMessages(index int32, amount int, hqc *blockchain.QuorumCertificate) {
	for i := 0; i < amount; i++ {
		epoch := hotstuff.CreateEpoch(ctx.pacer.Committee()[i], index, hqc)
		message, _ := epoch.GetMessage()
		ctx.protocolChan <- message
	}
}

func (ctx *TestContext) setMe(peerNumber int) {
	ctx.pacer.Committee()[peerNumber] = ctx.me
}

func (ctx *TestContext) createProposal(newBlock *blockchain.Block, peerNumber int) *msg.Message {
	proposal := hotstuff.CreateProposal(newBlock, newBlock.QC(), ctx.peers[peerNumber])
	proposal.Sign(ctx.peers[peerNumber].GetPrivateKey())
	any, _ := ptypes.MarshalAny(proposal.GetMessage())
	return msg.CreateMessage(pb.Message_PROPOSAL, any, ctx.peers[peerNumber])
}

func initContext(t *testing.T) *TestContext {
	identity := generateIdentity(t)
	srv := &mocks.Service{}
	//synchr := &mocks.Synchronizer{}
	bsrv := &mocks.BlockService{}
	storage := &mocks.Storage{}
	storage.On("PutBlock", mock.AnythingOfType("*blockchain.Block")).Return(nil)
	storage.On("GetBlock", mock.AnythingOfType("common.Hash")).Return(nil, nil)
	storage.On("Contains", mock.AnythingOfType("common.Hash")).Return(false)
	storage.On("PutCurrentTopHeight", mock.AnythingOfType("int32")).Return(nil)
	storage.On("PutCurrentEpoch", mock.AnythingOfType("int32")).Return(nil)

	peers := make([]*msg.Peer, 10)

	loader := &mocks.CommitteeLoader{}
	bc := blockchain.CreateBlockchainFromGenesisBlock(storage, bsrv)
	config := &hotstuff.ProtocolConfig{
		F:            10,
		Delta:        1 * time.Second,
		Blockchain:   bc,
		Me:           identity,
		Srv:          srv,
		Storage:      storage,
		Committee:    peers,
		RoundEndChan: make(chan int32),
		ControlChan:  make(chan hotstuff.Event),
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
	srv.On("Broadcast", mock.MatchedBy(matcher(pb.Message_EPOCH_START))).Run(func(args mock.Arguments) {
		startChan <- (args[0]).(*msg.Message)
	})
	proposalChan := make(chan *msg.Message)
	srv.On("Broadcast", mock.MatchedBy(matcher(pb.Message_PROPOSAL))).Run(func(args mock.Arguments) {
		proposalChan <- (args[0]).(*msg.Message)
	})

	voteChan := make(chan *msg.Message)
	srv.On("SendMessage", mock.AnythingOfType("*message.Peer"), mock.MatchedBy(matcher(pb.Message_VOTE))).
		Return(make(chan *msg.Message)).Run(func(args mock.Arguments) {
		voteChan <- (args[1]).(*msg.Message)
	})

	blockChan := make(chan *blockchain.Block)
	blockChanRead := func(c chan *blockchain.Block) <-chan *blockchain.Block {
		return c
	}(blockChan)
	bsrv.On("RequestBlock", mock.AnythingOfType("common.Hash")).Return(blockChanRead)
	loader.On("LoadFromFile").Return(peers)

	pacer := hotstuff.CreatePacer(config)
	config.Pacer = pacer
	p := hotstuff.CreateProtocol(config)
	pacer.SetViewGetter(p)
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
		me:           identity,
		protocolChan: protocolChan,
	}
}
