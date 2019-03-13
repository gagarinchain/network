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
	bc := blockchain.CreateBlockchainFromGenesisBlock()
	me := generateIdentity(t)
	toTest := blockchain.CreateSynchronizer(make(chan *blockchain.Block), me, srv, bc)

	head := bc.GetHead()
	newBlock := bc.NewBlock(head, bc.GetGenesisCert(), []byte(""))
	//newQC := blockchain.CreateQuorumCertificate([]byte("New QC"), newBlock.Header())
	log.Info("Head ", common.Bytes2Hex(newBlock.Header().Hash().Bytes()))

	pbBlock := bc.GetMessageForBlock(newBlock)
	any, _ := ptypes.MarshalAny(pbBlock)
	msg := message.CreateMessage(pb.Message_BLOCK_REQUEST, any)

	srv.On("SendMessageToRandomPeer", mock.AnythingOfType("*message.Message")).Return(msg)

	toTest.RequestBlockWithParent(newBlock.Header())

}

func TestSynchRequestBlocksForHeight(t *testing.T) {
	srv := &mocks.Service{}
	bc := blockchain.CreateBlockchainFromGenesisBlock()
	me := generateIdentity(t)
	blocks := make(chan *blockchain.Block)
	toTest := blockchain.CreateSynchronizer(blocks, me, srv, bc)

	head := bc.GetHead()
	block31 := bc.NewBlock(head, bc.GetGenesisCert(), []byte("newBlock31"))
	block32 := bc.NewBlock(head, bc.GetGenesisCert(), []byte("newBlock32"))
	block33 := bc.NewBlock(head, bc.GetGenesisCert(), []byte("newBlock33"))

	srv.On("SendMessageToRandomPeer", mock.MatchedBy(func(msg *message.Message) bool {
		br := &pb.BlockRequestPayload{}
		if err := ptypes.UnmarshalAny(msg.Payload, br); err != nil {
			t.Error("Can't unmarshal request payload")
		}
		return br.Height == 3
	})).Return(getMessage(block31.GetMessage(), block32.GetMessage(), block33.GetMessage())).Once()

	toTest.RequestBlocks(2, 3)

	assert.Equal(t, block31, bc.GetBlockByHash(block31.Header().Hash()))
	assert.Equal(t, block32, bc.GetBlockByHash(block32.Header().Hash()))
	assert.Equal(t, block33, bc.GetBlockByHash(block33.Header().Hash()))

}

func TestSynchRequestBlocksForHeightRange(t *testing.T) {
	srv := &mocks.Service{}
	bc := blockchain.CreateBlockchainFromGenesisBlock()
	me := generateIdentity(t)
	toTest := blockchain.CreateSynchronizer(make(chan *blockchain.Block), me, srv, bc)

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

	srv.On("SendMessageToRandomPeer", mock.MatchedBy(func(msg *message.Message) bool {
		br := &pb.BlockRequestPayload{}
		if err := ptypes.UnmarshalAny(msg.Payload, br); err != nil {
			t.Error("Can't unmarshal request payload")
		}
		return br.Height == 3
	})).Return(getMessage(block31.GetMessage(), block32.GetMessage(), block33.GetMessage())).Once()
	srv.On("SendMessageToRandomPeer", mock.MatchedBy(func(msg *message.Message) bool {
		br := &pb.BlockRequestPayload{}
		if err := ptypes.UnmarshalAny(msg.Payload, br); err != nil {
			t.Error("Can't unmarshal request payload")
		}
		return br.Height == 4
	})).Return(getMessage(block41.GetMessage(), block42.GetMessage())).Once()

	toTest.RequestBlocks(2, 4)
}

func getMessage(msgs ...*pb.Block) *message.Message {
	bpayload := &pb.BlockResponsePayload_Blocks{Blocks: &pb.Blocks{Blocks: msgs}}
	blocks3 := &pb.BlockResponsePayload{Response: bpayload}
	any, e := ptypes.MarshalAny(blocks3)
	if e != nil {
		panic("can't make payload")
	}
	return message.CreateMessage(pb.Message_HELLO_RESPONSE, any)
}
