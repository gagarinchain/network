package hotstuff

import (
	"github.com/poslibp2p/eth/common"
	msg "github.com/poslibp2p/message"
	"sync"
	"time"
)

//Static pacer that store validator set in file and round-robin elect proposer each 2 Delta-periods
type StaticPacer struct {
	config              *ProtocolConfig
	currentEpoch        int32
	IsStarting          bool
	epochMessageStorage map[common.Address]int32
	storageGuard        *sync.RWMutex
	committee           []*msg.Peer
	epochStartSubChan   []chan interface{}
	roundEndChan        chan interface{}
	stopChan            chan interface{}
}

func CreatePacer(config *ProtocolConfig) *StaticPacer {
	return &StaticPacer{
		config:              config,
		currentEpoch:        0,
		IsStarting:          false,
		epochMessageStorage: make(map[common.Address]int32),
		storageGuard:        &sync.RWMutex{},
		committee:           config.CommitteeLoader.LoadFromFile(),
		roundEndChan:        config.RoundEndChan,
		stopChan:            make(chan interface{}),
	}
}

func (p *StaticPacer) Bootstrap() {
	p.StartEpoch(1)
}
func (p *StaticPacer) Stop() {
	p.stopChan <- struct{}{}
}

func (p *StaticPacer) Committee() []*msg.Peer {
	return p.committee
}

func (p *StaticPacer) GetCurrentHeight() int32 {
	return p.config.Blockchain.GetHead().Header().Height() + 1
}

func (p *StaticPacer) CurrentProposerOrStartEpoch() (*msg.Peer, bool) {
	i := int(p.GetCurrentHeight()) % len(p.committee)
	return p.committee[i], i == 0
}

func (p *StaticPacer) GetCurrent() *msg.Peer {
	return p.committee[int(p.GetCurrentHeight())%len(p.committee)]
}

func (p *StaticPacer) GetNext() *msg.Peer {
	return p.committee[int(p.GetCurrentHeight()+1)%len(p.committee)]
}

func (p *StaticPacer) StartEpoch(i int32) {
	p.IsStarting = true
	epoch := CreateEpoch(p.config.Me, i)
	m, e := epoch.GetMessage()
	if e != nil {
		log.Error("Can't create Epoch message", e)
	}
	go p.config.Srv.Broadcast(m)
}

func (p *StaticPacer) OnEpochStart(m *msg.Message, s *msg.Peer) {
	p.storageGuard.Lock()
	defer p.storageGuard.Unlock()

	epoch, e := CreateEpochFromMessage(m, s)
	if e != nil {
		log.Error(e)
		return
	}

	if epoch.number < p.currentEpoch+1 {
		log.Warning("received epoch message for previous epoch ", epoch.number)
		return
	}

	p.epochMessageStorage[epoch.sender.GetAddress()] = epoch.number

	stats := make(map[int32]int32)
	for _, v := range p.epochMessageStorage {
		stats[v] += 1
	}
	max := struct {
		n int32
		c int32
	}{0, 0}
	for k, v := range stats {
		if max.c < v {
			max.n = k
			max.c = v
		}
	}

	if max.n <= p.currentEpoch && int(max.c) == p.config.F/3+1 {
		log.Fatal("Somehow we are ahead on epochs than f + 1 peers, it is impossible")
	}

	//We received at least 1 message from fair peer, should resynchronize our epoch
	if int(max.c) == p.config.F/3+1 {
		p.StartEpoch(max.n)
	}

	//We got quorum, lets start new epoch
	if int(max.c) == p.config.F/3*2+1 {
		p.newEpoch(max.n)
	}
}

func (p *StaticPacer) newEpoch(i int32) {
	if i > 1 {
		delta := (i-1)*int32(p.config.F) - p.GetCurrentHeight()
		for k := 0; k < int(delta); k++ {
			p.config.Blockchain.PadEmptyBlock()
		}
	}

	p.currentEpoch = i
	p.epochMessageStorage = make(map[common.Address]int32)
	for _, ch := range p.epochStartSubChan {
		go func(c chan interface{}) {
			c <- struct{}{}
		}(ch)
	}
	p.IsStarting = false
}

func (p *StaticPacer) SubscribeEpochChange(trigger chan interface{}) {
	p.storageGuard.Lock()
	defer p.storageGuard.Unlock()
	p.epochStartSubChan = append(p.epochStartSubChan, trigger)
}

func (p *StaticPacer) Run() {
	ticker := time.NewTicker(2 * p.config.Delta)
	timer := time.NewTimer(p.config.Delta)
	for {
		select {
		case <-timer.C:
			log.Info("Received no votes from peers in delta, proposing with last QC")

			p.config.ControlChan <- Event(PROPOSE)

		case <-ticker.C:
			//TODO ignore when synchronizing
			log.Info("Received no signal from underlying protocol about round ending, force proposer change")
			ticker.Stop()

			i := int(p.GetCurrentHeight()) % len(p.committee)
			if i == 0 {
				p.StartEpoch(p.currentEpoch + 1)
				ticker = time.NewTicker(4 * p.config.Delta)
			}

			p.config.Blockchain.PadEmptyBlock()
			ticker = time.NewTicker(2 * p.config.Delta)
		case <-p.roundEndChan:
			log.Info("Protocol round ended gracefully")
			ticker.Stop()

			i := int(p.GetCurrentHeight()) % len(p.committee)
			if i == 0 {
				p.StartEpoch(p.currentEpoch + 1)
				ticker = time.NewTicker(4 * p.config.Delta)
			}
			ticker.Stop()
			ticker = time.NewTicker(2 * p.config.Delta)
		case <-p.stopChan:
			ticker.Stop()
			return
		}
	}
}
