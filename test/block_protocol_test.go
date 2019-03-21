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
	bc := blockchain.CreateBlockchainFromGenesisBlock()
	bc.SetSynchronizer(synchr)
	p := blockchain.CreateBlockProtocol(srv, bc, synchr, 2)

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
