package hotstuff

import (
	"context"
	"errors"
	"fmt"
	cmn "github.com/gagarinchain/common"
	"github.com/gagarinchain/common/api"
	"github.com/gagarinchain/common/eth/common"
	"github.com/gagarinchain/common/eth/crypto"
	msg "github.com/gagarinchain/common/message"
	"github.com/gagarinchain/common/protobuff"
	"github.com/gagarinchain/network/storage"
	"math/big"
	"sync"
	"time"
)

type StateId int

const (
	Bootstrapped  StateId = iota
	StartingEpoch StateId = iota
	Voting        StateId = iota
	Proposing     StateId = iota
)

type PacerPersister struct {
	Storage storage.Storage
}

func (pp *PacerPersister) PutCurrentEpoch(currentEpoch int32) error {
	epoch := storage.Int32ToByte(currentEpoch)
	return pp.Storage.Put(storage.CurrentEpoch, nil, epoch)
}

func (pp *PacerPersister) GetCurrentEpoch() (int32, error) {
	value, err := pp.Storage.Get(storage.CurrentEpoch, nil)
	if err != nil {
		return storage.DefaultIntValue, err
	}
	return storage.ByteToInt32(value)
}
func (pp *PacerPersister) PutCurrentView(currentView int32) error {
	epoch := storage.Int32ToByte(currentView)
	return pp.Storage.Put(storage.CurrentView, nil, epoch)
}

func (pp *PacerPersister) GetCurrentView() (int32, error) {
	value, err := pp.Storage.Get(storage.CurrentView, nil)
	if err != nil {
		return storage.DefaultIntValue, err
	}
	return storage.ByteToInt32(value)
}

//Static pacer that store validator set in file and round-robin elect proposer each 2 Delta-periods
type StaticPacer struct {
	f                     int
	delta                 time.Duration
	me                    *cmn.Peer
	committee             []*cmn.Peer
	protocol              *Protocol
	protocolEventSubChans []chan api.Event
	persister             *PacerPersister
	view                  struct {
		current int32
		guard   *sync.RWMutex
	}
	//todo protect epoch with guard
	epoch struct {
		current        int32
		toStart        int32
		messageStorage map[common.Address]int32
		voteStorage    map[common.Address]*crypto.Signature
	}

	execution struct {
		parent context.Context
		ctx    context.Context
		f      context.CancelFunc
	}
	stateId StateId
}

func (p *StaticPacer) StateId() StateId {
	return p.stateId
}

func CreatePacer(cfg *ProtocolConfig) *StaticPacer {
	for i, c := range cfg.Committee {
		if c == cfg.Me {
			log.Infof("I am %dth %v proposer", i, cfg.Me.GetAddress().Hex())
		}
	}

	pacer := &StaticPacer{
		f:         cfg.F,
		delta:     cfg.Delta,
		me:        cfg.Me,
		committee: cfg.Committee,
		persister: &PacerPersister{Storage: cfg.Storage},
		view: struct {
			current int32
			guard   *sync.RWMutex
		}{
			current: cfg.InitialState.View,
			guard:   &sync.RWMutex{},
		},
		epoch: struct {
			current        int32
			toStart        int32
			messageStorage map[common.Address]int32
			voteStorage    map[common.Address]*crypto.Signature
		}{
			current:        cfg.InitialState.Epoch,
			toStart:        cfg.InitialState.Epoch + 1,
			messageStorage: make(map[common.Address]int32),
			voteStorage:    make(map[common.Address]*crypto.Signature),
		},
		stateId: Bootstrapped,
	}
	return pacer
}

func (p *StaticPacer) Bootstrap(ctx context.Context, protocol *Protocol) {
	log.Info("Bootstrapping pacer")
	p.protocol = protocol
	p.stateId = Bootstrapped
	p.execution.parent = ctx

	p.SubscribeEpochChange(ctx, func(event api.Event) {
		epoch := event.Payload.(int32)
		e := p.persister.PutCurrentEpoch(epoch)
		if e != nil {
			log.Error("Can'T persist new epoch")
		}
	})
	p.SubscribeViewChange(ctx, func(event api.Event) {
		view := event.Payload.(int32)
		e := p.persister.PutCurrentView(view)
		if e != nil {
			log.Error("Can'T persist new view")
		}
	})

	log.Info("Pacer bootstrapped successfully")
}

