package test

import (
	"github.com/libp2p/go-libp2p-peer"
	"github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/op/go-logging"
	"github.com/poslibp2p/blockchain"
	"github.com/poslibp2p/eth/crypto"
	"github.com/poslibp2p/hotstuff"
	msg "github.com/poslibp2p/message"
	"github.com/poslibp2p/message/protobuff"
	"github.com/poslibp2p/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	mrnd "math/rand"
	"strconv"
	"strings"
	"testing"
	"time"
)

var log = logging.MustGetLogger("hotstuff")

func TestProtocolProposeOnGenesisBlockchain(t *testing.T) {
	_, p, cfg := initProtocol(t)
	mocksrv := (cfg.Srv).(*mocks.Service)

	cfg.Pacer.Committee()[3] = cfg.Me
	msgChan := make(chan *msg.Message)
	mocksrv.On("Broadcast", mock.AnythingOfType("*message.Message")).Run(func(args mock.Arguments) {
		msgChan <- (args[0]).(*msg.Message)
		assert.Equal(t, pb.Message_PROPOSAL, args[0].(*msg.Message).Type)
	}).Once()

	go p.OnPropose()

	<-cfg.RoundEndChan
	m := <-msgChan
	assert.Equal(t, pb.Message_PROPOSAL, m.Type)

	mocksrv.AssertCalled(t, "Broadcast", mock.AnythingOfType("*message.Message"))
}

func TestProtocolUpdateWithHigherRankCertificate(t *testing.T) {
	bc, p, _ := initProtocol(t)

	newBlock := bc.NewBlock(bc.GetHead(), bc.GetGenesisCert(), []byte(""))
	log.Info("Head ", newBlock.Header().Hash().Hex())
	newQC := blockchain.CreateQuorumCertificate([]byte("New QC"), newBlock.Header())
	if e := bc.AddBlock(newBlock); e != nil {
		t.Error(e)
	}

	p.Update(newQC)
	hqc := p.HQC()

	assert.Equal(t, newQC, hqc)
}

func TestProtocolUpdateWithLowerRankCertificate(t *testing.T) {
	bc, p, _ := initProtocol(t)

	head := bc.GetHead()
	newBlock := bc.NewBlock(head, bc.GetGenesisCert(), []byte(""))
	log.Info("Head ", newBlock.Header().Hash().Hex())
	newQC := blockchain.CreateQuorumCertificate([]byte("New QC"), newBlock.Header())
	if e := bc.AddBlock(newBlock); e != nil {
		t.Error(e)
	}

	p.Update(newQC)     //Update to new
	p.Update(head.QC()) //Revert to old
	hqc := p.HQC()

	assert.Equal(t, newQC, hqc)
}

func TestOnReceiveProposal(t *testing.T) {
	bc, p, cfg := initProtocol(t)
	head := bc.GetHead()
	newBlock := bc.NewBlock(bc.GetHead(), bc.GetGenesisCert(), []byte("wonderful block"))
	//if e := bc.AddBlock(newBlock); e != nil {
	//	t.Error("can't add block", e)
	//}

	for _, p := range cfg.Pacer.Committee() {
		log.Info(p.GetAddress().Hex())
	}

	currentProposer := cfg.Pacer.Committee()[3]
	nextProposer := cfg.Pacer.Committee()[4]
	proposal := hotstuff.CreateProposal(newBlock, head.QC(), currentProposer)
	srv := (cfg.Srv).(*mocks.Service)
	msgChan := make(chan *msg.Message)
	srv.On("SendMessage", nextProposer, mock.AnythingOfType("*message.Message")).Run(func(args mock.Arguments) {
		msgChan <- (args[1]).(*msg.Message)
	}).Once()

	go func() {
		if err := p.OnReceiveProposal(proposal); err != nil {
			t.Error("Error while receiving proposal", err)
		}
	}()
	m := <-msgChan
	<-cfg.RoundEndChan

	vote, err := hotstuff.CreateVoteFromMessage(m, cfg.Me)
	if err != nil {
		t.Error("can't create vote", err)
	}

	srv.AssertCalled(t, "SendMessage", nextProposer, mock.AnythingOfType("*message.Message"))
	assert.Equal(t, proposal.NewBlock.Header().Hash(), vote.Header.Hash())
	assert.Equal(t, vote.Header.Height(), p.Vheight())

}

func TestOnReceiveProposalFromWrongProposer(t *testing.T) {
	bc, p, cfg := initProtocol(t)
	head := bc.GetHead()
	newBlock := bc.NewBlock(bc.GetHead(), bc.GetGenesisCert(), []byte("wonderful block"))
	if e := bc.AddBlock(newBlock); e != nil {
		t.Error("can't add block", e)
	}

	nextProposer := cfg.Pacer.GetNext(p.GetCurrentView())
	proposal := &hotstuff.Proposal{Sender: nextProposer, NewBlock: newBlock, HQC: head.QC()}

	assert.Error(t, p.OnReceiveProposal(proposal), "peer equivocated")
	assert.Equal(t, int32(2), p.Vheight())
}

