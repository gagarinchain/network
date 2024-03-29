package test

import (
	"context"
	"github.com/gagarinchain/common"
	"github.com/gagarinchain/common/api"
	"github.com/gagarinchain/common/eth/crypto"
	msg "github.com/gagarinchain/common/message"
	cmocks "github.com/gagarinchain/common/mocks"
	"github.com/gagarinchain/common/protobuff"
	"github.com/gagarinchain/network/blockchain"
	"github.com/gagarinchain/network/hotstuff"
	"github.com/gagarinchain/network/mocks"
	"github.com/golang/protobuf/ptypes"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/op/go-logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"math/big"
	"testing"
	"time"
)

var log = logging.MustGetLogger("hotstuff")

func TestProposalSignature(t *testing.T) {
	bc, _, cfg, _ := initProtocol(t)

	block, _ := bc.NewBlock(bc.GetGenesisBlock(), bc.GetGenesisCert(), []byte("hello sign"))
	proposal := hotstuff.CreateProposal(block, cfg.Me, block.QC())

	proposal.Sign(cfg.Me.GetPrivateKey())

	payload := proposal.GetMessage()
	any, e := ptypes.MarshalAny(payload)
	if e != nil {
		log.Error(e)
	}
	m := msg.CreateMessage(pb.Message_PROPOSAL, any, &common.Peer{})

	proposal2, e := hotstuff.CreateProposalFromMessage(m)
	if e != nil {
		t.Error(e)
	}

	assert.Equal(t, cfg.Me.GetAddress(), proposal2.Sender().GetAddress())

}

func TestProtocolProposeOnGenesisBlockchain(t *testing.T) {
	_, p, cfg, _ := initProtocol(t, 1)

	mocksrv := (cfg.Srv).(*mocks.Service)

	msgChan := make(chan *msg.Message)
	mocksrv.On("Broadcast", mock.MatchedBy(func(ctx context.Context) bool { return true }), mock.AnythingOfType("*message.Message")).Run(func(args mock.Arguments) {
		msgChan <- (args[1]).(*msg.Message)
		assert.Equal(t, pb.Message_PROPOSAL, args[1].(*msg.Message).Type)
	}).Once()

	go p.OnPropose(context.Background())

	m := <-msgChan
	assert.Equal(t, pb.Message_PROPOSAL, m.Type)

	mocksrv.AssertCalled(t, "Broadcast", mock.MatchedBy(func(ctx context.Context) bool { return true }), mock.AnythingOfType("*message.Message"))
}

func TestProtocolProposeOnGenesisBlockchainVoteForSelfProposal(t *testing.T) {
	_, p, cfg, _ := initProtocol(t, 1)

	mocksrv := (cfg.Srv).(*mocks.Service)

	msgChan := make(chan *msg.Message)
	mocksrv.On("Broadcast", mock.MatchedBy(func(ctx context.Context) bool { return true }), mock.AnythingOfType("*message.Message")).Run(func(args mock.Arguments) {
		msgChan <- (args[1]).(*msg.Message)
		assert.Equal(t, pb.Message_PROPOSAL, args[1].(*msg.Message).Type)
	}).Once()
	mocksrv.On("SendMessage", mock.MatchedBy(func(ctx context.Context) bool { return true }),
		cfg.Committee[2], mock.AnythingOfType("*message.Message")).Run(func(args mock.Arguments) {
		msgChan <- (args[2]).(*msg.Message)
	}).Return(make(chan *msg.Message), nil).Once()

	go p.OnPropose(context.Background())

	m := <-msgChan
	assert.Equal(t, pb.Message_PROPOSAL, m.Type)

	proposal, _ := hotstuff.CreateProposalFromMessage(msg.CreateMessage(m.Type, m.Payload, &common.Peer{}))
	go p.OnReceiveProposal(context.Background(), proposal)

	v := <-msgChan

	assert.Equal(t, pb.Message_VOTE, v.Type)

}

func TestProtocolUpdateWithHigherRankCertificate(t *testing.T) {
	bc, p, cfg, _ := initProtocol(t)

	newBlock, _ := bc.NewBlock(bc.GetHead(), bc.GetGenesisCert(), []byte(""))
	log.Info("Head ", newBlock.Header().Hash().Hex())

	newQC := createValidQC(newBlock, cfg)
	if _, e := bc.AddBlock(newBlock); e != nil {
		t.Error(e)
	}

	p.Update(newQC)
	hqc := p.HQC()

	assert.Equal(t, newQC, hqc)
}

