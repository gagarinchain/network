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
)

var log = logging.MustGetLogger("hotstuff")

func TestProtocolProposeOnGenesisBlockchain(t *testing.T) {
	_, p, cfg := initProtocol(t)
	mocksrv := (cfg.Srv).(*mocks.Service)

	log.Info("BlockProtocol ", p)

	mocksrv.On("Broadcast", mock.AnythingOfType("*message.Message")).Run(func(args mock.Arguments) {
		assert.Equal(t, pb.Message_PROPOSAL, args[0].(*msg.Message).Type)
	}).Once()

	p.OnPropose([]byte("proposing something"))

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
	if e := bc.AddBlock(newBlock); e != nil {
		t.Error("can't add block", e)
	}

	proposal := &hotstuff.Proposal{Sender: cfg.CurrentProposer, NewBlock: newBlock, HQC: head.QC()}
	srv := (cfg.Srv).(*mocks.Service)
	var vote *hotstuff.Vote
	srv.On("SendMessage", cfg.CurrentProposer, mock.AnythingOfType("*message.Message")).Run(func(args mock.Arguments) {
		m := (args[1]).(*msg.Message)
		var err error
		if vote, err = hotstuff.CreateVoteFromMessage(m, newBlock, cfg.Me); err != nil {
			t.Error("can't create vote", err)
		}
	}).Once()

	if err := p.OnReceiveProposal(proposal); err != nil {
		t.Error("Error while receiving proposal", err)
	}

	srv.AssertCalled(t, "SendMessage", cfg.CurrentProposer, mock.AnythingOfType("*message.Message"))
	assert.Equal(t, proposal.NewBlock.Header().Hash(), vote.NewBlock.Header().Hash())
	assert.Equal(t, vote.NewBlock.Header().Height(), p.Vheight())

}

func TestOnReceiveProposalFromWrongProposer(t *testing.T) {
	bc, p, cfg := initProtocol(t)
	head := bc.GetHead()
	newBlock := bc.NewBlock(bc.GetHead(), bc.GetGenesisCert(), []byte("wonderful block"))
	if e := bc.AddBlock(newBlock); e != nil {
		t.Error("can't add block", e)
	}

	proposal := &hotstuff.Proposal{Sender: cfg.NextProposer, NewBlock: newBlock, HQC: head.QC()}

	assert.Error(t, p.OnReceiveProposal(proposal), "peer equivocated")
	assert.Equal(t, int32(2), p.Vheight())
}

func TestOnReceiveVoteForNotProposer(t *testing.T) {
	bc, p, cfg := initProtocol(t)
	p.SetCurrentProposer(cfg.NextProposer)

	newBlock := bc.NewBlock(bc.GetHead(), bc.GetGenesisCert(), []byte("wonderful block"))
	vote := createVote(bc, newBlock, t)

	if err := p.OnReceiveVote(vote); err != nil {
		t.Error("failed OnReceive", err)
	}
}

func TestOnReceiveTwoVotesSamePeer(t *testing.T) {
	bc, p, cfg := initProtocol(t)
	p.SetCurrentProposer(cfg.Me)

	id := generateIdentity(t)
	newBlock1 := bc.NewBlock(bc.GetHead(), bc.GetGenesisCert(), []byte("wonderful block"))
	newBlock2 := bc.NewBlock(bc.GetHead(), bc.GetGenesisCert(), []byte("another wonderful block"))
	vote1 := hotstuff.CreateVote(newBlock1, bc.GetGenesisCert(), id)
	vote2 := hotstuff.CreateVote(newBlock2, bc.GetGenesisCert(), id)

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
	p.SetCurrentProposer(cfg.Me)

	newBlock := bc.NewBlock(bc.GetHead(), bc.GetGenesisCert(), []byte("wonderful block"))
	if e := bc.AddBlock(newBlock); e != nil {
		t.Error("can't add block", e)
	}

	votes := createVotes((cfg.F/3)*2+1, bc, newBlock, t)

	for _, vote := range votes {
		if err := p.OnReceiveVote(vote); err != nil {
			t.Error("failed OnReceive", err)
		}
	}

	assert.Equal(t, newBlock.Header().Hash(), p.HQC().QrefBlock().Hash())

}

func createVote(bc *blockchain.Blockchain, newBlock *blockchain.Block, t *testing.T) *hotstuff.Vote {
	vote := hotstuff.CreateVote(newBlock, bc.GetGenesisCert(), generateIdentity(t))
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
	current := generateIdentity(t)
	next := generateIdentity(t)
	srv := &mocks.Service{}
	synchr := &mocks.Synchronizer{}
	bc := blockchain.CreateBlockchainFromGenesisBlock()
	bc.SetSynchronizer(synchr)
	config := &hotstuff.ProtocolConfig{
		F:               10,
		Blockchain:      bc,
		Me:              identity,
		CurrentProposer: current,
		NextProposer:    next,
		Srv:             srv,
	}
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
