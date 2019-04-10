package hotstuff

import (
	"context"
	"errors"
	"fmt"
	"github.com/golang/protobuf/ptypes"
	"github.com/op/go-logging"
	bc "github.com/poslibp2p/blockchain"
	comm "github.com/poslibp2p/common"
	"github.com/poslibp2p/common/eth/common"
	"github.com/poslibp2p/common/protobuff"
	msg "github.com/poslibp2p/message"
	"github.com/poslibp2p/network"
	"math/rand"
	"strconv"
	"sync"
	"time"
)

var log = logging.MustGetLogger("hotstuff")

type Command struct {
	ctx       context.Context
	eventType CommandType
}

func NewCommand(ctx context.Context, eventType CommandType) *Command {
	return &Command{ctx: ctx, eventType: eventType}
}

type CommandType int

const (
	SuggestPropose CommandType = iota
	SuggestVote    CommandType = iota
	NextView       CommandType = iota
	StartEpoch     CommandType = iota
)

type Event int

const (
	StartedEpoch Event = iota
	ChangedView  Event = iota
)

type Proposer interface {
	OnReceiveVote(vote *Vote) error
	CheckConsensus() bool
	OnPropose()
	FinishQC(header *bc.Header)
}

type Replica interface {
	OnReceiveProposal(proposal *Proposal) error
	CheckCommit() bool
	OnNextView()
	StartEpoch(i int32)
	OnEpochStart(m *msg.Message, s *comm.Peer)
	SubscribeEpochChange(trigger chan interface{})
}

type HQCHandler interface {
	Update(qc *bc.QuorumCertificate)
	HQC() *bc.QuorumCertificate
}

type CurrentViewGetter interface {
	GetCurrentView() int32
}

type EventNotifier interface {
	SubscribeProtocolEvents(chan Event)
}

type ProtocolConfig struct {
	F           int
	Delta       time.Duration
	Blockchain  *bc.Blockchain
	Me          *comm.Peer
	Srv         network.Service
	Sync        bc.Synchronizer
	Pacer       *StaticPacer
	Storage     bc.Storage
	Committee   []*comm.Peer
	ControlChan chan Command
}

type Protocol struct {
	f                     int
	delta                 time.Duration
	blockchain            *bc.Blockchain
	currentView           int32
	currentViewGuard      *sync.RWMutex
	vheight               int32
	currentEpoch          int32
	IsStartingEpoch       bool
	epochMessageStorage   map[common.Address]int32
	epochVoteStorage      map[common.Address][]byte
	epochStartSubChan     []chan interface{}
	protocolEventSubChans []chan Event
	votes                 map[common.Address]*Vote
	lastExecutedBlock     *bc.Header
	hqc                   *bc.QuorumCertificate
	me                    *comm.Peer
	pacer                 *StaticPacer
	storage               bc.Storage //we can eliminate this dependency, setting value via epochStartSubChan and setting via conf, mb refactor in the future
	srv                   network.Service
	sync                  bc.Synchronizer
	controlChan           chan Command
	stopChan              chan bool
}

func (p *Protocol) Vheight() int32 {
	return p.vheight
}

func CreateProtocol(cfg *ProtocolConfig) *Protocol {
	val, err := cfg.Storage.GetCurrentEpoch()
	if err != nil {
		log.Warning("Can't load current epoch from storage")
	}
	return &Protocol{
		f:                   cfg.F,
		delta:               cfg.Delta,
		blockchain:          cfg.Blockchain,
		vheight:             0,
		votes:               make(map[common.Address]*Vote),
		lastExecutedBlock:   cfg.Blockchain.GetGenesisBlock().Header(),
		hqc:                 cfg.Blockchain.GetGenesisCert(),
		me:                  cfg.Me,
		pacer:               cfg.Pacer,
		storage:             cfg.Storage,
		srv:                 cfg.Srv,
		sync:                cfg.Sync,
		stopChan:            make(chan bool),
		controlChan:         cfg.ControlChan,
		currentEpoch:        val,
		currentView:         1,
		currentViewGuard:    &sync.RWMutex{},
		IsStartingEpoch:     false,
		epochMessageStorage: make(map[common.Address]int32),
		epochVoteStorage:    make(map[common.Address][]byte),
	}
}

