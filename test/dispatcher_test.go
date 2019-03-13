package test

import (
	"github.com/poslibp2p/message"
	"github.com/poslibp2p/message/protobuff"
	"github.com/poslibp2p/mocks"
	"github.com/stretchr/testify/mock"
	"sync"
	"testing"
)

func TestDispatch(t *testing.T) {
	ch := make(chan *message.Message, 1)

	mockHandler := &mocks.Handler{}
	handlers := map[pb.Message_MessageType]message.Handler{
		pb.Message_HELLO_REQUEST: mockHandler,
	}

	outM := &message.Message{Message: &pb.Message{Type: pb.Message_HELLO_REQUEST}}

	var wg = sync.WaitGroup{}
	wg.Add(1)
	mockHandler.On("Handle", outM).Run(func(args mock.Arguments) {
		//assert.Equal(t, outM.Hash, args[0].(*message.Message).Hash)
		defer wg.Done()
	}).Once()

	d := message.Dispatcher{Handlers: handlers, MsgChan: ch}

	//outM.Type = pb.Message_HELLO
	d.Dispatch(outM)
	go d.StartUp()

	wg.Wait()

	mockHandler.AssertCalled(t, "Handle", outM)
}

func TestDispatchWithDefaultDispatcher(t *testing.T) {
	ch := make(chan *message.Message, 1)

	mockHandler := &mocks.Handler{}
	handlers := map[pb.Message_MessageType]message.Handler{
		pb.Message_HELLO_REQUEST: mockHandler,
	}

	outM := &message.Message{Message: &pb.Message{Type: -1}}

	message.DEFAULT_HANDLER = mockHandler
	var wg = sync.WaitGroup{}
	wg.Add(1)
	mockHandler.On("Handle", outM).Run(func(args mock.Arguments) {
		//assert.Equal(t, outM.Hash, args[0].(*message.Message).Hash)
		defer wg.Done()
	}).Once()

	d := message.Dispatcher{Handlers: handlers, MsgChan: ch}

	//outM.Type = pb.Message_HELLO
	d.Dispatch(outM)
	go d.StartUp()

	wg.Wait()

	mockHandler.AssertCalled(t, "Handle", outM)
}