func TestOnReceiveVoteForNotProposer(t *testing.T) {
	bc, p, _ := initProtocol(t)

	newBlock := bc.NewBlock(bc.GetHead(), bc.GetGenesisCert(), []byte("wonderful block"))
	vote := createVote(bc, newBlock, t)

	assert.Error(t, p.OnReceiveVote(vote))
}

func TestOnReceiveTwoVotesSamePeer(t *testing.T) {
	bc, p, cfg := initProtocol(t)
	id := generateIdentity(t)
	cfg.Pacer.Committee()[3] = cfg.Me
	newBlock1 := bc.NewBlock(bc.GetHead(), bc.GetGenesisCert(), []byte("wonderful block"))
	newBlock2 := bc.NewBlock(bc.GetHead(), bc.GetGenesisCert(), []byte("another wonderful block"))
	vote1 := hotstuff.CreateVote(newBlock1.Header(), bc.GetGenesisCert(), id)
	vote2 := hotstuff.CreateVote(newBlock2.Header(), bc.GetGenesisCert(), id)

	if err := p.OnReceiveVote(vote1); err != nil {
		t.Error("failed OnReceive", err)
	}
	err := p.OnReceiveVote(vote2)
	if err == nil {
		t.Fail()
	}

	assert.Error(t, err)
}

func TestOnReceiveVote(t *testing.T) {
	bc, p, cfg := initProtocol(t)
	cfg.Pacer.Committee()[3] = cfg.Me
	newBlock := bc.NewBlock(bc.GetHead(), bc.GetGenesisCert(), []byte("wonderful block"))

	if e := bc.AddBlock(newBlock); e != nil {
		t.Error("can't add block", e)
	}

	votes := createVotes((cfg.F/3)*2+1, bc, newBlock, t)

	msgChan := make(chan *msg.Message)
	(cfg.Srv).(*mocks.Service).On("Broadcast", mock.AnythingOfType("*message.Message")).Run(func(args mock.Arguments) {
		msgChan <- (args[0]).(*msg.Message)
	})

	for _, vote := range votes[:(cfg.F/3)*2] {
		if err := p.OnReceiveVote(vote); err != nil {
			t.Error("failed OnReceive", err)
		}
	}

	go func() {
		if e := p.OnReceiveVote(votes[(cfg.F/3)*2]); e != nil {
			t.Error("failed OnReceive", e)
		}
	}()

	<-cfg.RoundEndChan
	m := <-msgChan

	assert.Equal(t, newBlock.Header().Hash(), p.HQC().QrefBlock().Hash())
	assert.Equal(t, m.Type, pb.Message_PROPOSAL)

}

//
//func TestStartEpochOnGenesisBlock(t *testing.T) {
//	_, p, cfg := initProtocol(t)
//
//	msgChan := make(chan *msg.Message)
//	(cfg.Srv).(*mocks.Service).On("Broadcast", mock.AnythingOfType("*message.Message")).Run(func(args mock.Arguments) {
//		msgChan <- (args[0]).(*msg.Message)
//	})
//
//	go ctx.pacer.Run()
//	go ctx.protocol.Run(ctx.protocolChan)
//	defer ctx.pacer.Stop()
//	defer ctx.protocol.Stop()
//
//	<-msgChan
//
//	for i := 0; i < 2*cfg.F/3; i++ {
//		epoch := hotstuff.CreateEpoch(cfg.Pacer.Committee()[i], 1, p.HQC())
//		message, _ := epoch.GetMessage()
//		p.OnEpochStart(message, cfg.Pacer.Committee()[i])
//	}
//
//	assert.True(t, p.IsStartingEpoch)
//
//	trigger := make(chan interface{})
//	p.SubscribeEpochChange(trigger)
//
//	epoch := hotstuff.CreateEpoch(cfg.Pacer.Committee()[2*cfg.F/3], 1, p.HQC())
//	message, _ := epoch.GetMessage()
//	p.OnEpochStart(message, cfg.Pacer.Committee()[2*cfg.F/3])
//
//	<-trigger
//	assert.False(t, p.IsStartingEpoch)
//	assert.Equal(t, cfg.Pacer.GetCurrent(p.GetCurrentView()), cfg.Pacer.Committee()[3]) //only first epoch starts from fourth peer and shorter than others
//}
//
//func TestPacerStartEpochWhenMissedSome(t *testing.T) {
//	_, p, cfg := initProtocol(t)
//
//	msgChan := make(chan *msg.Message)
//	(cfg.Srv).(*mocks.Service).On("Broadcast", mock.AnythingOfType("*message.Message")).Run(func(args mock.Arguments) {
//		msgChan <- (args[0]).(*msg.Message)
//	})
//
//	go func() {
//		for range cfg.ControlChan {
//
//		}
//	}()
//
//	cfg.Pacer.Bootstrap()
//	<-msgChan
//
//	for i := 0; i < 2*cfg.F/3+1; i++ {
//		epoch := hotstuff.CreateEpoch(cfg.Pacer.Committee()[i], 5, p.HQC())
//		message, _ := epoch.GetMessage()
//		p.OnEpochStart(message, cfg.Pacer.Committee()[i])
//	}
//
//	assert.False(t, p.IsStartingEpoch)
//
//	assert.Equal(t, int32(4*cfg.F), cfg.Blockchain.GetHead().Header().Height() + 1)
//}
//
//func TestStartEpochOnFPlusOneMessage(t *testing.T) {
//	_, p, cfg := initProtocol(t)
//
//	msgChan := make(chan *msg.Message)
//	(cfg.Srv).(*mocks.Service).On("Broadcast", mock.AnythingOfType("*message.Message")).Run(func(args mock.Arguments) {
//		msgChan <- (args[0]).(*msg.Message)
//	})
//
//	for i := 0; i < cfg.F/3+1; i++ {
//		epoch := hotstuff.CreateEpoch(cfg.Pacer.Committee()[i], 5, p.HQC())
//		message, _ := epoch.GetMessage()
//		p.OnEpochStart(message, cfg.Pacer.Committee()[i])
//	}
//
//	assert.True(t, p.IsStartingEpoch)
//
//	m := <-msgChan
//
//	payload := &pb.EpochStartPayload{}
//	_ = ptypes.UnmarshalAny(m.Payload, payload)
//
//	assert.Equal(t, payload.EpochNumber, int32(5))
//}
//
//func TestRoundChange(t *testing.T) {
//
//	_, _, cfg := initProtocol(t)
//
//	msgChan := make(chan *msg.Message)
//	(cfg.Srv).(*mocks.Service).On("Broadcast", mock.AnythingOfType("*message.Message")).Run(func(args mock.Arguments) {
//		msgChan <- (args[0]).(*msg.Message)
//	})
//
//	cfg.Pacer.Bootstrap()
//	<-msgChan
//
//	go cfg.Pacer.Run()
//	cfg.RoundEndChan <- 3
//
//}

