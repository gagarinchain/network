package test

import (
	"github.com/golang/protobuf/ptypes"
	"github.com/poslibp2p/blockchain"
	msg "github.com/poslibp2p/message"
	"github.com/poslibp2p/message/protobuff"
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
	storage := &mocks.Storage{}
	bsrv := &mocks.BlockService{}
	storage.On("PutBlock", mock.AnythingOfType("*blockchain.Block")).Return(nil)
	storage.On("GetBlock", mock.AnythingOfType("common.Hash")).Return(nil, nil)
	storage.On("Contains", mock.AnythingOfType("common.Hash")).Return(false)
	bc := blockchain.CreateBlockchainFromGenesisBlock(storage, bsrv)
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

	srv.On("SendRequestToRandomPeer", mock.AnythingOfType("*message.Message")).Run(func(args mock.Arguments) {
		m := (args[0]).(*msg.Message)
		if m.Type != pb.Message_HELLO_REQUEST {
			t.Error("Wrong message type")
		}
	}).Once().Return(resp)

	synchr.On("RequestBlocks", mock.AnythingOfType("int32"), mock.AnythingOfType("int32")).Run(func(args mock.Arguments) {
		low := (args[0]).(int32)
		high := (args[1]).(int32)

		assert.True(t, low == 2 && high == 4)
	})

	p.Bootstrap()
}

func TestBlockProtocolOnBlockRequest(t *testing.T) {
	srv := &mocks.Service{}
	synchr := &mocks.Synchronizer{}
	storage := &mocks.Storage{}
	bsrv := &mocks.BlockService{}
	storage.On("PutBlock", mock.AnythingOfType("*blockchain.Block")).Return(nil)
	storage.On("GetBlock", mock.AnythingOfType("common.Hash")).Return(nil, nil)
	storage.On("Contains", mock.AnythingOfType("common.Hash")).Return(false)
	bc := blockchain.CreateBlockchainFromGenesisBlock(storage, bsrv)
	p := blockchain.CreateBlockProtocol(srv, bc, synchr)

	block34 := bc.NewBlock(bc.GetHead(), bc.GetGenesisCert(), []byte("block 3<-4"))
	_ = bc.AddBlock(block34)
	block45 := bc.NewBlock(block34, bc.GetGenesisCert(), []byte("block 4<-5"))
	_ = bc.AddBlock(block45)
	block56 := bc.NewBlock(block45, bc.GetGenesisCert(), []byte("block 5<-6"))
	_ = bc.AddBlock(block56)
	block47 := bc.NewBlock(block34, bc.GetGenesisCert(), []byte("block 4<-7"))
	_ = bc.AddBlock(block47)
	bc.OnCommit(block34)

	peer := generateIdentity(t)
	msgChan := make(chan *blockchain.Block)
	srv.On("SendMessage", peer, mock.AnythingOfType("*message.Message")).Run(func(args mock.Arguments) {
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

	}).Return(nil)

	go func() {
		any, _ := ptypes.MarshalAny(&pb.BlockRequestPayload{Height: int32(3)})
		p.OnBlockRequest(msg.CreateMessage(pb.Message_BLOCK_REQUEST, any, peer))
	}()
	assert.Equal(t, block34, <-msgChan)

	go func() {
		any, _ := ptypes.MarshalAny(&pb.BlockRequestPayload{Height: int32(4)})
		p.OnBlockRequest(msg.CreateMessage(pb.Message_BLOCK_REQUEST, any, peer))
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
		p.OnBlockRequest(msg.CreateMessage(pb.Message_BLOCK_REQUEST, any, peer))
	}()
	assert.Equal(t, block56, <-msgChan)
}

func TestBlockProtocolOnHello(t *testing.T) {
	srv := &mocks.Service{}
	synchr := &mocks.Synchronizer{}
	storage := &mocks.Storage{}
	bsrv := &mocks.BlockService{}
	storage.On("PutBlock", mock.AnythingOfType("*blockchain.Block")).Return(nil)
	storage.On("GetBlock", mock.AnythingOfType("common.Hash")).Return(nil, nil)
	storage.On("Contains", mock.AnythingOfType("common.Hash")).Return(false)
	bc := blockchain.CreateBlockchainFromGenesisBlock(storage, bsrv)
	p := blockchain.CreateBlockProtocol(srv, bc, synchr)

	block34 := bc.NewBlock(bc.GetHead(), bc.GetGenesisCert(), []byte("block 3<-4"))
	_ = bc.AddBlock(block34)
	block45 := bc.NewBlock(block34, bc.GetGenesisCert(), []byte("block 4<-5"))
	_ = bc.AddBlock(block45)
	block56 := bc.NewBlock(block45, bc.GetGenesisCert(), []byte("block 5<-6"))
	_ = bc.AddBlock(block56)
	block47 := bc.NewBlock(block34, bc.GetGenesisCert(), []byte("block 4<-7"))
	_ = bc.AddBlock(block47)
	bc.OnCommit(block34)

	peer := generateIdentity(t)
	m := msg.CreateMessage(pb.Message_HELLO_REQUEST, nil, peer)
	srv.On("SendMessage", peer, mock.AnythingOfType("*message.Message")).Run(func(args mock.Arguments) {
		m := (args[1]).(*msg.Message)
		if m.Type != pb.Message_HELLO_RESPONSE {
			t.Error("Wrong message type")
		}

		payload := &pb.HelloPayload{}
		ptypes.UnmarshalAny(m.Payload, payload)
		assert.Equal(t, int32(5), payload.TopBlockHeight)
	}).Once().Return(nil)

	p.OnHello(m)

}
