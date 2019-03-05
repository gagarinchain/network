package test

import (
	"github.com/libp2p/go-libp2p-peer"
	"github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/op/go-logging"
	"github.com/poslibp2p/blockchain"
	"github.com/poslibp2p/eth/common"
	"github.com/poslibp2p/eth/crypto"
	"github.com/poslibp2p/hotstuff"
	msg "github.com/poslibp2p/message"
	"github.com/poslibp2p/message/protobuff"
	"github.com/poslibp2p/mocks"
	"github.com/poslibp2p/network"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	mrnd "math/rand"
	"strconv"
	"strings"
	"testing"
)

var log = logging.MustGetLogger("hotstuff")

func TestProtocolProposeOnGenesisBlockchain(t *testing.T) {
	identity := generateIdentity(t)
	current := generateIdentity(t)
	next := generateIdentity(t)
	mockSrv := &mocks.Service{}
	mockSynch := &mocks.Synchronizer{}
	var srv network.Service = mockSrv

	config := &hotstuff.ProtocolConfig{
		F:               10,
		Blockchain:      blockchain.CreateBlockchainFromGenesisBlock(),
		Me:              identity,
		CurrentProposer: current,
		NextProposer:    next,
		Srv:             srv,
		Synchronizer:    mockSynch,
	}

	p := hotstuff.CreateProtocol(config)
	log.Info("Protocol ", p)

	mockSrv.On("Broadcast", mock.AnythingOfType("*message.Message")).Run(func(args mock.Arguments) {
		assert.Equal(t, pb.Message_PROPOSAL, args[0].(*msg.Message).Type)
	}).Once()

	p.OnPropose([]byte("proposing something"))

	mockSrv.AssertCalled(t, "Broadcast", mock.AnythingOfType("*message.Message"))
}

func TestProtocolUpdateCertificate(t *testing.T) {
	identity := generateIdentity(t)
	current := generateIdentity(t)
	next := generateIdentity(t)
	srv := &mocks.Service{}
	synchr := &mocks.Synchronizer{}

	bc := blockchain.CreateBlockchainFromGenesisBlock()
	config := &hotstuff.ProtocolConfig{
		F:               10,
		Blockchain:      bc,
		Me:              identity,
		CurrentProposer: current,
		NextProposer:    next,
		Srv:             srv,
		Synchronizer:    synchr,
	}

	p := hotstuff.CreateProtocol(config)

	newBlock := bc.NewBlock(bc.GetHead(), bc.GetGenesisCert(), []byte(""))
	log.Info("Head ", common.Bytes2Hex(newBlock.Header().Hash().Bytes()))
	newQC := blockchain.CreateQuorumCertificate([]byte("New QC"), newBlock.Header())

	synchr.On("RequestBlockWithParent", mock.AnythingOfType("*blockchain.Header")).Run(func(args mock.Arguments) {
		assert.Equal(t, newQC.QrefBlock(), args[0].(*blockchain.Header))
	}).Once()

	p.Update(newQC)
	hqc := p.HQC()

	assert.Equal(t, newQC, hqc)
}

func generateIdentity(t *testing.T) *network.Peer {
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

	return network.CreatePeer(&privateKey.PublicKey, privateKey, pi)
}

func mustAddr(t *testing.T, s string) ma.Multiaddr {
	addr, err := ma.NewMultiaddr(s)
	if err != nil {
		t.Fatal(err)
	}

	return addr
}
