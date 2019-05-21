package hotstuff

import (
	"context"
	"errors"
	"fmt"
	"github.com/golang/protobuf/ptypes"
	"github.com/op/go-logging"
	"github.com/poslibp2p"
	bc "github.com/poslibp2p/blockchain"
	comm "github.com/poslibp2p/common"
	"github.com/poslibp2p/common/eth/common"
	msg "github.com/poslibp2p/common/message"
	"github.com/poslibp2p/common/protobuff"
	"github.com/poslibp2p/network"
	"math/rand"
	"strconv"
	"time"
)

var log = logging.MustGetLogger("hotstuff")

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

type ProtocolConfig struct {
	F          int
	Delta      time.Duration
	Blockchain *bc.Blockchain
	Me         *comm.Peer
	Srv        network.Service
	Sync       bc.Synchronizer
	Pacer      Pacer
	Validators []poslibp2p.Validator
	Storage    bc.Storage
	Committee  []*comm.Peer
}

type Protocol struct {
	f                 int
	blockchain        *bc.Blockchain
	vheight           int32
	votes             map[common.Address]*Vote
	lastExecutedBlock *bc.Header
	hqc               *bc.QuorumCertificate
	me                *comm.Peer
	pacer             Pacer
	validators        []poslibp2p.Validator
	storage           bc.Storage //we can eliminate this dependency, setting value via epochStartSubChan and setting via conf, mb refactor in the future
	srv               network.Service
	sync              bc.Synchronizer
}

//func (p Protocol) String() string {
//	return fmt.Sprintf("%b", b)
//}

func (p *Protocol) Vheight() int32 {
	return p.vheight
}

func CreateProtocol(cfg *ProtocolConfig) *Protocol {
	return &Protocol{
		f:                 cfg.F,
		blockchain:        cfg.Blockchain,
		vheight:           0,
		votes:             make(map[common.Address]*Vote),
		lastExecutedBlock: cfg.Blockchain.GetGenesisBlock().Header(),
		hqc:               cfg.Blockchain.GetGenesisCert(),
		me:                cfg.Me,
		pacer:             cfg.Pacer,
		validators:        cfg.Validators,
		storage:           cfg.Storage,
		srv:               cfg.Srv,
		sync:              cfg.Sync,
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

	log.Infof("current proposer %v", p.pacer.GetCurrent().GetAddress().Hex())

	//TODO move this two validations
	if !proposal.Sender.Equals(p.pacer.GetCurrent()) {
		log.Warningf("This proposer [%v] is not expected", proposal.Sender.GetAddress().Hex())
		p.equivocate(proposal.Sender)
		return errors.New("peer equivocated")
	}
	if proposal.NewBlock.Header().Height() != p.getCurrentView() {
		log.Warningf("This proposer [%v] is expected to propose at height [%v], not on [%v]",
			proposal.Sender.GetAddress().Hex(), p.getCurrentView(), proposal.NewBlock.Header().Height())
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
		go p.srv.SendMessage(ctx, p.pacer.GetNext(), m)
	}

	p.pacer.FireEvent(Voted)
	return nil
}

func (p *Protocol) OnReceiveVote(ctx context.Context, vote *Vote) error {
	log.Debugf("Received vote for block on height [%v]", vote.Header.Height())
	p.Update(vote.HQC)

	if p.me.GetAddress() != p.pacer.GetCurrent().GetAddress() && p.me.GetAddress() != p.pacer.GetNext().GetAddress() {
		return errors.New(fmt.Sprintf("Got unexpected vote from [%v], i'm not proposer now", vote.Sender.GetAddress().Hex()))
	}

	addr := vote.Sender.GetAddress()

	//todo add vote validation
	if stored, ok := p.votes[addr]; ok {
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
		p.blockchain.GetBlockByHashOrLoad(ctx, vote.Header.Hash())
		p.FinishQC(vote.Header)
		p.pacer.FireEvent(VotesCollected)
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
	if !p.pacer.GetCurrent().Equals(p.me) {
		log.Debug("Not my turn to propose, skipping")
		return
	}
	log.Debug("We are proposer, proposing")
	//TODO    write test to fail on signature
	head := p.blockchain.GetBlockByHash(p.HQC().QrefBlock().Hash())
	blocksToAdd := p.getCurrentView() - 1 - head.Header().Height()

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
	m := msg.CreateMessage(pb.Message_PROPOSAL, any, p.me)
	go p.srv.Broadcast(ctx, m)

	p.pacer.FireEvent(Proposed)
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
	p.Update(bc.CreateQuorumCertificate(aggregate, header))
	log.Debugf("Generated new QC for %v on height %v", header.Hash().Hex(), header.Height())
}

func (p *Protocol) FinishGenesisQC(aggregate []byte) {
	p.blockchain.UpdateGenesisBlockQC(bc.CreateQuorumCertificate(aggregate, p.blockchain.GetGenesisBlock().Header()))
	p.hqc = p.blockchain.GetGenesisCert()
}

func (p *Protocol) HQC() *bc.QuorumCertificate {
	return p.hqc
}

func (p *Protocol) validateMessage(entity interface{}, messageType pb.Message_MessageType) error {
	for _, v := range p.validators {
		if v.Supported(messageType) {
			isValid, e := v.IsValid(entity)
			if e != nil {
				return e
			}
			if !isValid {
				return e
			}
		}
	}

	return nil
}
func (p *Protocol) validate(proposal *Proposal) error {
	if proposal.NewBlock == nil {
		return errors.New("proposed block can't be empty")
	}

	return nil
}

func (p *Protocol) handleMessage(ctx context.Context, m *msg.Message) error {
	switch m.Type {
	case pb.Message_VOTE:
		log.Debugf("received vote")
		v, e := CreateVoteFromMessage(m)
		if e != nil {
			return e
		}
		if e := p.validateMessage(v, pb.Message_VOTE); e != nil {
			return e
		}
		if err := p.OnReceiveVote(ctx, v); err != nil {
			return err
		}
	case pb.Message_PROPOSAL:
		log.Debugf("received proposal")
		pr, err := CreateProposalFromMessage(m)
		if err != nil {
			return err
		}

		if e := p.validateMessage(pr, pb.Message_PROPOSAL); e != nil {
			return e
		}

		parent := pr.NewBlock.Header().Parent()
		if !p.blockchain.Contains(parent) {
			log.Debugf("Requesting for fork starting at block %v at height %v", parent.Hex(), pr.NewBlock.Height()-1)
			err := p.sync.RequestFork(ctx, parent, pr.Sender)
			if err != nil {
				return err
			}
		}

		if err := p.OnReceiveProposal(ctx, pr); err != nil {
			return err
		}
	}

	return nil
}

func (p *Protocol) getCurrentView() int32 {
	return p.pacer.GetCurrentView()
}
