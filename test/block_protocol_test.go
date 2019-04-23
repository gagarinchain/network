package test

import (
	"context"
	"github.com/golang/protobuf/ptypes"
	"github.com/poslibp2p/blockchain"
	msg "github.com/poslibp2p/common/message"
	"github.com/poslibp2p/common/protobuff"
	"github.com/poslibp2p/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

//Test that we request hello and than load missed blocks
func TestBlockProtocolBootstrap(t *testing.T) {
	srv := &mocks.Service{}
	synchr := &mocks.Synchronizer{}
	storage := initStorage()
	bsrv := &mocks.BlockService{}
	bc := blockchain.CreateBlockchainFromGenesisBlock(storage, bsrv)
	bc.GetGenesisBlock().SetQC(blockchain.CreateQuorumCertificate([]byte("valid"), bc.GetGenesisBlock().Header()))
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

	synchr.On("RequestBlocks", mock.MatchedBy(func(ctx context.Context) bool { return true }),
		mock.AnythingOfType("int32"), mock.AnythingOfType("int32"), mock.AnythingOfType("*common.Peer")).Run(func(args mock.Arguments) {
		low := (args[1]).(int32)
		high := (args[2]).(int32)

		assert.True(t, low == 0 && high == 4)
	}).Return(nil)

	p.Bootstrap(context.Background())
}

func TestBlockProtocolOnBlockRequest(t *testing.T) {
	srv := &mocks.Service{}
	synchr := &mocks.Synchronizer{}
	storage := initStorage()
	bsrv := &mocks.BlockService{}
	bc := blockchain.CreateBlockchainFromGenesisBlock(storage, bsrv)
	bc.GetGenesisBlock().SetQC(blockchain.CreateQuorumCertificate([]byte("valid"), bc.GetGenesisBlock().Header()))
	p := blockchain.CreateBlockProtocol(srv, bc, synchr)

	block12 := bc.NewBlock(bc.GetHead(), bc.GetGenesisCert(), []byte("block 1<-2"))
	_ = bc.AddBlock(block12)
	block23 := bc.NewBlock(bc.GetHead(), bc.GetGenesisCert(), []byte("block 2<-3"))
	_ = bc.AddBlock(block23)
	block34 := bc.NewBlock(bc.GetHead(), bc.GetGenesisCert(), []byte("block 3<-4"))
	_ = bc.AddBlock(block34)
	block45 := bc.NewBlock(block34, bc.GetGenesisCert(), []byte("block 4<-5"))
	_ = bc.AddBlock(block45)
	block56 := bc.NewBlock(block45, bc.GetGenesisCert(), []byte("block 5<-6"))
	_ = bc.AddBlock(block56)
	block47 := bc.NewBlock(block34, bc.GetGenesisCert(), []byte("block 4<-7"))
	_ = bc.AddBlock(block47)

	peer := generateIdentity(t, 0)
	msgChan := make(chan *blockchain.Block)
	resp := make(chan *msg.Message)
	close(resp)
	srv.On("SendResponse", mock.MatchedBy(func(ctx context.Context) bool { return true }),
		mock.AnythingOfType("*message.Message")).Run(func(args mock.Arguments) {
		m := (args[1]).(*msg.Message)
		if m.Type != pb.Message_BLOCK_RESPONSE {
			t.Error("Wrong message type")
		}
		payload := &pb.BlockResponsePayload{}
		ptypes.UnmarshalAny(m.Payload, payload)
		v, ok := (payload.GetResponse()).(*pb.BlockResponsePayload_Blocks)
		if !ok {
			t.Error("Got error in response")
		}

		for _, b := range v.Blocks.Blocks {
			msgChan <- blockchain.CreateBlockFromMessage(b)
		}

	}).Return(resp, nil)

	go func() {
		any, _ := ptypes.MarshalAny(&pb.BlockRequestPayload{Height: int32(3)})
		p.OnBlockRequest(context.Background(), msg.CreateMessage(pb.Message_BLOCK_REQUEST, any, peer))
	}()
	assert.Equal(t, block34, <-msgChan)

	go func() {
		any, _ := ptypes.MarshalAny(&pb.BlockRequestPayload{Height: int32(4)})
		p.OnBlockRequest(context.Background(), msg.CreateMessage(pb.Message_BLOCK_REQUEST, any, peer))
	}()
	block41 := <-msgChan
	if block41.Header().Hash() == block47.Header().Hash() {
		assert.Equal(t, block47, block41)
		assert.Equal(t, block45, <-msgChan)
	} else {
		assert.Equal(t, block47, <-msgChan)
		assert.Equal(t, block45, block41)
	}

	go func() {
		any, _ := ptypes.MarshalAny(&pb.BlockRequestPayload{Height: int32(5)})
		p.OnBlockRequest(context.Background(), msg.CreateMessage(pb.Message_BLOCK_REQUEST, any, peer))
	}()
	assert.Equal(t, block56, <-msgChan)
}

func TestBlockProtocolOnHello(t *testing.T) {
	srv := &mocks.Service{}
	synchr := &mocks.Synchronizer{}
	bsrv := &mocks.BlockService{}
	storage := initStorage()

	bc := blockchain.CreateBlockchainFromGenesisBlock(storage, bsrv)
	bc.GetGenesisBlock().SetQC(blockchain.CreateQuorumCertificate([]byte("valid"), bc.GetGenesisBlock().Header()))
	p := blockchain.CreateBlockProtocol(srv, bc, synchr)

	block12 := bc.NewBlock(bc.GetHead(), bc.GetGenesisCert(), []byte("block 1<-2"))
	_ = bc.AddBlock(block12)
	block23 := bc.NewBlock(bc.GetHead(), bc.GetGenesisCert(), []byte("block 2<-3"))
	_ = bc.AddBlock(block23)
	block34 := bc.NewBlock(bc.GetHead(), bc.GetGenesisCert(), []byte("block 3<-4"))
	_ = bc.AddBlock(block34)
	block45 := bc.NewBlock(block34, bc.GetGenesisCert(), []byte("block 4<-5"))
	_ = bc.AddBlock(block45)
	block56 := bc.NewBlock(block45, bc.GetGenesisCert(), []byte("block 5<-6"))
	_ = bc.AddBlock(block56)
	block47 := bc.NewBlock(block34, bc.GetGenesisCert(), []byte("block 4<-7"))
	_ = bc.AddBlock(block47)
	bc.OnCommit(block34)

	resp := make(chan *msg.Message)
	close(resp)

	peer := generateIdentity(t, 0)
	m := msg.CreateMessage(pb.Message_HELLO_REQUEST, nil, peer)
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

func initStorage() *mocks.Storage {
	storage := &mocks.Storage{}
	storage.On("PutBlock", mock.AnythingOfType("*blockchain.Block")).Return(nil)
	storage.On("GetBlock", mock.AnythingOfType("common.Hash")).Return(nil, nil)
	storage.On("Contains", mock.AnythingOfType("common.Hash")).Return(false)
	storage.On("PutCurrentTopHeight", mock.AnythingOfType("int32")).Return(nil)
	storage.On("GetTopCommittedHeight").Return(0)
	storage.On("PutTopCommittedHeight", mock.AnythingOfType("int32")).Return(nil)
	return storage
}
