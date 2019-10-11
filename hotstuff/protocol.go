package hotstuff

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	main "github.com/gagarinchain/network"
	bc "github.com/gagarinchain/network/blockchain"
	comm "github.com/gagarinchain/network/common"
	"github.com/gagarinchain/network/common/eth/common"
	"github.com/gagarinchain/network/common/eth/crypto"
	msg "github.com/gagarinchain/network/common/message"
	"github.com/gagarinchain/network/common/protobuff"
	"github.com/gagarinchain/network/network"
	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/op/go-logging"
	"math/rand"
	"strconv"
	"time"
)

var (
	log = logging.MustGetLogger("hotstuff")
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

type ProtocolConfig struct {
	F            int
	Delta        time.Duration
	Blockchain   *bc.Blockchain
	Me           *comm.Peer
	Srv          network.Service
	Sync         bc.Synchronizer
	Pacer        Pacer
	Validators   []main.Validator
	Storage      main.Storage
	Committee    []*comm.Peer
	InitialState *InitialState
}

type InitialState struct {
	View              int32
	Epoch             int32
	VHeight           int32
	LastExecutedBlock *bc.Header
	HQC               *bc.QuorumCertificate
}

func DefaultState(bc *bc.Blockchain) *InitialState {
	return &InitialState{
		View:              int32(0),
		Epoch:             int32(-1),
		VHeight:           0,
		LastExecutedBlock: bc.GetGenesisBlock().Header(),
		HQC:               bc.GetGenesisBlock().QC(),
	}
}

type Protocol struct {
	f                 int
	blockchain        *bc.Blockchain
	vheight           int32
	lastVote          *msg.Message
	votes             map[common.Address]*Vote
	lastExecutedBlock *bc.Header
	hqc               *bc.QuorumCertificate
	me                *comm.Peer
	pacer             Pacer
	validators        []main.Validator
	srv               network.Service
	sync              bc.Synchronizer
	persister         *ProtocolPersister
}

type ProtocolPersister struct {
	Storage main.Storage
}

func (pp *ProtocolPersister) PutVHeight(vheight int32) error {
	vhb := comm.Int32ToByte(vheight)
	return pp.Storage.Put(main.VHeight, nil, vhb)
}
func (pp *ProtocolPersister) GetVHeight() (int32, error) {
	value, err := pp.Storage.Get(main.VHeight, nil)
	if err != nil {
		return comm.DefaultIntValue, err
	}
	return comm.ByteToInt32(value)
}

func (pp *ProtocolPersister) PutLastExecutedBlockHash(hash common.Hash) error {
	return pp.Storage.Put(main.LastExecutedBlock, nil, hash.Bytes())
}
func (pp *ProtocolPersister) GetLastExecutedBlockHash() (common.Hash, error) {
	value, err := pp.Storage.Get(main.LastExecutedBlock, nil)
	if err != nil {
		return common.Hash{}, err
	}
	return common.BytesToHash(value), nil
}

func (pp *ProtocolPersister) PutHQC(hqc *bc.QuorumCertificate) error {
	m := hqc.GetMessage()
	bytes, e := proto.Marshal(m)
	if e != nil {
		return e
	}
	return pp.Storage.Put(main.HQC, nil, bytes)
}
func (pp *ProtocolPersister) GetHQC() (*bc.QuorumCertificate, error) {
	value, err := pp.Storage.Get(main.HQC, nil)
	if err != nil {
		return nil, err
	}
	pbqc := &pb.QuorumCertificate{}
	if err := proto.Unmarshal(value, pbqc); err != nil {
		return nil, err
	}
	return bc.CreateQuorumCertificateFromMessage(pbqc), nil
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
		vheight:           cfg.InitialState.VHeight,
		votes:             make(map[common.Address]*Vote),
		lastExecutedBlock: cfg.InitialState.LastExecutedBlock,
		hqc:               cfg.InitialState.HQC,
		me:                cfg.Me,
		pacer:             cfg.Pacer,
		validators:        cfg.Validators,
		srv:               cfg.Srv,
		sync:              cfg.Sync,
		persister:         &ProtocolPersister{Storage: cfg.Storage},
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
	if bytes.Equal(two.Header().Parent().Bytes(), one.Header().Hash().Bytes()) &&
		bytes.Equal(one.Header().Parent().Bytes(), zero.Header().Hash().Bytes()) {
		log.Debugf("Committing block %v height %v", zero.Header().Hash().Hex(), zero.Height())
		toCommit, _, err := p.blockchain.OnCommit(zero)
		if err != nil {
			log.Error(err)
			return false
		}
		p.lastExecutedBlock = toCommit[len(toCommit)-1].Header()
		if err := p.persister.PutLastExecutedBlockHash(p.lastExecutedBlock.Hash()); err != nil {
			log.Error(err)
			return false
		}
		return true
	}

	return false
}

func (p *Protocol) Update(qc *bc.QuorumCertificate) {
	//if new block has cert to block with greater number means that it is new HQC block
	// and we must consider it as new HEAD

	log.Infof("Qcs new [%v], old[%v]", qc.QrefBlock().Height(), p.hqc.QrefBlock().Height())

	if qc.QrefBlock().Height() > p.hqc.QrefBlock().Height() {
		b, e := qc.IsValid(qc.GetHash(), comm.PeersToPubs(p.pacer.GetPeers()))
		if !b || e != nil {
			log.Error("Bad HQC", e)
			return
		}
		log.Infof("Got new HQC block[%v], updating number [%v] -> [%v]",
			qc.QrefBlock().Hash().Hex(), p.hqc.QrefBlock().Height(), qc.QrefBlock().Height())

		p.hqc = qc
		if err := p.persister.PutHQC(qc); err != nil {
			log.Error(e)
			return
		}
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
		if err := p.persister.PutVHeight(p.vheight); err != nil {
			return err
		}

		newBlock := p.blockchain.GetBlockByHash(proposal.NewBlock.Header().Hash())
		vote := CreateVote(newBlock.Header(), p.hqc, p.me)
		vote.Sign(p.me.GetPrivateKey())

		vMsg := vote.GetMessage()
		any, e := ptypes.MarshalAny(vMsg)
		if e != nil {
			log.Error(e)
		}
		p.lastVote = msg.CreateMessage(pb.Message_VOTE, any, p.me)
		p.Vote(ctx)
	}
	return nil
}

}