func (p *StaticPacer) Committee() []*cmn.Peer {
	return p.committee
}

func (p *StaticPacer) ProposerForHeight(blockHeight int32) *cmn.Peer {
	return p.committee[blockHeight%int32(len(p.committee))]
}

func (p *StaticPacer) GetCurrent() *cmn.Peer {
	return p.committee[int(p.GetCurrentView())%len(p.committee)]
}

func (p *StaticPacer) GetNext() *cmn.Peer {
	return p.committee[int(p.GetCurrentView()+1)%len(p.committee)]
}

func (p *StaticPacer) Run(ctx context.Context, hotstuffChan chan *msg.Message, epochChan chan *msg.Message) {
	log.Info("Starting pacer...")

	if p.stateId != Bootstrapped {
		log.Errorf("Pacer is not bootstrapped")
		return
	}
	p.execution.ctx, p.execution.f = context.WithTimeout(ctx, 4*p.delta)
	p.stateId = StartingEpoch
	p.StartEpoch(p.execution.ctx)

	msgChan := hotstuffChan
	for {
		//We null hotstuffChan to prevent protocol from handling messages during epoch start phase.
		if p.stateId == StartingEpoch {
			msgChan = nil
		} else {
			msgChan = hotstuffChan
		}

		select {
		case m := <-msgChan:
			log.Debugf("Received %v message", m.Type.String())
			if m.Type == pb.Message_EPOCH_START {
				log.Error("Epoch start message is not expected on hotstuff channel")
			} else if err := p.protocol.handleMessage(ctx, m); err != nil {
				log.Error(err)
				break
			}
		case m := <-epochChan:
			log.Debugf("Received %v message", m.Type.String())
			if m.Type == pb.Message_EPOCH_START {
				if e := p.OnEpochStart(ctx, m); e != nil {
					log.Error(e)
				}
			} else {
				log.Error("Wrong message type sent to epoch start chan")
			}
		case <-p.execution.ctx.Done(): //case when we timed out
			if p.execution.ctx.Err() == context.DeadlineExceeded {
				p.FireEvent(api.Event{
					T: api.TimedOut,
				})
			}
		case <-ctx.Done():
			log.Info("Root context is cancelled, shutting down pacer")
			return
		}
	}
}

