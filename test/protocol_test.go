package test

import (
	"context"
	"github.com/gagarinchain/network/blockchain"
	"github.com/gagarinchain/network/common"
	"github.com/gagarinchain/network/common/eth/crypto"
	msg "github.com/gagarinchain/network/common/message"
	"github.com/gagarinchain/network/common/protobuff"
	"github.com/gagarinchain/network/hotstuff"
	"github.com/gagarinchain/network/mocks"
	"github.com/golang/protobuf/ptypes"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/op/go-logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"math/big"
	"strconv"
	"testing"
	"time"
)

var log = logging.MustGetLogger("hotstuff")

func TestProposalSignature(t *testing.T) {
	bc, _, cfg, _ := initProtocol(t)

	block := bc.NewBlock(bc.GetGenesisBlock(), bc.GetGenesisCert(), []byte("hello sign"))
	proposal := hotstuff.CreateProposal(block, bc.GetGenesisCert(), cfg.Me)

	proposal.Sign(cfg.Me.GetPrivateKey())

	payload := proposal.GetMessage()
	any, e := ptypes.MarshalAny(payload)
	if e != nil {
		log.Error(e)
	}
	m := msg.CreateMessage(pb.Message_PROPOSAL, any, &common.Peer{})

	proposal2, e := hotstuff.CreateProposalFromMessage(m)

	assert.Equal(t, cfg.Me.GetAddress(), proposal2.Sender.GetAddress())

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
	bc, p, _, _ := initProtocol(t)

	newBlock := bc.NewBlock(bc.GetHead(), bc.GetGenesisCert(), []byte(""))
	log.Info("Head ", newBlock.Header().Hash().Hex())
	newQC := blockchain.CreateQuorumCertificate(crypto.EmptyAggregateSignatures(), newBlock.Header())
	if e := bc.AddBlock(newBlock); e != nil {
		t.Error(e)
	}

	p.Update(newQC)
	hqc := p.HQC()

	assert.Equal(t, newQC, hqc)
}

func TestProtocolUpdateWithLowerRankCertificate(t *testing.T) {
	bc, p, _, _ := initProtocol(t)

	head := bc.GetHead()
	newBlock := bc.NewBlock(head, bc.GetGenesisCert(), []byte(""))
	log.Info("Head ", newBlock.Header().Hash().Hex())
	newQC := blockchain.CreateQuorumCertificate(crypto.EmptyAggregateSignatures(), newBlock.Header())
	if e := bc.AddBlock(newBlock); e != nil {
		t.Error(e)
	}

	p.Update(newQC)     //Update to new
	p.Update(head.QC()) //Revert to old
	hqc := p.HQC()

	assert.Equal(t, newQC, hqc)
}

func TestOnReceiveProposal(t *testing.T) {
	bc, p, cfg, _ := initProtocol(t)
	head := bc.GetHead()
	newBlock := bc.NewBlock(bc.GetHead(), bc.GetGenesisCert(), []byte("wonderful block"))

	for _, p := range cfg.Committee {
		log.Info(p.GetAddress().Hex())
	}

	currentProposer := cfg.Committee[1]
	nextProposer := cfg.Committee[2]
	proposal := hotstuff.CreateProposal(newBlock, head.QC(), currentProposer)
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
	assert.Equal(t, proposal.NewBlock.Header().Hash(), vote.Header.Hash())
	assert.Equal(t, vote.Header.Height(), p.Vheight())

}

func TestOnReceiveProposalFromWrongProposer(t *testing.T) {
	bc, p, cfg, _ := initProtocol(t)
	head := bc.GetHead()
	newBlock := bc.NewBlock(bc.GetHead(), bc.GetGenesisCert(), []byte("wonderful block"))
	if e := bc.AddBlock(newBlock); e != nil {
		t.Error("can't add block", e)
	}

	nextProposer := cfg.Pacer.GetNext()
	proposal := &hotstuff.Proposal{Sender: nextProposer, NewBlock: newBlock, HQC: head.QC()}

	assert.Error(t, p.OnReceiveProposal(context.Background(), proposal), "peer equivocated")
	assert.Equal(t, int32(0), p.Vheight())
}

func TestOnReceiveVoteForNotProposer(t *testing.T) {
	bc, p, _, _ := initProtocol(t, 4)

	newBlock := bc.NewBlock(bc.GetHead(), bc.GetGenesisCert(), []byte("wonderful block"))
	vote := createVote(bc, newBlock, t)

	assert.Error(t, p.OnReceiveVote(context.Background(), vote))
}

