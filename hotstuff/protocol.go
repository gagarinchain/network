package hotstuff

import (
	"errors"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/golang/protobuf/ptypes"
	"github.com/op/go-logging"
	bc "github.com/poslibp2p/blockchain"
	"github.com/poslibp2p/eth/common"
	msg "github.com/poslibp2p/message"
	"github.com/poslibp2p/message/protobuff"
	"github.com/poslibp2p/network"
	"math/rand"
	"strconv"
	"time"
)

var log = logging.MustGetLogger("hotstuff")

type Event struct {
	viewNumber int32
	etype      EventType
}

type EventType int

const (
	SUGGEST_PROPOSE EventType = iota
	NEXT_VIEW       EventType = iota
	START_EPOCH     EventType = iota
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
	OnEpochStart(m *msg.Message, s *msg.Peer)
	SubscribeEpochChange(trigger chan interface{})
}

type HQCHandler interface {
	Update(qc *bc.QuorumCertificate)
	HQC() *bc.QuorumCertificate
}

type ProtocolConfig struct {
	F               int
	Delta           time.Duration
	Blockchain      *bc.Blockchain
	Me              *msg.Peer
	Srv             network.Service
	Pacer           *StaticPacer
	CommitteeLoader msg.CommitteeLoader
	RoundEndChan    chan int32
	ControlChan     chan Event
}

type Protocol struct {
	f                   int
	delta               time.Duration
	blockchain          *bc.Blockchain
	currentView         int32
	vheight             int32
	currentEpoch        int32
	IsStartingEpoch     bool
	epochMessageStorage map[common.Address]int32
	epochStartSubChan   []chan interface{}
	votes               map[common.Address]*Vote
	lastExecutedBlock   *bc.Header
	hqc                 *bc.QuorumCertificate
	me                  *msg.Peer
	pacer               *StaticPacer
	srv                 network.Service
	controlChan         chan Event
	roundEndChan        chan int32
	stopChan            chan bool
}

func (p *Protocol) Vheight() int32 {
	return p.vheight
}

func CreateProtocol(cfg *ProtocolConfig) *Protocol {
	return &Protocol{
		f:                   cfg.F,
		delta:               cfg.Delta,
		blockchain:          cfg.Blockchain,
		vheight:             2,
		votes:               make(map[common.Address]*Vote),
		lastExecutedBlock:   cfg.Blockchain.GetGenesisBlock().Header(),
		hqc:                 cfg.Blockchain.GetGenesisCert(),
		me:                  cfg.Me,
		pacer:               cfg.Pacer,
		srv:                 cfg.Srv,
		roundEndChan:        cfg.RoundEndChan,
		stopChan:            make(chan bool),
		controlChan:         cfg.ControlChan,
		currentEpoch:        0,
		currentView:         3,
		IsStartingEpoch:     false,
		epochMessageStorage: make(map[common.Address]int32),
	}
}

//We return qref(qref(HQC_Block)) Pref block is the one that saw at least f+1 honest peers
func (p *Protocol) GetPref() *bc.Block {
	_, one, _ := p.blockchain.GetThreeChain(p.hqc.QrefBlock().Hash())
	return one
}

