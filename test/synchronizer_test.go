package test

import (
	"github.com/golang/protobuf/ptypes"
	"github.com/poslibp2p/blockchain"
	"github.com/poslibp2p/eth/common"
	"github.com/poslibp2p/message"
	"github.com/poslibp2p/message/protobuff"
	"github.com/poslibp2p/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestSynchRequestBlock(t *testing.T) {
	srv := &mocks.Service{}
	bc := blockchain.CreateBlockchainFromGenesisBlock(mockStorage(), nil)

	toTest := blockchain.NewBlockService(srv)
	head := bc.GetHead()
	newBlock := bc.NewBlock(head, bc.GetGenesisCert(), []byte(""))
	//newQC := blockchain.CreateQuorumCertificate([]byte("New QC"), newBlock.Header())
	log.Info("Head ", common.Bytes2Hex(newBlock.Header().Hash().Bytes()))

	pbBlock := newBlock.GetMessage()
	any, _ := ptypes.MarshalAny(pbBlock)
	msgChan := make(chan *message.Message)
	go func() {
		msgChan <- message.CreateMessage(pb.Message_BLOCK_REQUEST, any, nil)
		close(msgChan)
	}()
	srv.On("SendRequestToRandomPeer", mock.AnythingOfType("*message.Message")).Return(msgChan)

	toTest.RequestBlock(newBlock.Header().Hash())

}

func TestSynchRequestBlocksForHeight(t *testing.T) {
	srv := &mocks.Service{}
	bc := blockchain.CreateBlockchainFromGenesisBlock(mockStorage(), nil)
	bsrv := blockchain.NewBlockService(srv)
	me := generateIdentity(t)
	blocks := make(chan *blockchain.Block)
	toTest := blockchain.CreateSynchronizer(blocks, me, bsrv, bc)

	head := bc.GetHead()
	block31 := bc.NewBlock(head, bc.GetGenesisCert(), []byte("newBlock31"))
	block32 := bc.NewBlock(head, bc.GetGenesisCert(), []byte("newBlock32"))
	block33 := bc.NewBlock(head, bc.GetGenesisCert(), []byte("newBlock33"))

	srv.On("SendRequestToRandomPeer", mock.MatchedBy(func(msg *message.Message) bool {
		br := &pb.BlockRequestPayload{}
		if err := ptypes.UnmarshalAny(msg.Payload, br); err != nil {
			t.Error("Can't unmarshal request payload")
		}
		log.Info(br.Height)
		return br.Height == 3
	})).Return(getMessage(block31.GetMessage(), block32.GetMessage(), block33.GetMessage())).Once()

	toTest.RequestBlocks(2, 3)

	assert.Equal(t, block31, bc.GetBlockByHash(block31.Header().Hash()))
	assert.Equal(t, block32, bc.GetBlockByHash(block32.Header().Hash()))
	assert.Equal(t, block33, bc.GetBlockByHash(block33.Header().Hash()))

}

func TestSynchRequestBlocksForHeightRange(t *testing.T) {
	srv := &mocks.Service{}
	bc := blockchain.CreateBlockchainFromGenesisBlock(mockStorage(), nil)
	me := generateIdentity(t)
	bsrv := blockchain.NewBlockService(srv)
	toTest := blockchain.CreateSynchronizer(make(chan *blockchain.Block), me, bsrv, bc)

	head := bc.GetHead()
	block31 := bc.NewBlock(head, bc.GetGenesisCert(), []byte("newBlock31"))
	_ = bc.AddBlock(block31)
	block32 := bc.NewBlock(head, bc.GetGenesisCert(), []byte("newBlock32"))
	_ = bc.AddBlock(block32)
	block33 := bc.NewBlock(head, bc.GetGenesisCert(), []byte("newBlock33"))
	_ = bc.AddBlock(block33)

	block41 := bc.NewBlock(block31, bc.GetGenesisCert(), []byte("newBlock41"))
	_ = bc.AddBlock(block41)
	block42 := bc.NewBlock(block32, bc.GetGenesisCert(), []byte("newBlock42"))
	_ = bc.AddBlock(block42)

	srv.On("SendRequestToRandomPeer", mock.MatchedBy(func(msg *message.Message) bool {
		br := &pb.BlockRequestPayload{}
		if err := ptypes.UnmarshalAny(msg.Payload, br); err != nil {
			t.Error("Can't unmarshal request payload")
		}
		return br.Height == 3
	})).Return(getMessage(block31.GetMessage(), block32.GetMessage(), block33.GetMessage())).Once()
	srv.On("SendRequestToRandomPeer", mock.MatchedBy(func(msg *message.Message) bool {
		br := &pb.BlockRequestPayload{}
		if err := ptypes.UnmarshalAny(msg.Payload, br); err != nil {
			t.Error("Can't unmarshal request payload")
		}
		return br.Height == 4
	})).Return(getMessage(block41.GetMessage(), block42.GetMessage())).Once()

	toTest.RequestBlocks(2, 4)
}

func getMessage(msgs ...*pb.Block) chan *message.Message {
	resChan := make(chan *message.Message)
	go func() {
		bpayload := &pb.BlockResponsePayload_Blocks{Blocks: &pb.Blocks{Blocks: msgs}}
		blocks3 := &pb.BlockResponsePayload{Response: bpayload}
		any, e := ptypes.MarshalAny(blocks3)
		if e != nil {
			panic("can't make payload")
		}
		resChan <- message.CreateMessage(pb.Message_BLOCK_RESPONSE, any, nil)
		close(resChan)
	}()

	return resChan
}
