package hotstuff

import (
	"context"
	cmn "github.com/gagarinchain/common"
	"github.com/gagarinchain/common/api"
	"github.com/gagarinchain/common/eth/common"
	"github.com/gagarinchain/common/eth/crypto"
	msg "github.com/gagarinchain/common/message"
	"github.com/gagarinchain/common/protobuff"
	"github.com/gagarinchain/network/blockchain"
	"github.com/gagarinchain/network/storage"
	"github.com/golang/protobuf/ptypes"
	"github.com/status-im/keycard-go/hexutils"
	"math/big"
	"sync"
	"time"
)

type StateId int

const (
	Running       StateId = iota
	Synchronizing StateId = iota
	Bootstrapped  StateId = iota
)

type PacerPersister struct {
	Storage storage.Storage
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
	lastSynchronize       api.Sync
	view                  struct {
		current int32
		guard   *sync.RWMutex
	}
	sync struct {
		messageStorage       map[common.Address]api.Sync
		votingMessageStorage map[common.Address]api.Sync
		guard                *sync.RWMutex
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
		sync: struct {
			messageStorage       map[common.Address]api.Sync
			votingMessageStorage map[common.Address]api.Sync
			guard                *sync.RWMutex
		}{
			messageStorage:       make(map[common.Address]api.Sync),
			votingMessageStorage: make(map[common.Address]api.Sync),
			guard:                &sync.RWMutex{},
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

func (p *StaticPacer) Run(ctx context.Context, hotstuffChan chan *msg.Message) {
	log.Info("Starting pacer...")

	if p.stateId != Bootstrapped {
		log.Errorf("Pacer is not bootstrapped")
		return
	}

	p.FireEvent(api.Event{
		T: api.TimedOut,
	})

	msgChan := hotstuffChan
	for {
		select {
		case m := <-msgChan:
			log.Debugf("Received %v message", m.Type.String())
			if m.Type == pb.Message_SYNCHRONIZE {
				if err := p.OnSynchronize(ctx, m); err != nil {
					log.Error(err)
					break
				}
			} else if err := p.protocol.handleMessage(ctx, m); err != nil {
				log.Error(err)
				break
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

func (p *StaticPacer) NotifyEvent(event api.Event) {
	p.notifyProtocolEvent(p.execution.parent, event)
}

func (p *StaticPacer) FireEvent(event api.Event) {
	p.notifyProtocolEvent(p.execution.parent, event)

	switch event.T {
	case api.TimedOut:
		log.Info("Timed out during voting phase, synchronizing ")
		p.execution.ctx, p.execution.f = context.WithTimeout(p.execution.parent, 4*p.delta)
		p.Synchronize(p.execution.ctx)
	case api.HCUpdated:
		p.changeView(event.Payload.(int32) + 1)
		log.Infof("New view %d is started", p.view.current)
		p.execution.f() //cancelling previous context
		p.execution.ctx, p.execution.f = context.WithTimeout(p.execution.parent, 2*p.delta)
		if p.Committee()[int(p.GetCurrentView())%len(p.committee)].Equals(p.me) { //we are leaders
			log.Infof("We are proposer, proposing at height [%v]", p.view)
			p.protocol.OnPropose(p.execution.ctx)
		}
	default:
		log.Warningf("Unknown event %v", event)
	}

}

//if we need unsubscribe we will refactor list to map and identify subscribers
func (p *StaticPacer) SubscribeProtocolEvents(sub chan api.Event) {
	p.protocolEventSubChans = append(p.protocolEventSubChans, sub)
}

//warning: we generate a lot of goroutines here, that can block forever
func (p *StaticPacer) notifyProtocolEvent(ctx context.Context, event api.Event) {
	for _, ch := range p.protocolEventSubChans {
		c, _ := context.WithTimeout(ctx, time.Second)
		go func(ctx context.Context, c chan api.Event) {
			select {
			case c <- event:
			case <-ctx.Done():
			}
		}(c, ch)
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

func (p *StaticPacer) SubscribeViewChange(ctx context.Context, handler api.EventHandler) {
	p.subscribeEvent(ctx, map[api.EventType]interface{}{api.ChangedView: struct{}{}}, handler)
}

func (p *StaticPacer) GetCurrentView() int32 {
	p.view.guard.RLock()
	defer p.view.guard.RUnlock()
	return p.view.current
}

func (p *StaticPacer) GetCurrentEpoch() int32 {
	return p.GetCurrentView() % int32(len(p.Committee()))
}

func (p *StaticPacer) changeView(view int32) {
	p.view.guard.Lock()
	defer p.view.guard.Unlock()

	p.view.current = view

	p.NotifyEvent(api.Event{
		T:       api.ChangedView,
		Payload: view,
	})
}

func (p *StaticPacer) Synchronize(ctx context.Context) {
	log.Debugf("Synchronizing on current view %v", p.GetCurrentView())

	var s api.Sync
	view := p.GetCurrentView()

	if p.lastSynchronize != nil && p.lastSynchronize.Height() == view {
		s = p.lastSynchronize
	} else {
		if view == 0 {
			signedHash := p.protocol.blockchain.GetGenesisBlockSignedHash(p.me.GetPrivateKey())
			serialize := signedHash.Sign().Serialize()
			log.Debugf("current sync is for genesis, got signature %v", hexutils.BytesToHex(serialize[:]))
			s = CreateSync(0, true, p.protocol.hc, p.me)
		} else {
			voting := view > p.protocol.vheight
			if voting {
				p.protocol.vheight = view
			}
			s = CreateSync(view, voting, p.protocol.hc, p.me)
		}
		s.Sign(p.me.GetPrivateKey())
	}

	p.lastSynchronize = s

	payload := s.GetMessage()
	any, err := ptypes.MarshalAny(payload)
	if err != nil {
		log.Error("Can'T create Synchronize message", err)
		return
	}
	m := msg.CreateMessage(pb.Message_SYNCHRONIZE, any, s.Sender())

	go p.protocol.srv.Broadcast(ctx, m)
}

func (p *StaticPacer) OnSynchronize(ctx context.Context, m *msg.Message) error {
	p.sync.guard.Lock()
	defer p.sync.guard.Unlock()

	s, e := CreateSyncFromMessage(m)
	if e != nil {
		return e
	}

	//TODO try to remove this check and update signatures in qc, can be useful in voting qcs
	if p.protocol.hc != nil && p.protocol.hc.Height() >= s.Height() {
		return nil
	}

	if e := p.protocol.validateMessage(s, pb.Message_SYNCHRONIZE); e != nil {
		return e
	}

	actualHeight := int32(-1)
	if p.protocol.hc != nil {
		actualHeight = p.protocol.hc.Height()
	}

	p.sync.messageStorage[s.Sender().GetAddress()] = s
	if s.Voting() {
		p.sync.votingMessageStorage[s.Sender().GetAddress()] = s
		h, vAggr := p.aggregateSignature(p.sync.votingMessageStorage, actualHeight, true)
		if vAggr != nil {
			if h == 0 {
				p.protocol.FinishGenesisQCAndUpdate(vAggr)
			} else {
				p.protocol.FinishVotingSCAndUpdate(h, vAggr)
			}
			return nil
		}
	}

	h, aggr := p.aggregateSignature(p.sync.messageStorage, actualHeight, false)
	if aggr != nil {
		p.protocol.FinishSCAndUpdate(h, aggr)
	}

	return nil
}

func (p *StaticPacer) aggregateSignature(messageStorage map[common.Address]api.Sync, actualHeight int32, voting bool) (int32, *crypto.SignatureAggregate) {
	stats := make(map[int32]int32)
	for _, v := range messageStorage {
		stats[v.Height()] += 1
	}
	max := struct {
		h int32
		c int32
	}{0, 0}
	for k, v := range stats {
		if k > actualHeight && max.h < v {
			max.h = k
			max.c = v
		}
	}

	//We got quorum, lets start new epoch
	if int(max.c) >= (p.f/3)*2+1 {
		log.Debugf("Received %v synchronize messages, generating QC", max.c)
		signatures := make(map[common.Address]*crypto.Signature)
		for k, v := range messageStorage {
			if v.Height() == max.h {
				if voting {
					signatures[k] = v.VotingSignature()
				} else {
					signatures[k] = v.Signature()
				}
			}
		}

		var signs []*crypto.Signature
		bitmap := p.GetBitmap(signatures)
		for _, v := range signatures {
			signs = append(signs, v)
		}
		return max.h, crypto.AggregateSignatures(bitmap, signs)
	}

	return -1, nil
}

func (p *StaticPacer) GetBitmap(src map[common.Address]*crypto.Signature) *big.Int {
	var committee []common.Address
	for _, peer := range p.GetPeers() {
		committee = append(committee, peer.GetAddress())
	}

	return cmn.GetBitmap(src, committee)
}

func (p *StaticPacer) GetPeers() []*cmn.Peer {
	cpy := make([]*cmn.Peer, len(p.committee))
	copy(cpy, p.committee)
	return cpy
}

func (p *StaticPacer) FinishSyncAndUpdate(height int32, aggr *crypto.SignatureAggregate) {
	blockchain.CreateSynchronizeCertificate(aggr, height)
}