func (p *StaticPacer) FireEvent(event api.Event) {
	p.notifyProtocolEvent(event)

	switch event.T {
	case api.TimedOut:
		switch p.stateId {
		case StartingEpoch: //we are timed out during epoch starting, should retry
			log.Info("Can'T start epoch in 4*delta, retry...")
			p.execution.ctx, p.execution.f = context.WithTimeout(p.execution.parent, 4*p.delta)
			p.stateId = StartingEpoch
			p.StartEpoch(p.execution.ctx)
		case Proposing: //we are timed out during proposing, let's start to vote then
			log.Info("Timed out during proposing phase, possibly no votes received for QC, propose with hqc and go to voting")
			p.execution.ctx, p.execution.f = context.WithTimeout(p.execution.parent, p.delta)
			p.stateId = Voting //if we timeout during proposing probably during next call OnPropose we move to next phase
			p.protocol.OnPropose(p.execution.parent)
		case Voting: //we received no valid proposal during delta, current proposer equivocated, let's resend last successful vote to next proposer
			log.Info("Timed out during voting phase, resending last successful vote")
			//vote with default timeout, we won't cancel it anyway since it is only message send
			//we have choice to make this call async with timeout or sync with managing timeouts in pacer
			//actually failed message sending and it's duration doesn't make sense for period duration
			p.protocol.Vote(p.execution.parent)
		default:
			log.Errorf("Unknown transition %v %v", event, p.stateId)
		}
	case api.EpochStartTriggered:
		log.Info("Force starting new epoch")
		p.execution.f() //cancelling previous context
		p.execution.ctx, p.execution.f = context.WithTimeout(p.execution.parent, 4*p.delta)
		p.stateId = StartingEpoch
		p.StartEpoch(p.execution.ctx)
	case api.EpochStarted:
		log.Info("Started new epoch, propose")
		p.execution.f() //cancelling previous context
		p.execution.ctx, p.execution.f = context.WithTimeout(p.execution.parent, p.delta)
		p.stateId = Proposing
		p.OnNextView()
	case api.Voted:
		log.Info("Voted for block, start new round")
		p.execution.f() //cancelling previous context
		i := int(p.GetCurrentView()) % len(p.committee)
		if i == 0 {
			log.Info("Starting new epoch")
			p.execution.ctx, p.execution.f = context.WithTimeout(p.execution.parent, 4*p.delta)
			p.stateId = StartingEpoch
			p.StartEpoch(p.execution.ctx)
		} else {
			log.Info("Start new round, collect votes")
			p.OnNextView()
			p.execution.ctx, p.execution.f = context.WithTimeout(p.execution.parent, p.delta)
			p.stateId = Proposing
		}
	case api.VotesCollected:
		log.Info("Collected all votes for new QC, proposing")
		p.stateId = Proposing
		p.protocol.OnPropose(p.execution.parent)
	case api.Proposed:
		log.Info("Proposed")
		p.stateId = Voting
		p.execution.f() //cancelling previous context
		p.execution.ctx, p.execution.f = context.WithTimeout(p.execution.parent, p.delta)
	case api.ChangedView:
		log.Infof("New view %d is started", p.view.current)
		p.stateId = Proposing
		p.execution.f() //cancelling previous context
		p.execution.ctx, p.execution.f = context.WithTimeout(p.execution.parent, p.delta)
	}

}

//if we need unsubscribe we will refactor list to map and identify subscribers
func (p *StaticPacer) SubscribeProtocolEvents(sub chan api.Event) {
	p.protocolEventSubChans = append(p.protocolEventSubChans, sub)
}

//warning: we generate a lot of goroutines here, that can block forever
func (p *StaticPacer) notifyProtocolEvent(event api.Event) {
	for _, ch := range p.protocolEventSubChans {
		go func(c chan api.Event) {
			c <- event
		}(ch)
	}
}

func (p *StaticPacer) subscribeEvent(ctx context.Context, types map[api.EventType]interface{}, handler api.EventHandler) {
	events := make(chan api.Event)
	p.SubscribeProtocolEvents(events)
	go func() {
		for {
			select {
			case e, ok := <-events:
				if ok {
					_, contains := types[e.T]
					if contains {
						handler(e)
					}
				}
			case <-ctx.Done():
				log.Debug("SubscribeEvent is done", ctx.Err())
				return
			}
		}
	}()
}

func (p *StaticPacer) SubscribeEvents(ctx context.Context, handler api.EventHandler, types map[api.EventType]interface{}) {
	p.subscribeEvent(ctx, types, handler)
}

func (p *StaticPacer) SubscribeEpochChange(ctx context.Context, handler api.EventHandler) {
	p.subscribeEvent(ctx, map[api.EventType]interface{}{api.EpochStarted: struct{}{}}, handler)
}

func (p *StaticPacer) SubscribeViewChange(ctx context.Context, handler api.EventHandler) {
	p.subscribeEvent(ctx, map[api.EventType]interface{}{api.ChangedView: struct{}{}}, handler)
}

func (p *StaticPacer) GetCurrentView() int32 {
	p.view.guard.RLock()
	defer p.view.guard.RUnlock()
	return p.view.current
}
func (p *StaticPacer) GetCurrentEpoch() int32 {
	return p.epoch.current
}

func (p *StaticPacer) OnNextView() {
	nextView := p.GetCurrentView() + 1
	p.changeView(nextView)
	p.FireEvent(api.Event{
		T:       api.ChangedView,
		Payload: nextView,
	})
}

