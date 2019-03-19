package test

import (
	"github.com/poslibp2p/hotstuff"
	msg "github.com/poslibp2p/message"
	"github.com/poslibp2p/mocks"
	"github.com/stretchr/testify/mock"
	"testing"
)

func Test1(t *testing.T) {
	bc, p, cfg := initProtocol(t)

	msgChan := make(chan *msg.Message)
	(cfg.Srv).(*mocks.Service).On("Broadcast", mock.AnythingOfType("*message.Message")).Run(func(args mock.Arguments) {
		msgChan <- (args[0]).(*msg.Message)
	})

	cfg.Pacer.Bootstrap()
	<-msgChan

	go cfg.Pacer.Run()
	proposer := cfg.Pacer.Committee()[4]
	proposal := hotstuff.CreateProposal(bc.NewBlock(bc.GetHead(), bc.GetGenesisCert(), []byte("Hello")), bc.GetGenesisCert(), proposer)
	_ = p.OnReceiveProposal(proposal)

}
