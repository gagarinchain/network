package test

import (
	"github.com/golang/protobuf/ptypes"
	"github.com/poslibp2p/hotstuff"
	msg "github.com/poslibp2p/message"
	"github.com/poslibp2p/message/protobuff"
	"github.com/poslibp2p/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestPacerBootstrap(t *testing.T) {
	_, _, cfg := initProtocol(t)

	msgChan := make(chan *msg.Message)
	(cfg.Srv).(*mocks.Service).On("Broadcast", mock.AnythingOfType("*message.Message")).Run(func(args mock.Arguments) {
		msgChan <- (args[0]).(*msg.Message)
	})

	cfg.Pacer.Bootstrap()
	m := <-msgChan
	assert.True(t, m.Type == pb.Message_EPOCH_START)
	assert.True(t, cfg.Pacer.IsStarting)
	payload := &pb.EpochStartPayload{}
	_ = ptypes.UnmarshalAny(m.Payload, payload)
	assert.True(t, payload.EpochNumber == 1)
}

func TestPacerStartEpochOnGenesisBlock(t *testing.T) {
	_, _, cfg := initProtocol(t)

	msgChan := make(chan *msg.Message)
	(cfg.Srv).(*mocks.Service).On("Broadcast", mock.AnythingOfType("*message.Message")).Run(func(args mock.Arguments) {
		msgChan <- (args[0]).(*msg.Message)
	})

	cfg.Pacer.Bootstrap()
	<-msgChan

	for i := 0; i < 2*cfg.F/3; i++ {
		epoch := hotstuff.CreateEpoch(cfg.Pacer.Committee()[i], 1)
		message, _ := epoch.GetMessage()
		cfg.Pacer.OnEpochStart(message, cfg.Pacer.Committee()[i])
	}

	assert.True(t, cfg.Pacer.IsStarting)

	trigger := make(chan interface{})
	cfg.Pacer.SubscribeEpochChange(trigger)

	epoch := hotstuff.CreateEpoch(cfg.Pacer.Committee()[2*cfg.F/3], 1)
	message, _ := epoch.GetMessage()
	cfg.Pacer.OnEpochStart(message, cfg.Pacer.Committee()[2*cfg.F/3])

	<-trigger
	assert.False(t, cfg.Pacer.IsStarting)
	assert.Equal(t, cfg.Pacer.GetCurrent(), cfg.Pacer.Committee()[3]) //only first epoch starts from fourth peer and shorter than others
}

func TestPacerStartEpochWhenMissedSome(t *testing.T) {
	_, _, cfg := initProtocol(t)

	msgChan := make(chan *msg.Message)
	(cfg.Srv).(*mocks.Service).On("Broadcast", mock.AnythingOfType("*message.Message")).Run(func(args mock.Arguments) {
		msgChan <- (args[0]).(*msg.Message)
	})

	cfg.Pacer.Bootstrap()
	<-msgChan

	for i := 0; i < 2*cfg.F/3+1; i++ {
		epoch := hotstuff.CreateEpoch(cfg.Pacer.Committee()[i], 5)
		message, _ := epoch.GetMessage()
		cfg.Pacer.OnEpochStart(message, cfg.Pacer.Committee()[i])
	}

	assert.False(t, cfg.Pacer.IsStarting)

	epoch := hotstuff.CreateEpoch(cfg.Pacer.Committee()[2*cfg.F/3], 1)
	message, _ := epoch.GetMessage()
	cfg.Pacer.OnEpochStart(message, cfg.Pacer.Committee()[2*cfg.F/3])

	assert.Equal(t, cfg.Blockchain.GetHead().Header().Height()+1, int32(4*cfg.F))
}

func TestStartEpochOnFPlusOneMessage(t *testing.T) {
	_, _, cfg := initProtocol(t)

	msgChan := make(chan *msg.Message)
	(cfg.Srv).(*mocks.Service).On("Broadcast", mock.AnythingOfType("*message.Message")).Run(func(args mock.Arguments) {
		msgChan <- (args[0]).(*msg.Message)
	})

	for i := 0; i < cfg.F/3+1; i++ {
		epoch := hotstuff.CreateEpoch(cfg.Pacer.Committee()[i], 5)
		message, _ := epoch.GetMessage()
		cfg.Pacer.OnEpochStart(message, cfg.Pacer.Committee()[i])
	}

	assert.True(t, cfg.Pacer.IsStarting)

	m := <-msgChan

	payload := &pb.EpochStartPayload{}
	_ = ptypes.UnmarshalAny(m.Payload, payload)

	assert.Equal(t, payload.EpochNumber, int32(5))
}

func TestRoundChange(t *testing.T) {

	_, _, cfg := initProtocol(t)

	msgChan := make(chan *msg.Message)
	(cfg.Srv).(*mocks.Service).On("Broadcast", mock.AnythingOfType("*message.Message")).Run(func(args mock.Arguments) {
		msgChan <- (args[0]).(*msg.Message)
	})

	cfg.Pacer.Bootstrap()
	<-msgChan

	go cfg.Pacer.Run()
	cfg.RoundEndChan <- struct{}{}

}