func createVote(bc *blockchain.Blockchain, newBlock *blockchain.Block, t *testing.T) *hotstuff.Vote {
	vote := hotstuff.CreateVote(newBlock.Header(), bc.GetGenesisCert(), generateIdentity(t))
	return vote
}

func createVotes(count int, bc *blockchain.Blockchain, newBlock *blockchain.Block, t *testing.T) []*hotstuff.Vote {
	votes := make([]*hotstuff.Vote, count)

	for i := 0; i < count; i++ {
		votes[i] = createVote(bc, newBlock, t)
	}

	return votes
}

func initProtocol(t *testing.T) (*blockchain.Blockchain, *hotstuff.Protocol, *hotstuff.ProtocolConfig) {
	identity := generateIdentity(t)
	srv := &mocks.Service{}
	synchr := &mocks.Synchronizer{}
	loader := &mocks.CommitteeLoader{}
	bc := blockchain.CreateBlockchainFromGenesisBlock()
	bc.SetSynchronizer(synchr)
	config := &hotstuff.ProtocolConfig{
		F:               10,
		Delta:           5 * time.Second,
		Blockchain:      bc,
		Me:              identity,
		Srv:             srv,
		CommitteeLoader: loader,
		RoundEndChan:    make(chan int32),
		ControlChan:     make(chan hotstuff.Event),
	}

	peers := make([]*msg.Peer, 10)
	for i := 0; i < 10; i++ {
		peers[i] = generateIdentity(t)
	}

	loader.On("LoadFromFile").Return(peers)

	pacer := hotstuff.CreatePacer(config)
	config.Pacer = pacer
	p := hotstuff.CreateProtocol(config)

	return bc, p, config
}

func generateIdentity(t *testing.T) *msg.Peer {
	privateKey, e := crypto.GenerateKey()
	if e != nil {
		t.Errorf("failed to generate key")
	}

	var sb strings.Builder
	sb.WriteString("/ip4/1.2.3.4/tcp/")
	sb.WriteString(strconv.Itoa(mrnd.Intn(10000)))
	a := mustAddr(t, sb.String())
	sb.Reset()

	sb.WriteString("/ip4/1.2.3.4/tcp/")
	sb.WriteString(strconv.Itoa(mrnd.Intn(10000)))
	b := mustAddr(t, sb.String())

	id, err := peer.IDB58Decode("QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ")
	if err != nil {
		t.Fatal(err)
	}

	pi := &peerstore.PeerInfo{
		ID:    id,
		Addrs: []ma.Multiaddr{a, b},
	}

	return msg.CreatePeer(&privateKey.PublicKey, privateKey, pi)
}

func mustAddr(t *testing.T, s string) ma.Multiaddr {
	addr, err := ma.NewMultiaddr(s)
	if err != nil {
		t.Fatal(err)
	}

	return addr
}