//We return qref(qref(HQC_Block))
func (p *Protocol) GetPref() *bc.Block {
	_, one, _ := p.blockchain.GetThreeChain(p.hqc.QrefBlock().Hash())
	return one
}

func (p *Protocol) CheckCommit() bool {
	log.Info("Check commit for", p.hqc.QrefBlock().Hash().Hex())
	zero, one, two := p.blockchain.GetThreeChain(p.hqc.QrefBlock().Hash())
	if two.Header().Parent() == one.Header().Hash() && one.Header().Parent() == zero.Header().Hash() {
		log.Debugf("Committing block %v height %v", zero.Header().Hash().Hex(), zero.Height())
		p.blockchain.OnCommit(zero)
		return true
	}

	return false
}

func (p *Protocol) Update(qc *bc.QuorumCertificate) {
	//if new block has cert to block with greater number means that it is new HQC block
	// and we must consider it as new HEAD

	log.Infof("Qcs new [%v], old[%v]", qc.QrefBlock().Height(), p.hqc.QrefBlock().Height())

	if qc.QrefBlock().Height() > p.hqc.QrefBlock().Height() {

		log.Infof("Got new HQC block[%v], updating number [%v] -> [%v]",
			qc.QrefBlock().Hash().Hex(), p.hqc.QrefBlock().Height(), qc.QrefBlock().Height())

		p.hqc = qc
		p.CheckCommit()
	}
}

func (p *Protocol) OnReceiveProposal(ctx context.Context, proposal *Proposal) error {
	p.Update(proposal.HQC)

	log.Info(p.pacer.GetCurrent(p.GetCurrentView()).GetAddress().Hex())
	//TODO move this two validations
	if !proposal.Sender.Equals(p.pacer.GetCurrent(p.GetCurrentView())) {
		log.Warningf("This proposer [%v] is not expected", proposal.Sender.GetAddress().Hex())
		p.equivocate(proposal.Sender)
		return errors.New("peer equivocated")
	}
	if proposal.NewBlock.Header().Height() != p.GetCurrentView() {
		log.Warningf("This proposer [%v] is expected to propose at height [%v], not on [%v]",
			proposal.Sender.GetAddress().Hex(), p.GetCurrentView(), proposal.NewBlock.Header().Height())
		p.equivocate(proposal.Sender)
		return errors.New("peer equivocated")
	}

	if err := p.blockchain.AddBlock(proposal.NewBlock); err != nil {
		return err
	}

	if proposal.NewBlock.Header().Height() > p.vheight && p.blockchain.IsSibling(proposal.NewBlock.Header(), p.GetPref().Header()) {
		log.Infof("Received proposal for block [%v] with higher number [%v] from proposer [%v], voting for it",
			proposal.NewBlock.Header().Hash().Hex(), proposal.NewBlock.Header().Height(), proposal.Sender.GetAddress().Hex())
		p.vheight = proposal.NewBlock.Header().Height()

		newBlock := p.blockchain.GetBlockByHash(proposal.NewBlock.Header().Hash())
		vote := CreateVote(newBlock.Header(), p.hqc, p.me)
		vote.Sign(p.me.GetPrivateKey())

		vMsg := vote.GetMessage()
		any, e := ptypes.MarshalAny(vMsg)
		if e != nil {
			log.Error(e)
		}
		m := msg.CreateMessage(pb.Message_VOTE, any, p.me)

		willTrigger := p.pacer.WillNextViewForceEpochStart(p.GetCurrentView())
		if willTrigger {
			trigger := make(chan interface{})
			p.SubscribeEpochChange(trigger)
			go p.srv.SendMessageTriggered(ctx, p.pacer.GetNext(p.GetCurrentView()), m, trigger)
		} else {
			go p.srv.SendMessage(ctx, p.pacer.GetNext(p.GetCurrentView()), m)
		}
	}

	p.OnNextView()
	return nil
}