func TestOnReceiveTwoVotesSamePeer(t *testing.T) {
	bc, p, cfg, _ := initProtocol(t, 1)
	id := generateIdentity(t, 4)
	(cfg.Pacer).(*mocks.Pacer).On("GetCurrent").Return(cfg.Me)
	newBlock1 := bc.NewBlock(bc.GetHead(), bc.GetGenesisCert(), []byte("wonderful block"))
	newBlock2 := bc.NewBlock(bc.GetHead(), bc.GetGenesisCert(), []byte("another wonderful block"))
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

func TestOnReceiveVote(t *testing.T) {
	bc, p, cfg, _ := initProtocol(t, 1)
	newBlock := bc.NewBlock(bc.GetHead(), bc.GetGenesisCert(), []byte("wonderful block"))

	if e := bc.AddBlock(newBlock); e != nil {
		t.Error("can't add block", e)
	}

	votes := createVotes((cfg.F/3)*2+1, bc, newBlock, t)

	msgChan := make(chan *msg.Message)
	(cfg.Srv).(*mocks.Service).On("Broadcast", mock.MatchedBy(func(ctx context.Context) bool { return true }),
		mock.AnythingOfType("*message.Message")).Run(func(args mock.Arguments) {
		msgChan <- (args[1]).(*msg.Message)
	})

	for _, vote := range votes[:(cfg.F/3)*2] {
		if err := p.OnReceiveVote(context.Background(), vote); err != nil {
			t.Error("failed OnReceive", err)
		}
	}
	if e := p.OnReceiveVote(context.Background(), votes[(cfg.F/3)*2]); e != nil {
		t.Error("failed OnReceive", e)
	}

	assert.Equal(t, newBlock.Header().Hash(), p.HQC().QrefBlock().Hash())
}

func createVote(bc *blockchain.Blockchain, newBlock *blockchain.Block, t *testing.T) *hotstuff.Vote {
	vote := hotstuff.CreateVote(newBlock.Header(), bc.GetGenesisCert(), generateIdentity(t, 2))
	return vote
}

func createVotes(count int, bc *blockchain.Blockchain, newBlock *blockchain.Block, t *testing.T) []*hotstuff.Vote {
	votes := make([]*hotstuff.Vote, count)

	for i := 0; i < count; i++ {
		votes[i] = hotstuff.CreateVote(newBlock.Header(), bc.GetGenesisCert(), generateIdentity(t, i))
	}

	return votes
}

func initProtocol(t *testing.T, inds ...int) (*blockchain.Blockchain, *hotstuff.Protocol, *hotstuff.ProtocolConfig, chan hotstuff.Event) {
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
	loader := &mocks.CommitteeLoader{}
	bsrv := &mocks.BlockService{}
	bc := blockchain.CreateBlockchainFromGenesisBlock(&blockchain.BlockchainConfig{
		ChainPersister: cpersister, BlockPerister: bpersister, BlockService: bsrv, Pool: mockPool(), Db: mockDB(), ProposerGetter: MockProposerForHeight(),
	})
	bc.GetGenesisBlock().SetQC(blockchain.CreateQuorumCertificate(crypto.EmptyAggregateSignatures(), bc.GetGenesisBlock().Header()))

	peers := make([]*common.Peer, 10)

	config := &hotstuff.ProtocolConfig{
		F:          10,
		Delta:      5 * time.Second,
		Blockchain: bc,
		Me:         me,
		Srv:        srv,
		Storage:    storage,
		Committee:  peers,
	}

	for i := 0; i < 10; i++ {
		peers[i] = generateIdentity(t, i)
	}

	loader.On("LoadPeerListFromFile").Return(peers)

	pacer := &mocks.Pacer{}
	config.Pacer = pacer
	pacer.On("FireEvent", mock.AnythingOfType("hotstuff.EventType"))
	pacer.On("GetCurrentView").Return(int32(1))
	pacer.On("GetCurrent").Return(peers[1])
	pacer.On("GetNext").Return(peers[2])
	pacer.On("SubscribeProtocolEvents", mock.AnythingOfType("chan hotstuff.Event"))
	pacer.On("FireEvent", mock.AnythingOfType("hotstuff.Event"))
	pacer.On("GetBitmap", mock.AnythingOfType("map[common.Address]*crypto.Signature")).Return(big.NewInt(0), 0)

	p := hotstuff.CreateProtocol(config)
	eventChan := make(chan hotstuff.Event)
	pacer.SubscribeProtocolEvents(eventChan)

	return bc, p, config, eventChan
}

func generateIdentity(t *testing.T, ind int) *common.Peer {
	loader := &common.CommitteeLoaderImpl{}
	committee := loader.LoadPeerListFromFile("../static/peers.json")
	_, _ = loader.LoadPeerFromFile("../static/peer"+strconv.Itoa(ind)+".json", committee[ind])

	return committee[ind]
}

func mustAddr(t *testing.T, s string) ma.Multiaddr {
	addr, err := ma.NewMultiaddr(s)
	if err != nil {
		t.Fatal(err)
	}

	return addr
}
