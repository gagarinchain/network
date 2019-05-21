package test

import (
	"context"
	"github.com/golang/protobuf/ptypes"
	"github.com/poslibp2p/blockchain"
	"github.com/poslibp2p/common/eth/common"
	"github.com/poslibp2p/common/message"
	"github.com/poslibp2p/common/protobuff"
	"github.com/poslibp2p/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

func TestSynchRequestBlock(t *testing.T) {
	srv := &mocks.Service{}
	bc := blockchain.CreateBlockchainFromGenesisBlock(&blockchain.Config{Storage: mockStorage(), Pool: mockPool(), Db: mockDB()})
	bc.GetGenesisBlock().SetQC(blockchain.CreateQuorumCertificate([]byte("valid"), bc.GetGenesisBlock().Header()))

	toTest := blockchain.NewBlockService(srv)
	head := bc.GetHead()
	newBlock := bc.NewBlock(head, bc.GetGenesisCert(), []byte(""))
	log.Info("Head ", common.Bytes2Hex(newBlock.Header().Hash().Bytes()))

	pbBlock := newBlock.GetMessage()
	any, _ := ptypes.MarshalAny(pbBlock)
	msgChan := make(chan *message.Message)
	go func() {
		msgChan <- message.CreateMessage(pb.Message_BLOCK_REQUEST, any, nil)
		close(msgChan)
	}()
	srv.On("SendRequestToRandomPeer", mock.MatchedBy(func(ctx context.Context) bool { return true }),
		mock.AnythingOfType("*message.Message")).Return(msgChan, nil)

	toTest.RequestBlock(context.Background(), newBlock.Header().Hash(), nil)

}

func TestSynchRequestBlocksForHeight(t *testing.T) {
	srv := &mocks.Service{}
	bc := blockchain.CreateBlockchainFromGenesisBlock(&blockchain.Config{Storage: mockStorage(), Pool: mockPool(), Db: mockDB()})
	bc.GetGenesisBlock().SetQC(blockchain.CreateQuorumCertificate([]byte("valid"), bc.GetGenesisBlock().Header()))

	bsrv := blockchain.NewBlockService(srv)
	me := generateIdentity(t, 0)
	toTest := blockchain.CreateSynchronizer(me, bsrv, bc)

	block11 := bc.NewBlock(bc.GetGenesisBlock(), bc.GetGenesisCert(), []byte("newBlock11"))
	_ = bc.AddBlock(block11)
	block21 := bc.NewBlock(block11, bc.GetGenesisCert(), []byte("newBlock21"))
	_ = bc.AddBlock(block21)
	block31 := bc.NewBlock(block21, bc.GetGenesisCert(), []byte("newBlock31"))
	block32 := bc.NewBlock(block21, bc.GetGenesisCert(), []byte("newBlock32"))
	block33 := bc.NewBlock(block21, bc.GetGenesisCert(), []byte("newBlock33"))

	srv.On("SendRequestToRandomPeer", mock.MatchedBy(func(ctx context.Context) bool { return true }), mock.MatchedBy(func(msg *message.Message) bool {
		br := &pb.BlockRequestPayload{}
		if err := ptypes.UnmarshalAny(msg.Payload, br); err != nil {
			t.Error("Can't unmarshal request payload")
		}
		return br.Height == 3
	})).Return(getMessage(block31.GetMessage(), block32.GetMessage(), block33.GetMessage()), nil).Once()

	e := toTest.RequestBlocks(context.Background(), 2, 3, nil)
	assert.Nil(t, e)

	assert.Equal(t, block31, bc.GetBlockByHash(block31.Header().Hash()))
	assert.Equal(t, block32, bc.GetBlockByHash(block32.Header().Hash()))
	assert.Equal(t, block33, bc.GetBlockByHash(block33.Header().Hash()))
}

func TestSynchRequestBlocksForWrongHeight(t *testing.T) {
	srv := &mocks.Service{}
	bc := blockchain.CreateBlockchainFromGenesisBlock(&blockchain.Config{Storage: mockStorage(), Pool: mockPool(), Db: mockDB()})
	bc.GetGenesisBlock().SetQC(blockchain.CreateQuorumCertificate([]byte("valid"), bc.GetGenesisBlock().Header()))

	bsrv := blockchain.NewBlockService(srv)
	me := generateIdentity(t, 0)
	toTest := blockchain.CreateSynchronizer(me, bsrv, bc)

	block31 := bc.NewBlock(bc.GetGenesisBlock(), bc.GetGenesisCert(), []byte("newBlock31"))
	block32 := bc.NewBlock(bc.GetGenesisBlock(), bc.GetGenesisCert(), []byte("newBlock32"))
	block33 := bc.NewBlock(bc.GetGenesisBlock(), bc.GetGenesisCert(), []byte("newBlock33"))

	srv.On("SendRequestToRandomPeer", mock.MatchedBy(func(ctx context.Context) bool { return true }), mock.MatchedBy(func(msg *message.Message) bool {
		br := &pb.BlockRequestPayload{}
		if err := ptypes.UnmarshalAny(msg.Payload, br); err != nil {
			t.Error("Can't unmarshal request payload")
		}
		log.Info(br.Height)
		return br.Height == 3
	})).Return(getMessage(block31.GetMessage(), block32.GetMessage(), block33.GetMessage()), nil).Once()

	e := toTest.RequestBlocks(context.Background(), 2, 3, nil)
	assert.Error(t, e)
}

func TestSynchRequestBlocksForHeightRange(t *testing.T) {
	srv := &mocks.Service{}
	bc := blockchain.CreateBlockchainFromGenesisBlock(&blockchain.Config{Storage: mockStorage(), Pool: mockPool(), Db: mockDB()})
	bc.GetGenesisBlock().SetQC(blockchain.CreateQuorumCertificate([]byte("valid"), bc.GetGenesisBlock().Header()))

	me := generateIdentity(t, 0)
	bsrv := blockchain.NewBlockService(srv)
	toTest := blockchain.CreateSynchronizer(me, bsrv, bc)

	block11 := bc.NewBlock(bc.GetGenesisBlock(), bc.GetGenesisCert(), []byte("newBlock11"))
	_ = bc.AddBlock(block11)
	block21 := bc.PadEmptyBlock(block11)
	_ = bc.AddBlock(block21)

	block31 := bc.NewBlock(block21, bc.GetGenesisCert(), []byte("newBlock31"))
	block32 := bc.NewBlock(block21, bc.GetGenesisCert(), []byte("newBlock32"))
	block33 := bc.NewBlock(block21, bc.GetGenesisCert(), []byte("newBlock33"))

	block41 := bc.NewBlock(block31, bc.GetGenesisCert(), []byte("newBlock41"))
	block42 := bc.NewBlock(block32, bc.GetGenesisCert(), []byte("newBlock42"))

	srv.On("SendRequestToRandomPeer", mock.MatchedBy(func(ctx context.Context) bool { return true }),
		mock.MatchedBy(func(msg *message.Message) bool {
			br := &pb.BlockRequestPayload{}
			if err := ptypes.UnmarshalAny(msg.Payload, br); err != nil {
				t.Error("Can't unmarshal request payload")
			}
			return br.Height == 3
		})).Return(getMessage(block31.GetMessage(), block32.GetMessage(), block33.GetMessage()), nil).Once()
	srv.On("SendRequestToRandomPeer", mock.MatchedBy(func(ctx context.Context) bool { return true }),
		mock.MatchedBy(func(msg *message.Message) bool {
			br := &pb.BlockRequestPayload{}
			if err := ptypes.UnmarshalAny(msg.Payload, br); err != nil {
				t.Error("Can't unmarshal request payload")
			}
			return br.Height == 4
		})).Return(getMessage(block41.GetMessage(), block42.GetMessage()), nil).Once()

	err := toTest.RequestBlocks(context.Background(), 2, 4, nil)
	assert.Nil(t, err)

	assert.Equal(t, block31, bc.GetBlockByHash(block31.Header().Hash()))
	assert.Equal(t, block32, bc.GetBlockByHash(block32.Header().Hash()))
	assert.Equal(t, block33, bc.GetBlockByHash(block33.Header().Hash()))
	assert.Equal(t, block41, bc.GetBlockByHash(block41.Header().Hash()))
	assert.Equal(t, block42, bc.GetBlockByHash(block42.Header().Hash()))
}

func TestSynchRequestBlocksForHeightRangePartiallyWithTimeout(t *testing.T) {
	srv := &mocks.Service{}
	bc := blockchain.CreateBlockchainFromGenesisBlock(&blockchain.Config{Storage: mockStorage(), Pool: mockPool(), Db: mockDB()})
	bc.GetGenesisBlock().SetQC(blockchain.CreateQuorumCertificate([]byte("valid"), bc.GetGenesisBlock().Header()))

	me := generateIdentity(t, 0)
	bsrv := blockchain.NewBlockService(srv)
	toTest := blockchain.CreateSynchronizer(me, bsrv, bc)

	block11 := bc.NewBlock(bc.GetGenesisBlock(), bc.GetGenesisCert(), []byte("newBlock11"))
	_ = bc.AddBlock(block11)
	block21 := bc.NewBlock(block11, bc.GetGenesisCert(), []byte("newBlock21"))
	_ = bc.AddBlock(block21)

	block31 := bc.NewBlock(block21, bc.GetGenesisCert(), []byte("newBlock31"))
	block32 := bc.NewBlock(block21, bc.GetGenesisCert(), []byte("newBlock32"))
	block33 := bc.NewBlock(block21, bc.GetGenesisCert(), []byte("newBlock33"))

	block41 := bc.NewBlock(block31, bc.GetGenesisCert(), []byte("newBlock41"))
	block42 := bc.NewBlock(block32, bc.GetGenesisCert(), []byte("newBlock42"))

	errorChan := make(chan error)
	srv.On("SendRequestToRandomPeer", mock.MatchedBy(func(ctx context.Context) bool { return true }),
		mock.MatchedBy(func(msg *message.Message) bool {
			br := &pb.BlockRequestPayload{}
			if err := ptypes.UnmarshalAny(msg.Payload, br); err != nil {
				t.Error("Can't unmarshal request payload")
			}
			return br.Height == 3
		})).Return(getMessage(block31.GetMessage(), block32.GetMessage(), block33.GetMessage()), nil).Once()
	srv.On("SendRequestToRandomPeer", mock.MatchedBy(func(ctx context.Context) bool { return true }),
		mock.MatchedBy(func(msg *message.Message) bool {
			br := &pb.BlockRequestPayload{}
			if err := ptypes.UnmarshalAny(msg.Payload, br); err != nil {
				t.Error("Can't unmarshal request payload")
			}
			return br.Height == 4
		})).Run(func(args mock.Arguments) {
		ctx := (args[0]).(context.Context)
		<-ctx.Done()
		go func() {
			errorChan <- ctx.Err()
		}()
	}).Return(nil, errorChan).Once()

	timeout, _ := context.WithTimeout(context.Background(), 2*time.Second)
	err := toTest.RequestBlocks(timeout, 2, 4, nil)
	assert.Error(t, err)

	assert.False(t, bc.Contains(block31.Header().Hash()))
	assert.False(t, bc.Contains(block32.Header().Hash()))
	assert.False(t, bc.Contains(block33.Header().Hash()))
	assert.False(t, bc.Contains(block41.Header().Hash()))
	assert.False(t, bc.Contains(block42.Header().Hash()))
}

func TestSynchRequestBlocksForHeightRangeBreakingBlockchain(t *testing.T) {
	srv := &mocks.Service{}
	bc := blockchain.CreateBlockchainFromGenesisBlock(&blockchain.Config{Storage: mockStorage(), Pool: mockPool(), Db: mockDB()})
	bc.GetGenesisBlock().SetQC(blockchain.CreateQuorumCertificate([]byte("valid"), bc.GetGenesisBlock().Header()))

	me := generateIdentity(t, 0)
	bsrv := blockchain.NewBlockService(srv)
	toTest := blockchain.CreateSynchronizer(me, bsrv, bc)

	block11 := bc.NewBlock(bc.GetGenesisBlock(), bc.GetGenesisCert(), []byte("newBlock11"))
	block21 := bc.NewBlock(block11, bc.GetGenesisCert(), []byte("newBlock21"))

	block31 := bc.NewBlock(block21, bc.GetGenesisCert(), []byte("newBlock31"))
	block32 := bc.NewBlock(block21, bc.GetGenesisCert(), []byte("newBlock32"))
	block33 := bc.NewBlock(block21, bc.GetGenesisCert(), []byte("newBlock33"))

	block41 := bc.NewBlock(block31, bc.GetGenesisCert(), []byte("newBlock41"))
	block42 := bc.NewBlock(block32, bc.GetGenesisCert(), []byte("newBlock42"))

	srv.On("SendRequestToRandomPeer", mock.MatchedBy(func(ctx context.Context) bool { return true }),
		mock.MatchedBy(func(msg *message.Message) bool {
			br := &pb.BlockRequestPayload{}
			if err := ptypes.UnmarshalAny(msg.Payload, br); err != nil {
				t.Error("Can't unmarshal request payload")
			}
			return br.Height == 3
		})).Return(getMessage(block31.GetMessage(), block32.GetMessage(), block33.GetMessage()), nil).Once()
	srv.On("SendRequestToRandomPeer", mock.MatchedBy(func(ctx context.Context) bool { return true }),
		mock.MatchedBy(func(msg *message.Message) bool {
			br := &pb.BlockRequestPayload{}
			if err := ptypes.UnmarshalAny(msg.Payload, br); err != nil {
				t.Error("Can't unmarshal request payload")
			}
			return br.Height == 4
		})).Return(getMessage(block41.GetMessage(), block42.GetMessage()), nil).Once()

	err := toTest.RequestBlocks(context.Background(), 2, 4, nil)
	assert.NotNil(t, err)
}

func TestSyncFork(t *testing.T) {
	srv := &mocks.Service{}
	bc := blockchain.CreateBlockchainFromGenesisBlock(&blockchain.Config{Storage: mockStorage(), Pool: mockPool(), Db: mockDB()})
	bc.GetGenesisBlock().SetQC(blockchain.CreateQuorumCertificate([]byte("valid"), bc.GetGenesisBlock().Header()))

	me := generateIdentity(t, 0)
	bsrv := blockchain.NewBlockService(srv)
	toTest := blockchain.CreateSynchronizer(me, bsrv, bc)

	block11 := bc.NewBlock(bc.GetGenesisBlock(), bc.GetGenesisCert(), []byte("newBlock11"))
	block21 := bc.NewBlock(block11, bc.GetGenesisCert(), []byte("newBlock21"))

	block31 := bc.NewBlock(block21, bc.GetGenesisCert(), []byte("newBlock31"))

	block41 := bc.NewBlock(block31, bc.GetGenesisCert(), []byte("newBlock41"))

	srv.On("SendRequestToRandomPeer", mock.MatchedBy(func(ctx context.Context) bool { return true }),
		mock.MatchedBy(func(msg *message.Message) bool {
			br := &pb.BlockRequestPayload{}
			if err := ptypes.UnmarshalAny(msg.Payload, br); err != nil {
				t.Error("Can't unmarshal request payload")
			}
			return br.Height == 1
		})).Return(getMessage(block11.GetMessage(), block21.GetMessage(), block31.GetMessage(), block41.GetMessage()), nil).Once()

	err := toTest.RequestFork(context.Background(), block41.Header().Hash(), nil)
	assert.Nil(t, err)

	assert.Equal(t, block11, bc.GetBlockByHash(block11.Header().Hash()))
	assert.Equal(t, block21, bc.GetBlockByHash(block21.Header().Hash()))
	assert.Equal(t, block31, bc.GetBlockByHash(block31.Header().Hash()))
	assert.Equal(t, block41, bc.GetBlockByHash(block41.Header().Hash()))

}

func TestSyncForkIntegrityViolation(t *testing.T) {
	srv := &mocks.Service{}
	bc := blockchain.CreateBlockchainFromGenesisBlock(&blockchain.Config{Storage: mockStorage(), Pool: mockPool(), Db: mockDB()})
	bc.GetGenesisBlock().SetQC(blockchain.CreateQuorumCertificate([]byte("valid"), bc.GetGenesisBlock().Header()))

	me := generateIdentity(t, 0)
	bsrv := blockchain.NewBlockService(srv)
	toTest := blockchain.CreateSynchronizer(me, bsrv, bc)

	block11 := bc.NewBlock(bc.GetGenesisBlock(), bc.GetGenesisCert(), []byte("newBlock11"))
	block21 := bc.NewBlock(block11, bc.GetGenesisCert(), []byte("newBlock21"))

	block31 := bc.NewBlock(block21, bc.GetGenesisCert(), []byte("newBlock31"))
	block32 := bc.NewBlock(block21, bc.GetGenesisCert(), []byte("newBlock32"))
	block33 := bc.NewBlock(block21, bc.GetGenesisCert(), []byte("newBlock33"))

	block41 := bc.NewBlock(block31, bc.GetGenesisCert(), []byte("newBlock41"))

	srv.On("SendRequestToRandomPeer", mock.MatchedBy(func(ctx context.Context) bool { return true }),
		mock.MatchedBy(func(msg *message.Message) bool {
			br := &pb.BlockRequestPayload{}
			if err := ptypes.UnmarshalAny(msg.Payload, br); err != nil {
				t.Error("Can't unmarshal request payload")
			}
			return br.Height == 1
		})).Return(getMessage(block31.GetMessage(), block32.GetMessage(), block33.GetMessage()), nil).Once()

	err := toTest.RequestFork(context.Background(), block41.Header().Hash(), nil)
	assert.NotNil(t, err)
}
func TestSyncForkPartial(t *testing.T) {
	srv := &mocks.Service{}
	bc := blockchain.CreateBlockchainFromGenesisBlock(&blockchain.Config{Storage: mockStorage(), Pool: mockPool(), Db: mockDB()})
	bc.GetGenesisBlock().SetQC(blockchain.CreateQuorumCertificate([]byte("valid"), bc.GetGenesisBlock().Header()))

	me := generateIdentity(t, 0)
	bsrv := blockchain.NewBlockService(srv)
	toTest := blockchain.CreateSynchronizer(me, bsrv, bc)

	block11 := bc.NewBlock(bc.GetGenesisBlock(), bc.GetGenesisCert(), []byte("newBlock11"))
	block21 := bc.NewBlock(block11, bc.GetGenesisCert(), []byte("newBlock21"))

	block31 := bc.NewBlock(block21, bc.GetGenesisCert(), []byte("newBlock31"))

	block41 := bc.NewBlock(block31, bc.GetGenesisCert(), []byte("newBlock41"))

	srv.On("SendRequestToRandomPeer", mock.MatchedBy(func(ctx context.Context) bool { return true }),
		mock.MatchedBy(func(msg *message.Message) bool {
			br := &pb.BlockRequestPayload{}
			if err := ptypes.UnmarshalAny(msg.Payload, br); err != nil {
				t.Error("Can't unmarshal request payload")
			}
			return br.Height == 1
		})).Return(getMessage(block11.GetMessage(), block21.GetMessage(), block31.GetMessage()), nil).Once()

	err := toTest.RequestFork(context.Background(), block41.Header().Hash(), nil)
	assert.NotNil(t, err)
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