//Votes with last vote
func (p *Protocol) Vote(ctx context.Context) {
	if p.lastVote != nil {
		go p.srv.SendMessage(ctx, p.pacer.GetNext(), p.lastVote)
	}
	p.pacer.FireEvent(Event{
		T: Voted,
	})
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
		p.pacer.FireEvent(Event{
			T: VotesCollected,
		})
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

//We must propose block atop preferred block.  "It then chooses to extend a branch from the Preferred Block determined by it."
//In later versions of protocol this block is called SafeBlock, block on which we have locked certificate, simply 2-chain block.
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

	//todo remove rand data
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

	p.pacer.FireEvent(Event{
		T: Proposed,
	})
}

func (*Protocol) equivocate(peer *comm.Peer) {
	log.Warningf("Peer [%v] equivocated", peer.GetAddress().Hex())
}

func (p *Protocol) FinishQC(header *bc.Header) {
	var signs []*crypto.Signature
	signsByAddress := make(map[common.Address]*crypto.Signature)

	for k, v := range p.votes {
		signs = append(signs, v.Signature)
		signsByAddress[k] = v.Signature
	}
	bitmap := p.pacer.GetBitmap(signsByAddress)
	aggregate := crypto.AggregateSignatures(bitmap, signs)
	p.Update(bc.CreateQuorumCertificate(aggregate, header))
	log.Debugf("Generated new QC for %v on height %v", header.Hash().Hex(), header.Height())
}

func (p *Protocol) FinishGenesisQC(aggregate *crypto.SignatureAggregate) {
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
		return errors.New("proposed block can'T be empty")
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