func (p *Protocol) OnReceiveVote(ctx context.Context, vote *Vote) error {
	log.Debugf("Received vote for block on height [%v]", vote.Header.Height())
	p.Update(vote.HQC)

	if p.me != p.pacer.GetCurrent(p.GetCurrentView()) && p.me != p.pacer.GetNext(p.GetCurrentView()) {
		return errors.New(fmt.Sprintf("Got unexpected vote from [%v], i'm not proposer now", vote.Sender.GetAddress().Hex()))
	}

	addr := vote.Sender.GetAddress()

	if stored, ok := p.votes[addr]; ok {
		log.Infof("Already got vote for block [%v] from [%v]", stored.Header.Hash().Hex(), addr.Hex())

		//check whether peer voted previously for block with higher number
		if stored.Header.Height() > vote.Header.Height() {
			p.equivocate(vote.Sender)
			return errors.New("peer voted for block with lower height")
		}
		if stored.Header.Height() == vote.Header.Height() &&
			stored.Header.Hash() != vote.Header.Hash() {
			p.equivocate(vote.Sender)
			return errors.New("peer voted for different blocks on the same height")
		}
	}

	p.votes[addr] = vote

	if p.CheckConsensus() {
		p.FinishQC(vote.Header)
		//todo load blocks earlier maybe
		_, loaded := p.blockchain.GetBlockByHashOrLoad(vote.Header.Hash())
		//rare case when we missed proposal, but we are next proposer and received all votes, in this case we go to next view and propose
		//this situation is the same as when we received proposal
		//TODO think about it again, mb it is safer to ignore votes and simply push next view after 2 deltas
		if loaded && p.me == p.pacer.GetNext(p.GetCurrentView()) {
			p.OnNextView()
		}
		p.OnPropose(ctx)
	}
	return nil
}

func (p *Protocol) CheckConsensus() bool {
	type stat struct {
		score  int
		header *bc.Header
	}
	stats := make(map[common.Hash]*stat, len(p.votes))
	for _, each := range p.votes {
		if s, ok := stats[each.Header.Hash()]; !ok {
			stats[each.Header.Hash()] = &stat{score: 1, header: each.Header}
		} else {
			s.score += 1
		}
	}

	bestStat := &stat{}
	secondBestStat := &stat{}

	for _, v := range stats {
		if v.score > bestStat.score {
			bestStat = v
			secondBestStat = &stat{}
		} else if v.score == bestStat.score {
			secondBestStat = v
		}
	}

	if secondBestStat.score >= (p.f/3)*2+1 {
		panic(fmt.Sprintf("at list two blocks [%v], [%v]  got consensus score, reload chain than go on",
			bestStat.header, secondBestStat.header))
	}

	if bestStat.score >= (p.f/3)*2+1 {
		return true
	}

	return false
}

//We must propose block atop preferred block.  "It then chooses to extend a branch from the Preferred Block
//determined by it."
func (p *Protocol) OnPropose(ctx context.Context) {
	if p.pacer.GetCurrent(p.GetCurrentView()) != p.me {
		log.Debug("Can't propose when we are not proposers")
		return
	}
	log.Debug("We are proposer, proposing")
	//TODO    write test to fail on signature
	head := p.blockchain.GetBlockByHash(p.HQC().QrefBlock().Hash())
	blocksToAdd := p.GetCurrentView() - 1 - head.Header().Height()

	log.Debugf("Padding with %v empty blocks", blocksToAdd)

	for i := 0; i < int(blocksToAdd); i++ {
		head = p.blockchain.PadEmptyBlock(head)
	}

	block := p.blockchain.NewBlock(head, p.hqc, []byte(strconv.Itoa(rand.Int())))
	proposal := CreateProposal(block, p.hqc, p.me)

	proposal.Sign(p.me.GetPrivateKey())

	payload := proposal.GetMessage()
	any, e := ptypes.MarshalAny(payload)
	if e != nil {
		log.Error(e)
	}
	m := msg.CreateMessage(pb.Message_PROPOSAL, any, nil)
	go p.srv.Broadcast(ctx, m)
	p.OnNextView()
}