func createValidQC(newBlock api.Block, cfg *hotstuff.ProtocolConfig) api.QuorumCertificate {
	m := newBlock.Header().Hash().Bytes()
	var signs []*crypto.Signature
	for _, v := range cfg.Committee {
		sign := crypto.Sign(m, v.GetPrivateKey())
		signs = append(signs, sign)
	}
	aggregate := crypto.AggregateSignatures(big.NewInt(1<<len(cfg.Committee)-1), signs)
	newQC := blockchain.CreateQuorumCertificate(aggregate, newBlock.Header(), api.QRef)
	return newQC
}

func TestProtocolUpdateWithLowerRankCertificate(t *testing.T) {
	bc, p, cfg, _ := initProtocol(t)

	head := bc.GetHead()
	newBlock, _ := bc.NewBlock(head, bc.GetGenesisCert(), []byte(""))
	log.Info("Head ", newBlock.Header().Hash().Hex())
	newQC := createValidQC(newBlock, cfg)
	if _, e := bc.AddBlock(newBlock); e != nil {
		t.Error(e)
	}

	p.Update(newQC)     //Update to new
	p.Update(head.QC()) //Revert to old
	hqc := p.HQC()

	assert.Equal(t, newQC, hqc)
}

func TestOnReceiveProposal(t *testing.T) {
	bc, p, cfg, _ := initProtocol(t)
	newBlock, _ := bc.NewBlock(bc.GetHead(), bc.GetGenesisCert(), []byte("wonderful block"))

	for _, p := range cfg.Committee {
		log.Info(p.GetAddress().Hex())
	}

	currentProposer := cfg.Committee[1]
	nextProposer := cfg.Committee[2]
	proposal := hotstuff.CreateProposal(newBlock, currentProposer, newBlock.QC())
	srv := (cfg.Srv).(*mocks.Service)
	msgChan := make(chan *msg.Message)
	srv.On("SendMessage", mock.MatchedBy(func(ctx context.Context) bool { return true }),
		nextProposer, mock.AnythingOfType("*message.Message")).Run(func(args mock.Arguments) {
		msgChan <- (args[2]).(*msg.Message)
	}).Return(make(chan *msg.Message), nil).Once()

	go func() {
		if err := p.OnReceiveProposal(context.Background(), proposal); err != nil {
			t.Error("Error while receiving proposal", err)
		}
	}()
	m := <-msgChan

	vote, err := hotstuff.CreateVoteFromMessage(m)
	if err != nil {
		t.Error("can't create vote", err)
	}

	srv.AssertCalled(t, "SendMessage", mock.MatchedBy(func(ctx context.Context) bool { return true }),
		nextProposer, mock.AnythingOfType("*message.Message"))
	assert.Equal(t, proposal.NewBlock().Header().Hash(), vote.Header().Hash())
	assert.Equal(t, vote.Header().Height(), p.Vheight())

}

func TestOnReceiveProposalFromWrongProposer(t *testing.T) {
	bc, p, cfg, _ := initProtocol(t)
	newBlock, _ := bc.NewBlock(bc.GetHead(), bc.GetGenesisCert(), []byte("wonderful block"))
	if _, e := bc.AddBlock(newBlock); e != nil {
		t.Error("can't add block", e)
	}

	nextProposer := cfg.Pacer.GetNext()
	proposal := hotstuff.CreateProposal(newBlock, nextProposer, newBlock.QC())

	assert.Error(t, p.OnReceiveProposal(context.Background(), proposal), "peer equivocated")
	assert.Equal(t, int32(0), p.Vheight())
}

func TestOnReceiveVoteForNotProposer(t *testing.T) {
	bc, p, _, _ := initProtocol(t, 4)

	newBlock, _ := bc.NewBlock(bc.GetHead(), bc.GetGenesisCert(), []byte("wonderful block"))
	vote := createVote(bc, newBlock, t)

	assert.Nil(t, p.OnReceiveVote(context.Background(), vote))
}

func TestOnReceiveTwoVotesSamePeer(t *testing.T) {
	bc, p, cfg, _ := initProtocol(t, 1)
	id := generateIdentity(t, 4)
	(cfg.Pacer).(*cmocks.Pacer).On("GetCurrent").Return(cfg.Me)
	newBlock1, _ := bc.NewBlock(bc.GetHead(), bc.GetGenesisCert(), []byte("wonderful block"))
	newBlock2, _ := bc.NewBlock(bc.GetHead(), bc.GetGenesisCert(), []byte("another wonderful block"))
	vote1 := hotstuff.CreateVote(newBlock1.Header(), bc.GetGenesisCert(), id)
	vote2 := hotstuff.CreateVote(newBlock2.Header(), bc.GetGenesisCert(), id)

	if err := p.OnReceiveVote(context.Background(), vote1); err != nil {
		t.Error("failed OnReceive", err)
	}
	err := p.OnReceiveVote(context.Background(), vote2)
	if err == nil {
		t.Fail()
	}

	assert.Error(t, err)
}

