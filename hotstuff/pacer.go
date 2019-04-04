package hotstuff

import (
	msg "github.com/poslibp2p/message"
	"time"
)

//Static pacer that store validator set in file and round-robin elect proposer each 2 Delta-periods
type StaticPacer struct {
	config       *ProtocolConfig
	committee    []*msg.Peer
	roundEndChan chan Event
	stopChan     chan interface{}
	viewGetter   CurrentViewGetter
}

func CreatePacer(config *ProtocolConfig) *StaticPacer {
	return &StaticPacer{
		config:       config,
		committee:    config.Committee,
		roundEndChan: config.RoundEndChan,
		stopChan:     make(chan interface{}),
	}
}

func (p *StaticPacer) SetViewGetter(getter CurrentViewGetter) {
	p.viewGetter = getter
}
func (p *StaticPacer) Bootstrap() {
	go p.Run()
}

func (p *StaticPacer) Stop() {
	p.stopChan <- struct{}{}
}

func (p *StaticPacer) Committee() []*msg.Peer {
	return p.committee
}

func (p *StaticPacer) WillNextViewForceEpochStart(currentView int32) bool {
	return int(currentView+1)%len(p.committee) == 0
}

func (p *StaticPacer) GetCurrent(currentView int32) *msg.Peer {
	return p.committee[int(currentView)%len(p.committee)]
}

func (p *StaticPacer) GetNext(currentView int32) *msg.Peer {
	return p.committee[int(currentView+1)%len(p.committee)]
}

func (p *StaticPacer) Run() {
	log.Info("Starting pacer...")
	roundTimer := time.NewTimer(2 * p.config.Delta)
	proposeTimer := time.NewTimer(p.config.Delta)
	epochTimer := time.NewTimer(4 * p.config.Delta)

	p.config.ControlChan <- Event(START_EPOCH)

	for {
		select {
		case <-proposeTimer.C:
			log.Info("Received no votes from peers in delta, proposing with last QC")
			p.config.ControlChan <- Event(SUGGEST_PROPOSE)
		case <-roundTimer.C:
			log.Info("Received no signal from underlying protocol about round ending, force proposer change")
			p.config.ControlChan <- Event(NEXT_VIEW)
		case event := <-p.roundEndChan:
			switch event {
			case STARTED_EPOCH:
				epochTimer.Stop()
				roundTimer = time.NewTimer(2 * p.config.Delta)
				proposeTimer = time.NewTimer(p.config.Delta)
			case CHANGED_VIEW:
				log.Infof("Round %v ended", p.viewGetter.GetCurrentView())

				proposeTimer.Stop()
				roundTimer.Stop()

				i := int(p.viewGetter.GetCurrentView()) % len(p.committee)
				if i == 0 {
					p.config.ControlChan <- Event(START_EPOCH)

					epochTimer = time.NewTimer(4 * p.config.Delta)
				} else {
					roundTimer = time.NewTimer(2 * p.config.Delta)
					proposeTimer = time.NewTimer(p.config.Delta)
				}
			}

		case <-epochTimer.C:
			log.Info("Can't start epoch in 4*delta, retry...")
			p.config.ControlChan <- Event(START_EPOCH)
			epochTimer = time.NewTimer(4 * p.config.Delta)
		case <-p.stopChan:
			proposeTimer.Stop()
			roundTimer.Stop()
			return
		}
	}
}
