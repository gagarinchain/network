package hotstuff

import (
	"context"
	"github.com/poslibp2p/common"
)

//Static pacer that store validator set in file and round-robin elect proposer each 2 Delta-periods
type StaticPacer struct {
	config        *ProtocolConfig
	committee     []*common.Peer
	stopChan      chan interface{}
	viewGetter    CurrentViewGetter
	eventNotifier EventNotifier
	eventChan     chan Event
}

func CreatePacer(config *ProtocolConfig) *StaticPacer {
	pacer := &StaticPacer{
		config:    config,
		committee: config.Committee,
		stopChan:  make(chan interface{}),
		eventChan: make(chan Event),
	}
	return pacer
}

func (p *StaticPacer) SetViewGetter(getter CurrentViewGetter) {
	p.viewGetter = getter
}
func (p *StaticPacer) SetEventNotifier(notifier EventNotifier) {
	p.eventNotifier = notifier
}
func (p *StaticPacer) Bootstrap(ctx context.Context) {
	go p.Run(ctx)
}

func (p *StaticPacer) Stop() {
	p.stopChan <- struct{}{}
}

func (p *StaticPacer) Committee() []*common.Peer {
	return p.committee
}

func (p *StaticPacer) WillNextViewForceEpochStart(currentView int32) bool {
	return int(currentView+1)%len(p.committee) == 0
}

func (p *StaticPacer) GetCurrent(currentView int32) *common.Peer {
	return p.committee[int(currentView)%len(p.committee)]
}

func (p *StaticPacer) GetNext(currentView int32) *common.Peer {
	return p.committee[int(currentView+1)%len(p.committee)]
}

func (p *StaticPacer) Run(ctx context.Context) error {
	log.Info("Starting pacer...")
	p.eventNotifier.SubscribeProtocolEvents(p.eventChan)

	timeout, f := context.WithTimeout(ctx, 4*p.config.Delta)
	p.config.ControlChan <- Command{eventType: StartEpoch, ctx: timeout}
	currentState := StartEpoch
	for {
		select {
		case <-timeout.Done(): //case when we timed out
			switch currentState {
			case StartEpoch: //we failed to start epoch, should retry
				log.Info("Can't start epoch in 4*delta, retry...")
				timeout, f = context.WithTimeout(ctx, 4*p.config.Delta)
				p.config.ControlChan <- Command{eventType: StartEpoch, ctx: timeout}
				currentState = StartEpoch
			case SuggestVote: //we timed out interval after new round start, failed to collect qc, should propose without new qc
				log.Info("Received no votes from peers in delta, proposing with last QC")
				timeout, f = context.WithTimeout(ctx, p.config.Delta)
				p.config.ControlChan <- Command{eventType: SuggestPropose, ctx: timeout}
				currentState = SuggestPropose
			case SuggestPropose: //we failed to propose in time, change view and go on progress
				log.Info("Couldn't propose in time, force view change")
				timeout, f = context.WithTimeout(ctx, p.config.Delta)
				p.config.ControlChan <- Command{eventType: NextView, ctx: timeout}
				currentState = NextView
			case NextView:
				panic("Don't know what to do when we timed out view increment")
			}
		case event := <-p.eventChan:
			switch event {
			case StartedEpoch:
				log.Info("Started new epoch, collect votes")
				timeout, f = context.WithTimeout(ctx, p.config.Delta)
				p.config.ControlChan <- Command{eventType: SuggestVote, ctx: timeout}
				currentState = SuggestVote
			case ChangedView:
				log.Infof("Round %v ended", p.viewGetter.GetCurrentView())

				i := int(p.viewGetter.GetCurrentView()) % len(p.committee)
				if i == 0 {
					log.Info("Starting new epoch")
					timeout, f = context.WithTimeout(ctx, 4*p.config.Delta)
					p.config.ControlChan <- Command{eventType: StartEpoch, ctx: timeout}
					currentState = StartEpoch
				} else {
					log.Info("Start new round, collect votes")
					timeout, f = context.WithTimeout(ctx, p.config.Delta)
					p.config.ControlChan <- Command{eventType: SuggestVote, ctx: timeout}
					currentState = SuggestVote
				}
			}
		case <-p.stopChan:
			f()
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}

	}
}