func (p *StaticPacer) changeView(view int32) {
	p.view.guard.Lock()
	defer p.view.guard.Unlock()

	p.view.current = view
}
func (p *StaticPacer) StartEpoch(ctx context.Context) {
	var epoch *Epoch
	//todo think about moving it to epoch
	log.Debugf("current epoch %v", p.epoch.current)
	if p.epoch.current == -1 { //not yet started
		signedHash := p.protocol.blockchain.GetGenesisBlockSignedHash(p.me.GetPrivateKey())
		log.Debugf("current epoch is genesis, got signature %v", signedHash)
		epoch = CreateEpoch(p.me, p.epoch.toStart, nil, signedHash)
	} else {
		epoch = CreateEpoch(p.me, p.epoch.toStart, p.protocol.HQC(), nil)
	}

	m, e := epoch.GetMessage()
	if e != nil {
		log.Error("Can'T create Epoch message", e)
		return
	}
	go p.protocol.srv.Broadcast(ctx, m)
}

func (p *StaticPacer) OnEpochStart(ctx context.Context, m *msg.Message) error {
	epoch, e := CreateEpochFromMessage(m)
	if e != nil {
		return e
	}

	if e := p.protocol.validateMessage(epoch, pb.Message_EPOCH_START); e != nil {
		return e
	}

	if epoch.number < p.epoch.toStart {
		log.Warning("received epoch message for previous epoch ", epoch.number)
		return nil
	}

	if epoch.genesisSignature != nil {
		res := p.protocol.blockchain.ValidateGenesisBlockSignature(epoch.genesisSignature, epoch.Sender().GetAddress())
		if !res {
			p.protocol.equivocate(epoch.sender)
			return fmt.Errorf("peer %v sent wrong genesis block signature", epoch.sender.GetAddress().Hex())
		}
		p.epoch.voteStorage[epoch.sender.GetAddress()] = epoch.genesisSignature
	}
	p.epoch.messageStorage[epoch.sender.GetAddress()] = epoch.number

	stats := make(map[int32]int32)
	for _, v := range p.epoch.messageStorage {
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

	if max.n <= p.epoch.current && int(max.c) == p.f/3+1 {
		//really impossible, because one peer is fair, and is not synchronized
		return errors.New("somehow we are ahead on epochs than f + 1 peers, it is impossible")
	}

	//We received at least 1 message from fair peer, should resynchronize our epoch
	if int(max.c) == p.f/3+1 {
		if p.stateId == StartingEpoch && p.epoch.current < max.n-1 || p.stateId != StartingEpoch {
			log.Debugf("Received F/3 + 1 start epoch messages, force starting new epoch")
			p.epoch.toStart = max.n
			p.FireEvent(api.Event{
				T: api.EpochStartTriggered,
			})
		}
	}

	//We got quorum, lets start new epoch
	if int(max.c) == (p.f/3)*2+1 {
		log.Debugf("Received 2 * F/3 + 1 start epoch messages, starting new epoch")
		if max.n == 0 {
			var signs []*crypto.Signature
			bitmap := p.GetBitmap(p.epoch.voteStorage)

			for _, v := range p.epoch.voteStorage {
				signs = append(signs, v)
			}
			aggregate := crypto.AggregateSignatures(bitmap, signs)
			p.protocol.FinishGenesisQC(aggregate)
		}
		p.newEpoch(max.n)
	}

	return nil
}

func (p *StaticPacer) GetBitmap(src map[common.Address]*crypto.Signature) *big.Int {
	var committee []common.Address
	for _, peer := range p.GetPeers() {
		committee = append(committee, peer.GetAddress())
	}

	return cmn.GetBitmap(src, committee)
}

func (p *StaticPacer) newEpoch(i int32) {
	if i > 0 {
		p.changeView((i) * int32(p.f))
	}

	p.epoch.current = i
	p.epoch.toStart = i + 1
	p.epoch.messageStorage = make(map[common.Address]int32)
	log.Infof("Started new epoch %v", i)
	log.Infof("Current view number %v, proposer %v", p.view.current, p.GetCurrent().GetAddress().Hex())
	p.FireEvent(api.Event{
		T:       api.EpochStarted,
		Payload: p.epoch.current,
	})
}

func (p *StaticPacer) GetPeers() []*cmn.Peer {
	cpy := make([]*cmn.Peer, len(p.committee))
	copy(cpy, p.committee)
	return cpy
}
