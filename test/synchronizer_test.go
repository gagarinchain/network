package test

import (
	"github.com/golang/protobuf/ptypes"
	"github.com/poslibp2p/blockchain"
	"github.com/poslibp2p/eth/common"
	"github.com/poslibp2p/message"
	"github.com/poslibp2p/message/protobuff"
	"github.com/poslibp2p/mocks"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestSynchronizer(t *testing.T) {
	srv := &mocks.Service{}
	bc := blockchain.CreateBlockchainFromGenesisBlock()
	me := generateIdentity(t)
	toTest := blockchain.CreateSynchronizer(make(chan *blockchain.Block), me, srv, bc)

	head := bc.GetHead()
	newBlock := blockchain.NewBlock(head, bc.GetGenesisCert(), []byte(""))
	//newQC := blockchain.CreateQuorumCertificate([]byte("New QC"), newBlock.Header())
	log.Info("Head ", common.Bytes2Hex(newBlock.Header().Hash().Bytes()))

	pbBlock := newBlock.GetMessage()
	any, _ := ptypes.MarshalAny(pbBlock)
	msg := message.CreateMessage(pb.Message_BLOCK_REQUEST, me.GetPrivateKey(), any)

	srv.On("SendMessageToRandomPeer", mock.AnythingOfType("*message.Message")).Return(msg)

	toTest.RequestBlockWithDeps(newBlock.Header())

}
