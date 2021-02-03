package test

import (
	"context"
	"github.com/gagarinchain/common/api"
	"github.com/gagarinchain/common/eth/crypto"
	msg "github.com/gagarinchain/common/message"
	"github.com/gagarinchain/common/protobuff"
	"github.com/gagarinchain/network/blockchain"
	"github.com/gagarinchain/network/mocks"
	"github.com/golang/protobuf/ptypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

//Test that we request hello and than load missed blocks
func TestBlockProtocolBootstrap(t *testing.T) {
	srv := &mocks.Service{}
	synchr := &mocks.Synchronizer{}
	storage := SoftStorageMock()
	bpersister := &blockchain.BlockPersister{storage}
	cpersister := &blockchain.BlockchainPersister{storage}
	bc := blockchain.CreateBlockchainFromGenesisBlock(&blockchain.BlockchainConfig{
		BlockPerister: bpersister, ChainPersister: cpersister, Pool: mockPool(), Db: mockDB(), ProposerGetter: MockProposerForHeight(),
	})
	bc.GetGenesisBlock().SetQC(blockchain.CreateQuorumCertificate(crypto.EmptyAggregateSignatures(), bc.GetGenesisBlock().Header(), api.QRef))
	p := blockchain.CreateBlockProtocol(srv, bc, synchr)

	payload := &pb.HelloPayload{Version: 1, Time: time.Now().Unix(), TopBlockHeight: 4}
	any, e := ptypes.MarshalAny(payload)
	if e != nil {
		t.Error("Error constructing payload", e)
	}
	resp := make(chan *msg.Message)
	go func() {
		resp <- &msg.Message{Message: &pb.Message{
			Type:    pb.Message_HELLO_RESPONSE,
			Payload: any,
		}}
		close(resp)
	}()

	srv.On("SendRequestToRandomPeer", mock.MatchedBy(func(ctx context.Context) bool { return true }),
		mock.AnythingOfType("*message.Message")).Run(func(args mock.Arguments) {
		m := (args[1]).(*msg.Message)
		if m.Type != pb.Message_HELLO_REQUEST {
			t.Error("Wrong message type")
		}
	}).Once().Return(resp, nil)

	synchr.On("LoadBlocks", mock.MatchedBy(func(ctx context.Context) bool { return true }),
		mock.AnythingOfType("int32"), mock.AnythingOfType("int32"), mock.AnythingOfType("*common.Peer")).Run(func(args mock.Arguments) {
		low := (args[1]).(int32)
		high := (args[2]).(int32)
		assert.True(t, low == 0 && high == 4)
	}).Return(nil)

	respChan, _ := p.Bootstrap(context.Background())

	i := <-respChan
	assert.Equal(t, 0, i)
}

func TestBlockProtocolOnHello(t *testing.T) {
	srv := &mocks.Service{}
	synchr := &mocks.Synchronizer{}
	storage := SoftStorageMock()
	bpersister := &blockchain.BlockPersister{storage}
	cpersister := &blockchain.BlockchainPersister{storage}
	bc := blockchain.CreateBlockchainFromGenesisBlock(&blockchain.BlockchainConfig{
		BlockPerister: bpersister, ChainPersister: cpersister, Pool: mockPool(), Db: mockDB(), ProposerGetter: MockProposerForHeight(),
	})
	bc.GetGenesisBlock().SetQC(blockchain.CreateQuorumCertificate(crypto.EmptyAggregateSignatures(), bc.GetGenesisBlock().Header(), api.QRef))
	p := blockchain.CreateBlockProtocol(srv, bc, synchr)

	block12, _ := bc.NewBlock(bc.GetHead(), bc.GetGenesisCert(), []byte("block 1<-2"))
	_, _ = bc.AddBlock(block12)
	block23, _ := bc.NewBlock(bc.GetHead(), bc.GetGenesisCert(), []byte("block 2<-3"))
	_, _ = bc.AddBlock(block23)
	block34, _ := bc.NewBlock(bc.GetHead(), bc.GetGenesisCert(), []byte("block 3<-4"))
	_, _ = bc.AddBlock(block34)
	block45, _ := bc.NewBlock(block34, bc.GetGenesisCert(), []byte("block 4<-5"))
	_, _ = bc.AddBlock(block45)
	block56, _ := bc.NewBlock(block45, bc.GetGenesisCert(), []byte("block 5<-6"))
	_, _ = bc.AddBlock(block56)
	block47, _ := bc.NewBlock(block34, bc.GetGenesisCert(), []byte("block 4<-7"))
	_, _ = bc.AddBlock(block47)
	bc.OnCommit(block34)

	resp := make(chan *msg.Message)
	close(resp)

	peer := generateIdentity(t, 0)
	m := msg.CreateMessage(pb.Message_HELLO_REQUEST, nil, peer)
	synchr.On("LoadBlocks", mock.MatchedBy(func(ctx context.Context) bool { return true }),
		0, 4, mock.AnythingOfType("*common.Peer")).Return(nil)

	srv.On("SendResponse", mock.MatchedBy(func(ctx context.Context) bool { return true }),
		mock.AnythingOfType("*message.Message")).Run(func(args mock.Arguments) {
		m := (args[1]).(*msg.Message)
		if m.Type != pb.Message_HELLO_RESPONSE {
			t.Error("Wrong message type")
		}

		payload := &pb.HelloPayload{}
		ptypes.UnmarshalAny(m.Payload, payload)
		assert.Equal(t, int32(5), payload.TopBlockHeight)
	}).Once().Return(resp, nil)

	p.OnHello(context.Background(), m)

}