func TestQCUpdateOnVotesCollectFinish(t *testing.T) {
	bc, p, cfg, _ := initProtocol(t, 1)
	newBlock, _ := bc.NewBlock(bc.GetHead(), bc.GetGenesisCert(), []byte("wonderful block"))

	if _, e := bc.AddBlock(newBlock); e != nil {
		t.Error("can't add block", e)
	}

	votes := createVotes((cfg.F/3)*2+1, bc, newBlock, t)

	msgChan := make(chan *msg.Message)
	(cfg.Srv).(*mocks.Service).On("Broadcast", mock.MatchedBy(func(ctx context.Context) bool { return true }),
		mock.AnythingOfType("*message.Message")).Run(func(args mock.Arguments) {
		msgChan <- (args[1]).(*msg.Message)
	})

	for _, vote := range votes {
		if err := p.OnReceiveVote(context.Background(), vote); err != nil {
			t.Error("failed OnReceive", err)
		}
	}

	assert.Equal(t, newBlock.Header().Hash(), p.HQC().QrefBlock().Hash())
}

func createVote(bc api.Blockchain, newBlock api.Block, t *testing.T) api.Vote {
	vote := hotstuff.CreateVote(newBlock.Header(), bc.GetGenesisCert(), generateIdentity(t, 2))
	return vote
}

func createVotes(count int, bc api.Blockchain, newBlock api.Block, t *testing.T) []api.Vote {
	votes := make([]api.Vote, count)

	for i := 0; i < count; i++ {
		peer := generateIdentity(t, i)
		vote := hotstuff.CreateVote(newBlock.Header(), bc.GetGenesisCert(), peer)
		vote.Sign(peer.GetPrivateKey())
		votes[i] = vote
	}

	return votes
}

func initProtocol(t *testing.T, inds ...int) (api.Blockchain, *hotstuff.Protocol, *hotstuff.ProtocolConfig, chan api.Event) {
	var ind int
	if inds != nil {
		ind = inds[0]
	}
	me := generateIdentity(t, ind)
	srv := &mocks.Service{}
	storage := SoftStorageMock()
	bpersister := &blockchain.BlockPersister{storage}
	cpersister := &blockchain.BlockchainPersister{storage}

	//synchr := &mocks.Synchronizer{}
	loader := &cmocks.CommitteeLoader{}
	bc := blockchain.CreateBlockchainFromGenesisBlock(&blockchain.BlockchainConfig{
		ChainPersister: cpersister, BlockPerister: bpersister, Pool: mockPool(), Db: mockDB(), ProposerGetter: MockProposerForHeight(),
	})
	bc.GetGenesisBlock().SetQC(blockchain.CreateQuorumCertificate(crypto.EmptyAggregateSignatures(), bc.GetGenesisBlock().Header(), api.QRef))

	peers := make([]*common.Peer, 10)

	f := 10
	config := &hotstuff.ProtocolConfig{
		F:            f,
		Delta:        5 * time.Second,
		Blockchain:   bc,
		Me:           me,
		Srv:          srv,
		Storage:      storage,
		Committee:    peers,
		InitialState: hotstuff.DefaultState(bc),
	}

	for i := 0; i < f; i++ {
		peers[i] = generateIdentity(t, i)
	}

	loader.On("LoadPeerListFromFile").Return(peers)

	pacer := &cmocks.Pacer{}
	config.Pacer = pacer
	pacer.On("FireEvent", mock.AnythingOfType("api.EventType"))
	pacer.On("NotifyEvent", mock.AnythingOfType("api.Event"))
	pacer.On("GetCurrentView").Return(int32(1))
	pacer.On("GetCurrent").Return(peers[1])
	pacer.On("GetNext").Return(peers[2])
	pacer.On("SubscribeProtocolEvents", mock.AnythingOfType("chan api.Event"))
	pacer.On("FireEvent", mock.AnythingOfType("api.Event"))
	pacer.On("GetBitmap", mock.AnythingOfType("map[common.Address]*crypto.Signature")).Return(big.NewInt(1<<((f/3)*2+1)-1), 0)
	pacer.On("GetPeers").Return(peers)

	p := hotstuff.CreateProtocol(config)
	eventChan := make(chan api.Event)
	pacer.SubscribeProtocolEvents(eventChan)

	p.Update(bc.GetGenesisCert())
	return bc, p, config, eventChan
}

func mustAddr(t *testing.T, s string) ma.Multiaddr {
	addr, err := ma.NewMultiaddr(s)
	if err != nil {
		t.Fatal(err)
	}

	return addr
}
