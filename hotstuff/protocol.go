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

type Event int

const (
	PROPOSE     Event = iota
	EMPTY_BLOCK Event = iota
)

type ProtocolConfig struct {
	F               int
	Delta           time.Duration
	Blockchain      *bc.Blockchain
	Me              *msg.Peer
	Srv             network.Service
	Pacer           *StaticPacer
	CommitteeLoader msg.CommitteeLoader
	RoundEndChan    chan interface{}
	ControlChan     chan Event
}

type Protocol struct {
	f          int
	delta      time.Duration
	blockchain *bc.Blockchain
	// number of last voted block, the number of genesis block is 2
	vheight           int32
	votes             map[common.Address]*Vote
	lastExecutedBlock *bc.Header
	hqc               *bc.QuorumCertificate
	me                *msg.Peer
	pacer             *StaticPacer
	srv               network.Service
	controlChan       chan Event
	roundEndChan      chan interface{}
	stopChan          chan bool
}

func (p *Protocol) Vheight() int32 {
	return p.vheight
}

func CreateProtocol(cfg *ProtocolConfig) *Protocol {
	return &Protocol{
		f:                 cfg.F,
		delta:             cfg.Delta,
		blockchain:        cfg.Blockchain,
		vheight:           2,
		votes:             make(map[common.Address]*Vote),
		lastExecutedBlock: cfg.Blockchain.GetGenesisBlock().Header(),
		hqc:               cfg.Blockchain.GetGenesisCert(),
		me:                cfg.Me,
		pacer:             cfg.Pacer,
		srv:               cfg.Srv,
		roundEndChan:      cfg.RoundEndChan,
		stopChan:          make(chan bool),
		controlChan:       cfg.ControlChan,
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

	log.Info(p.pacer.GetCurrent().GetAddress().Hex())
	//TODO move this two validations
	if !proposal.Sender.Equals(p.pacer.GetCurrent()) {
		log.Warningf("This proposer [%v] is not expected", proposal.Sender.GetAddress().Hex())
		p.equivocate(proposal.Sender)
		return errors.New("peer equivocated")
	}
	if proposal.NewBlock.Header().Height() != p.pacer.GetCurrentHeight() {
		log.Warningf("This proposer [%v] is expected to propose at height [%v], not on [%v]",
			proposal.Sender.GetAddress().Hex(), p.pacer.GetCurrentHeight(), proposal.NewBlock.Header().Height())
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

		peer, isSync := p.pacer.CurrentProposerOrStartEpoch()
		if isSync {
			trigger := make(chan interface{})
			p.pacer.SubscribeEpochChange(trigger)
			go p.srv.SendMessageTriggered(peer, m, trigger)
		} else {
			go p.srv.SendMessage(peer, m)
		}
	}

	p.roundEndChan <- struct{}{}
	return nil
}

func (p *Protocol) OnReceiveVote(vote *Vote) error {
	p.Update(vote.HQC)

	if p.me != p.pacer.GetCurrent() && p.me != p.pacer.GetNext() {
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
		p.blockchain.GetBlockByHashOrLoad(vote.Header.Hash())
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
	//TODO    write test to fail on signature
	block := p.blockchain.NewBlock(p.blockchain.GetHead(), p.hqc, []byte(strconv.Itoa(rand.Int())))
	proposal := CreateProposal(block, p.hqc, p.me)

	proposal.Sign(p.me.GetPrivateKey())

	payload := proposal.GetMessage()
	any, e := ptypes.MarshalAny(payload)
	if e != nil {
		log.Error(e)
	}
	m := msg.CreateMessage(pb.Message_PROPOSAL, any)
	go p.srv.Broadcast(m)
	p.roundEndChan <- struct{}{}
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

	//we get new QC, it means it is a good time to propose
}

func (p *Protocol) HQC() *bc.QuorumCertificate {
	return p.hqc
}

func (p *Protocol) Run(msgChan chan *msg.Message) {
	for {
		select {
		case m := <-msgChan:
			if p.pacer.IsStarting {
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
					log.Error("Error while handling vote", err)
				}
			case pb.Message_PROPOSAL:
				pr, err := CreateProposalFromMessage(m, &msg.Peer{})
				if err != nil {
					log.Error(err)
					break
				}
				if err := p.OnReceiveProposal(pr); err != nil {
					log.Error("Error while handling vote", err)
				}
			case pb.Message_EPOCH_START:
				p.pacer.OnEpochStart(m, &msg.Peer{})
			}
		case event := <-p.controlChan:
			switch event {
			case PROPOSE:
				p.OnPropose()
			case EMPTY_BLOCK:
				p.OnEmptyBlock()
			}
		case <-p.stopChan:
			log.Info("Stopping pacer...")
			return
		}

	}

}

func (p *Protocol) Delta() time.Duration {
	return p.delta
}

func (p *Protocol) Stop() {
	p.stopChan <- true
}

func (p *Protocol) OnEmptyBlock() {
	p.blockchain.PadEmptyBlock()
	p.roundEndChan <- struct{}{}
}
