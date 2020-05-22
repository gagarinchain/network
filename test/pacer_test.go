package test

import (
	"context"
	msg "github.com/gagarinchain/common/message"
	"github.com/gagarinchain/network/hotstuff"
	"github.com/gagarinchain/network/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestOnStartEpochWithBrokenSignature(t *testing.T) {
	bc, p, cfg, _ := initProtocol(t, 1)
	pacer := initPacer(t, cfg, p)
	go func() {
		pacer.Run(context.Background(), make(chan *msg.Message), make(chan *msg.Message))
	}()

	msgChan := make(chan *msg.Message)
	(cfg.Srv).(*mocks.Service).On("Broadcast", mock.MatchedBy(func(ctx context.Context) bool { return true }),
		mock.AnythingOfType("*message.Message")).Run(func(args mock.Arguments) {
		msgChan <- (args[1]).(*msg.Message)
	})

	<-msgChan

	for i := 0; i < 2*cfg.F/3; i++ {
		epoch := hotstuff.CreateEpoch(pacer.Committee()[i], 1, nil, bc.GetGenesisBlockSignedHash(pacer.Committee()[i].GetPrivateKey()))
		message, _ := epoch.GetMessage()
		pacer.OnEpochStart(context.Background(), message)
	}

	assert.Equal(t, hotstuff.StartingEpoch, pacer.StateId())

	sign := bc.GetGenesisBlockSignedHash(generateIdentity(t, 3).GetPrivateKey()) // generate random signature
	epoch := hotstuff.CreateEpoch(pacer.Committee()[2*cfg.F/3], 1, nil, sign)
	message, _ := epoch.GetMessage()
	pacer.OnEpochStart(context.Background(), message)

	assert.Equal(t, hotstuff.StartingEpoch, pacer.StateId())

	sign1 := bc.GetGenesisBlockSignedHash(pacer.Committee()[2*cfg.F/3].GetPrivateKey()) // generate random signature
	epoch1 := hotstuff.CreateEpoch(pacer.Committee()[2*cfg.F/3], 1, nil, sign1)
	message1, _ := epoch1.GetMessage()
	pacer.OnEpochStart(context.Background(), message1)

	assert.NotEqual(t, hotstuff.StartingEpoch, pacer.StateId())

}

func initPacer(t *testing.T, cfg *hotstuff.ProtocolConfig, p *hotstuff.Protocol) *hotstuff.StaticPacer {
	pacer := hotstuff.CreatePacer(cfg)
	pacer.Bootstrap(context.Background(), p)
	return pacer
}

//
//func TestStartEpochOnGenesisBlock(t *testing.T) {
//	_, p, cfg := initProtocol(t)
//
//	msgChan := make(chan *msg.Message)
//	(cfg.Srv).(*mocks.Service).On("Broadcast", mock.AnythingOfType("*message.Message")).Run(func(args mock.Arguments) {
//		msgChan <- (args[0]).(*msg.Message)
//	})
//
//	go ctx.pacer.Run()
//	go ctx.protocol.Run(ctx.hotstuffChan)
//	defer ctx.pacer.Stop()
//	defer ctx.protocol.Stop()
//
//	<-msgChan
//
//	for i := 0; i < 2*cfg.F/3; i++ {
//		epoch := hotstuff.CreateEpoch(cfg.Pacer.Committee()[i], 1, p.HQC())
//		message, _ := epoch.GetMessage()
//		p.OnEpochStart(message, cfg.Pacer.Committee()[i])
//	}
//
//	assert.True(t, p.IsStartingEpoch)
//
//	trigger := make(chan interface{})
//	p.SubscribeEpochChange(trigger)
//
//	epoch := hotstuff.CreateEpoch(cfg.Pacer.Committee()[2*cfg.F/3], 1, p.HQC())
//	message, _ := epoch.GetMessage()
//	p.OnEpochStart(message, cfg.Pacer.Committee()[2*cfg.F/3])
//
//	<-trigger
//	assert.False(t, p.IsStartingEpoch)
//	assert.Equal(t, cfg.Pacer.GetCurrent(p.GetCurrentView()), cfg.Pacer.Committee()[3]) //only first epoch starts from fourth peer and shorter than others
//}
//
//func TestPacerStartEpochWhenMissedSome(t *testing.T) {
//	_, p, cfg := initProtocol(t)
//
//	msgChan := make(chan *msg.Message)
//	(cfg.Srv).(*mocks.Service).On("Broadcast", mock.AnythingOfType("*message.Message")).Run(func(args mock.Arguments) {
//		msgChan <- (args[0]).(*msg.Message)
//	})
//
//	go func() {
//		for range cfg.ControlChan {
//
//		}
//	}()
//
//	cfg.Pacer.Bootstrap()
//	<-msgChan
//
//	for i := 0; i < 2*cfg.F/3+1; i++ {
//		epoch := hotstuff.CreateEpoch(cfg.Pacer.Committee()[i], 5, p.HQC())
//		message, _ := epoch.GetMessage()
//		p.OnEpochStart(message, cfg.Pacer.Committee()[i])
//	}
//
//	assert.False(t, p.IsStartingEpoch)
//
//	assert.Equal(t, int32(4*cfg.F), cfg.Blockchain.GetHead().Header().Height() + 1)
//}
//
//func TestStartEpochOnFPlusOneMessage(t *testing.T) {
//	_, p, cfg := initProtocol(t)
//
//	msgChan := make(chan *msg.Message)
//	(cfg.Srv).(*mocks.Service).On("Broadcast", mock.AnythingOfType("*message.Message")).Run(func(args mock.Arguments) {
//		msgChan <- (args[0]).(*msg.Message)
//	})
//
//	for i := 0; i < cfg.F/3+1; i++ {
//		epoch := hotstuff.CreateEpoch(cfg.Pacer.Committee()[i], 5, p.HQC())
//		message, _ := epoch.GetMessage()
//		p.OnEpochStart(message, cfg.Pacer.Committee()[i])
//	}
//
//	assert.True(t, p.IsStartingEpoch)
//
//	m := <-msgChan
//
//	payload := &pb.EpochStartPayload{}
//	_ = ptypes.UnmarshalAny(m.Payload, payload)
//
//	assert.Equal(t, payload.EpochNumber, int32(5))
//}
//
//func TestRoundChange(t *testing.T) {
//
//	_, _, cfg := initProtocol(t)
//
//	msgChan := make(chan *msg.Message)
//	(cfg.Srv).(*mocks.Service).On("Broadcast", mock.AnythingOfType("*message.Message")).Run(func(args mock.Arguments) {
//		msgChan <- (args[0]).(*msg.Message)
//	})
//
//	cfg.Pacer.Bootstrap()
//	<-msgChan
//
//	go cfg.Pacer.Run()
//	cfg.RoundEndChan <- 3
//
//}