func (*Protocol) equivocate(peer *comm.Peer) {
	log.Warningf("Peer [%v] equivocated", peer.GetAddress().Hex())
}

func (p *Protocol) FinishQC(header *bc.Header) {
	//Simply concatenate votes for now
	var aggregate []byte
	for _, v := range p.votes {
		aggregate = append(aggregate, v.Signature...)
	}
	p.hqc = bc.CreateQuorumCertificate(aggregate, header)
	log.Debugf("Generated new QC for %v on height %v", header.Hash().Hex(), header.Height())
}

func (p *Protocol) FinishGenesisQC(header *bc.Header) {
	//Simply concatenate votes for now
	var aggregate []byte
	for _, v := range p.epochVoteStorage {
		aggregate = append(aggregate, v...)
	}
	p.blockchain.GetGenesisBlock().SetQC(bc.CreateQuorumCertificate(aggregate, header))
	p.hqc = p.blockchain.GetGenesisCert()
}

func (p *Protocol) HQC() *bc.QuorumCertificate {
	return p.hqc
}

func (p *Protocol) OnNextView() {
	p.changeView(p.GetCurrentView() + 1)
	p.notifyProtocolEvent(ChangedView)
}

func (p *Protocol) changeView(view int32) {
	p.currentViewGuard.Lock()
	defer p.currentViewGuard.Unlock()

	p.currentView = view
}
func (p *Protocol) StartEpoch(ctx context.Context, i int32) {
	p.IsStartingEpoch = true
	var epoch *Epoch
	//todo think about moving it to epoch
	if p.currentEpoch == 0 {
		epoch = CreateEpoch(p.me, i, nil, p.blockchain.GetGenesisBlockSignedHash(p.me.GetPrivateKey()))
	} else {
		epoch = CreateEpoch(p.me, i, p.HQC(), nil)
	}

	m, e := epoch.GetMessage()
	if e != nil {
		log.Error("Can't create Epoch message", e)
	}
	go p.srv.Broadcast(ctx, m)
}