func (p *Protocol) CheckCommit() bool {
	log.Info("Check commit for", p.hqc.QrefBlock().Hash().Hex())
	zero, one, two := p.blockchain.GetThreeChain(p.hqc.QrefBlock().Hash())
	spew.Dump(zero)
	spew.Dump(one)
	spew.Dump(two)
	if two.Header().Parent() == one.Header().Hash() && one.Header().Parent() == zero.Header().Hash() {
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

func (p *Protocol) OnReceiveProposal(proposal *Proposal) error {
	//-At first i thought it should be equivocation, but later i found out that we must process all proposers
	// n. g. bad proposers can isolate us and make progress separately, the problem with it is that  in that case we have no way
	// to synchronize proposal rotation with others. Accepting all proposals we will build forks,
	// but it is ok, since we will have proof of equivocation later on and synchronize eventually
	//-It is not ok, we must store proposal that was sent out of order, to collect equivocation proofs

	p.Update(proposal.HQC)

	log.Info(p.pacer.GetCurrent(p.currentView).GetAddress().Hex())
	//TODO move this two validations
	if !proposal.Sender.Equals(p.pacer.GetCurrent(p.currentView)) {
		log.Warningf("This proposer [%v] is not expected", proposal.Sender.GetAddress().Hex())
		p.equivocate(proposal.Sender)
		return errors.New("peer equivocated")
	}
	if proposal.NewBlock.Header().Height() != p.currentView {
		log.Warningf("This proposer [%v] is expected to propose at height [%v], not on [%v]",
			proposal.Sender.GetAddress().Hex(), p.currentView, proposal.NewBlock.Header().Height())
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
		m := msg.CreateMessage(pb.Message_VOTE, any)

		willTrigger := p.pacer.WillNextViewForceEpochStart(p.currentView)
		if willTrigger {
			trigger := make(chan interface{})
			p.SubscribeEpochChange(trigger)
			go p.srv.SendMessageTriggered(p.pacer.GetNext(p.currentView), m, trigger)
		} else {
			go p.srv.SendMessage(p.pacer.GetNext(p.currentView), m)
		}
	}

	p.OnNextView()
	return nil
}

func (p *Protocol) OnReceiveVote(vote *Vote) error {
	p.Update(vote.HQC)

	if p.me != p.pacer.GetCurrent(p.currentView) && p.me != p.pacer.GetNext(p.currentView) {
		return errors.New(fmt.Sprintf("Got unexpected vote from [%v], i'm not proposer now", vote.Sender.GetAddress().Hex()))
	}

	addr := vote.Sender.GetAddress()

	if stored, ok := p.votes[addr]; ok {
		log.Infof("Already got vote for block [%v] from [%v]", stored.Header.Hash().Hex(), addr.Hex())

		//check whether peer voted previously for block with higher number
		if stored.Header.Height() > vote.Header.Height() {
			p.equivocate(vote.Sender)
			return errors.New("peer voted for block with lower number")
		}
		if stored.Header.Height() == vote.Header.Height() &&
			stored.Header.Hash() != vote.Header.Hash() {
			p.equivocate(vote.Sender)
			return errors.New("peer voted for different blocks on the same number")
		}
	}

	p.votes[addr] = vote

	if p.CheckConsensus() {
		p.FinishQC(vote.Header)
		_, loaded := p.blockchain.GetBlockByHashOrLoad(vote.Header.Hash())
		//rare case when we missed proposal, but we are next proposer and received all votes, in this case we go to next view and propose
		//this situation is the same as when we received proposal
		//TODO think about it again, mb it is safer to ignore votes and simply push next view after 2 deltas
		if loaded && p.me == p.pacer.GetNext(p.currentView) {
			p.OnNextView()
		}
		p.OnPropose() //propose random
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
func (p *Protocol) OnPropose() {
	if p.pacer.GetCurrent(p.currentView) != p.me {
		log.Info("Can't propose when we are not proposers")
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
	m := msg.CreateMessage(pb.Message_PROPOSAL, any)
	go p.srv.Broadcast(m)
	p.OnNextView()
}

func (*Protocol) equivocate(peer *msg.Peer) {
	log.Warningf("Peer [%v] equivocated", peer.GetAddress().Hex())
}

func (p *Protocol) FinishQC(header *bc.Header) {
	//Simply concatenate votes for now
	var aggregate []byte
	for _, v := range p.votes {
		aggregate = append(aggregate, v.Signature...)
	}
	p.hqc = bc.CreateQuorumCertificate(aggregate, header)
}

func (p *Protocol) HQC() *bc.QuorumCertificate {
	return p.hqc
}

func (p *Protocol) OnNextView() {
	p.changeView(p.currentView + 1)
}
func (p *Protocol) changeView(view int32) {
	p.currentView = view
	go func() {
		p.roundEndChan <- p.currentView
	}()
}
func (p *Protocol) StartEpoch(i int32) {
	p.IsStartingEpoch = true
	epoch := CreateEpoch(p.me, i, p.HQC())
	m, e := epoch.GetMessage()
	if e != nil {
		log.Error("Can't create Epoch message", e)
	}
	go p.srv.Broadcast(m)
}

func (p *Protocol) OnEpochStart(m *msg.Message, s *msg.Peer) {
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

	if max.n <= p.currentEpoch && int(max.c) == p.f/3+1 {
		//really impossible, because one peer is fair, and is not synchronized
		log.Fatal("Somehow we are ahead on epochs than f + 1 peers, it is impossible")
	}

	//We received at least 1 message from fair peer, should resynchronize our epoch
	if int(max.c) == p.f/3+1 {
		p.StartEpoch(max.n)
	}

	//We got quorum, lets start new epoch
	if int(max.c) == p.f/3*2+1 {
		p.newEpoch(max.n)
	}
}

func (p *Protocol) newEpoch(i int32) {
	if i > 1 {
		p.changeView((i - 1) * int32(p.f))
	}

	p.currentEpoch = i
	p.epochMessageStorage = make(map[common.Address]int32)
	for _, ch := range p.epochStartSubChan {
		go func(c chan interface{}) {
			c <- struct{}{}
		}(ch)
	}
	p.IsStartingEpoch = false
	log.Infof("Started new epoch %v", i)
	log.Infof("Current view number %v, proposer %v", p.currentView, p.pacer.GetCurrent(p.currentView).GetAddress().Hex())
}

func (p *Protocol) SubscribeEpochChange(trigger chan interface{}) {
	p.epochStartSubChan = append(p.epochStartSubChan, trigger)
}

func (p *Protocol) Run(msgChan chan *msg.Message) {
	for {
		select {
		case m := <-msgChan:
			if p.IsStartingEpoch && m.Type != pb.Message_EPOCH_START {
				log.Warningf("Received message [%v] while starting new epoch, skipping...", m.Type)
				break
			}

			switch m.Type {
			case pb.Message_VOTE:
				//put real sender here, we don't know ip, name
				v, err := CreateVoteFromMessage(m, &msg.Peer{})
				if err != nil {
					log.Error(err)
					break
				}
				if err := p.OnReceiveVote(v); err != nil {
					log.Error("Error while handling vote, ", err)
				}
			case pb.Message_PROPOSAL:
				pr, err := CreateProposalFromMessage(m, &msg.Peer{})
				if err != nil {
					log.Error(err)
					break
				}
				if err := p.OnReceiveProposal(pr); err != nil {
					log.Error("Error while handling proposal, ", err)
				}
			case pb.Message_EPOCH_START:
				p.OnEpochStart(m, &msg.Peer{})
			}
			//TODO fix race condition when timer triggers at the moment we changed view but not informed pacer yet
		case event := <-p.controlChan:
			switch event.etype {
			case SUGGEST_PROPOSE:
				p.OnPropose()
			case NEXT_VIEW:
				p.OnNextView()
			case START_EPOCH:
				p.StartEpoch(p.currentEpoch + 1)
			}
		case <-p.stopChan:
			log.Info("Stopping hotstuff...")
			return
		}

	}

}

func (p *Protocol) Stop() {
	p.stopChan <- true
}

func (p *Protocol) GetCurrentView() int32 {
	return p.currentView
}