func (p *Protocol) OnEpochStart(ctx context.Context, m *msg.Message) {
	epoch, e := CreateEpochFromMessage(m)
	if e != nil {
		log.Error(e)
		return
	}

	if epoch.number < p.currentEpoch+1 {
		log.Warning("received epoch message for previous epoch ", epoch.number)
		return
	}

	if epoch.genesisSignature != nil {
		res := p.blockchain.ValidateGenesisBlockSignature(epoch.genesisSignature, epoch.sender.GetAddress())
		if !res {
			log.Errorf("Peer %v sent wrong genesis block signature", epoch.sender.GetAddress().Hex())
			p.equivocate(epoch.sender)
			return
		}
		p.epochVoteStorage[epoch.sender.GetAddress()] = epoch.genesisSignature
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

	if max.n <= p.currentEpoch && int(max.c) == p.f/3+1 {
		//really impossible, because one peer is fair, and is not synchronized
		log.Fatal("Somehow we are ahead on epochs than f + 1 peers, it is impossible")
	}

	//We received at least 1 message from fair peer, should resynchronize our epoch
	if int(max.c) == p.f/3+1 {
		p.StartEpoch(ctx, max.n)
	}

	//We got quorum, lets start new epoch
	if int(max.c) == p.f/3*2+1 {
		if max.n == 1 {
			p.FinishGenesisQC(p.blockchain.GetGenesisBlock().Header())
		}
		p.newEpoch(max.n)
	}
}

func (p *Protocol) newEpoch(i int32) {
	if i > 1 {
		p.changeView((i - 1) * int32(p.f))
	}
	e := p.storage.PutCurrentEpoch(i)
	if e != nil {
		log.Error(e)
	}

	p.currentEpoch = i
	p.epochMessageStorage = make(map[common.Address]int32)
	p.notifyEpochChange()

	p.IsStartingEpoch = false
	log.Infof("Started new epoch %v", i)
	log.Infof("Current view number %v, proposer %v", p.currentView, p.pacer.GetCurrent(p.currentView).GetAddress().Hex())
	p.notifyProtocolEvent(StartedEpoch)
}

func (p *Protocol) SubscribeEpochChange(trigger chan interface{}) {
	p.epochStartSubChan = append(p.epochStartSubChan, trigger)
}
func (p *Protocol) notifyEpochChange() {
	for _, ch := range p.epochStartSubChan {
		go func(c chan interface{}) {
			c <- struct{}{}
		}(ch)
	}
}

//if we will need unsubscribe we can refactor list to map and identify subscribers
func (p *Protocol) SubscribeProtocolEvents(sub chan Event) {
	p.protocolEventSubChans = append(p.protocolEventSubChans, sub)
}
func (p *Protocol) notifyProtocolEvent(event Event) {
	for _, ch := range p.protocolEventSubChans {
		go func(c chan Event) {
			c <- event
		}(ch)
	}
}

func (p *Protocol) Run(msgChan chan *msg.Message) {
	log.Info("Starting hotstuff protocol...")

	//bootstrap command
	event := <-p.controlChan
	ctx := event.ctx
	p.handleControlEvent(event)

	for {
		select {
		case m := <-msgChan:
			if p.IsStartingEpoch && m.Type != pb.Message_EPOCH_START {
				log.Warningf("Received message [%v] while starting new epoch, skipping...", m.Type)
				break
			}

			switch m.Type {
			case pb.Message_VOTE:
				log.Debugf("received vote")

				v, err := CreateVoteFromMessage(m)
				if err != nil {
					log.Error(err)
					break
				}
				timeout, _ := context.WithTimeout(ctx, p.delta)
				if err := p.OnReceiveVote(timeout, v); err != nil {
					log.Error("Error while handling vote, ", err)
				}
			case pb.Message_PROPOSAL:
				log.Debugf("received proposal")
				pr, err := CreateProposalFromMessage(m)
				if err != nil {
					log.Error(err)
					break
				}
				parent := pr.NewBlock.Header().Parent()
				if !p.blockchain.Contains(parent) {
					err := p.sync.RequestFork(ctx, parent, pr.Sender)
					if err != nil {
						log.Error("Error while processing proposal", err)
						continue
					}
				}

				p.validate(pr)
				timeout, _ := context.WithTimeout(ctx, p.delta)
				if err := p.OnReceiveProposal(timeout, pr); err != nil {
					log.Error("Error while handling proposal, ", err)
				}
			case pb.Message_EPOCH_START:
				timeout, _ := context.WithTimeout(ctx, p.delta)
				p.OnEpochStart(timeout, m)
			}
		case event := <-p.controlChan:
			ctx = event.ctx
			p.handleControlEvent(event)
		case <-p.stopChan:
			log.Info("Stopping hotstuff...")

			return
		}
	}
}

func (p *Protocol) handleControlEvent(event Command) {
	switch event.eventType {
	case SuggestPropose:
		p.OnPropose(event.ctx)
	case NextView:
		p.OnNextView()
	case StartEpoch:
		p.StartEpoch(event.ctx, p.currentEpoch+1)
	}
}

func (p *Protocol) Stop() {
	p.stopChan <- true
}

func (p *Protocol) GetCurrentView() int32 {
	p.currentViewGuard.RLock()
	defer p.currentViewGuard.RUnlock()
	return p.currentView
}

func (p *Protocol) validate(proposal *Proposal) error {
	if proposal.NewBlock == nil {
		return errors.New("proposed block can't be empty")
	}

	return nil
}
