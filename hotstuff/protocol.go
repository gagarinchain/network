package hotstuff

import (
	"errors"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/golang/protobuf/ptypes"
	"github.com/op/go-logging"
	bc "github.com/poslibp2p/blockchain"
	"github.com/poslibp2p/eth/common"
	"github.com/poslibp2p/message"
	"github.com/poslibp2p/message/protobuff"
	"github.com/poslibp2p/network"
)

var log = logging.MustGetLogger("hotstuff")

type ProtocolConfig struct {
	F               int
	Blockchain      *bc.Blockchain
	Me              *network.Peer
	CurrentProposer *network.Peer
	NextProposer    *network.Peer
	Srv             network.Service
}

type Protocol struct {
	f          int
	blockchain *bc.Blockchain
	// height of last voted block, the height of genesis block is 2
	vheight           int32
	votes             map[common.Address]*Vote
	lastExecutedBlock *bc.Header
	hqc               *bc.QuorumCertificate
	me                *network.Peer
	currentProposer   *network.Peer
	nextProposer      *network.Peer
	srv               network.Service
}

func (p *Protocol) Vheight() int32 {
	return p.vheight
}

func CreateProtocol(cfg *ProtocolConfig) *Protocol {
	return &Protocol{
		f:                 cfg.F,
		blockchain:        cfg.Blockchain,
		vheight:           2,
		votes:             make(map[common.Address]*Vote),
		lastExecutedBlock: cfg.Blockchain.GetGenesisBlock().Header(),
		hqc:               cfg.Blockchain.GetGenesisCert(),
		me:                cfg.Me,
		currentProposer:   cfg.CurrentProposer,
		nextProposer:      cfg.NextProposer,
		srv:               cfg.Srv,
	}
}

//We return qref(qref(HQC_Block))
func (p *Protocol) GetPref() *bc.Block {
	_, one, _ := p.blockchain.GetThreeChainForTwo(p.hqc.QrefBlock().Hash())
	return one
}

func (p *Protocol) CheckCommit() bool {
	log.Info("Check commit for", p.hqc.QrefBlock().Hash().Hex())
	zero, one, two := p.blockchain.GetThreeChainForTwo(p.hqc.QrefBlock().Hash())
	spew.Dump(zero)
	spew.Dump(one)
	spew.Dump(two)
	if two.Header().Parent() == one.Header().Hash() && one.Header().Parent() == zero.Header().Hash() {
		p.onCommit(zero)
		return true
	}

	return false
}

func (p *Protocol) Update(qc *bc.QuorumCertificate) {
	//if new block has cert to block with greater height means that it is new HQC block
	// and we must consider it as new HEAD

	log.Infof("Qcs new [%v], old[%v]", qc.QrefBlock().Height(), p.hqc.QrefBlock().Height())

	if qc.QrefBlock().Height() > p.hqc.QrefBlock().Height() {

		log.Infof("Got new HQC block[%v], updating height [%v] -> [%v]",
			qc.QrefBlock().Hash().Hex(), p.hqc.QrefBlock().Height(), qc.QrefBlock().Height())

		p.hqc = qc
		p.CheckCommit()
	}
}

func (p *Protocol) onCommit(block *bc.Block) {
	if p.lastExecutedBlock.Height() < block.Header().Height() {
		parent := p.blockchain.GetBlockByHash(block.Header().Parent())
		p.onCommit(parent)

		err := p.blockchain.Execute(block)
		//TODO decide whether we should panic when we can't execute some block in the middle of the commit chain
		if err != nil {
			panic(fmt.Sprintf("Fatal error, commiting block %s which can't be executed properly", block.Header().Hash()))
		}
		p.lastExecutedBlock = block.Header()
	}

}

func (p *Protocol) OnReceiveProposal(proposal *Proposal) error {
	if !proposal.Sender.Equals(p.currentProposer) {
		p.equivocate(proposal.Sender)
		return errors.New("peer equivocated")
	}

	p.Update(proposal.HQC)

	if proposal.NewBlock.Header().Height() > p.vheight && p.blockchain.IsSibling(proposal.NewBlock.Header(), p.GetPref().Header()) {
		log.Infof("Received proposal for block [%v] with higher height [%v] from current proposer [%v], voting for it",
			proposal.NewBlock.Header().Hash().Hex(), proposal.NewBlock.Header().Height(), proposal.Sender.GetAddress().Hex())
		p.vheight = proposal.NewBlock.Header().Height()

		newBlock := p.blockchain.GetBlockByHash(proposal.NewBlock.Header().Hash())
		vote := CreateVote(newBlock, p.hqc, p.me)

		vMsg := vote.GetMessage()
		any, e := ptypes.MarshalAny(vMsg)
		if e != nil {
			log.Error(e)
		}
		msg := message.CreateMessage(pb.Message_VOTE, p.me.GetPrivateKey(), any)

		p.srv.SendMessage(proposal.Sender, msg)
	}

	return nil
}
func (p *Protocol) SetCurrentProposer(peer *network.Peer) {
	p.currentProposer = peer
}

func (p *Protocol) OnReceiveVote(vote *Vote) error {
	p.Update(vote.HQC)

	if p.me != p.currentProposer {
		log.Infof("Got stale vote from [%v], i'm not proposer now", vote.Sender.GetAddress())
		return nil
	}

	id := vote.Sender.GetAddress()

	if v, ok := p.votes[id]; ok {
		log.Info("Already got vote [%v] from [%v]", v, id)

		//check whether peer voted again for the same block
		if v.NewBlock.Header().Hash() != vote.NewBlock.Header().Hash() {
			p.equivocate(vote.Sender)
			return errors.New("peer equivocated")
		}
		return nil
	}

	p.votes[id] = vote

	if p.CheckConsensus() {
		p.FinishQC(vote.NewBlock)
	}
	return nil
}

func (p *Protocol) CheckConsensus() bool {
	type stat struct {
		score int
		block *bc.Block
	}
	stats := make(map[common.Hash]*stat, len(p.votes))
	for _, each := range p.votes {
		if s, ok := stats[each.NewBlock.Header().Hash()]; !ok {
			stats[each.NewBlock.Header().Hash()] = &stat{score: 1, block: each.NewBlock}
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

	if secondBestStat.score >= p.f/3+1 {
		panic(fmt.Sprintf("at list two blocks [%v], [%v]  got consensus score, reload chain than go on",
			bestStat.block, secondBestStat.block))
	}

	if bestStat.score >= p.f/3+1 {
		return true
	}

	return false
}

func (p *Protocol) OnPropose(data []byte) {
	block := p.blockchain.NewBlock(p.blockchain.GetHead(), p.hqc, data)
	proposal := &Proposal{Sender: p.me, NewBlock: block, HQC: p.hqc}

	msg, _ := proposal.GetMessage()
	p.srv.Broadcast(msg)
}

func (*Protocol) equivocate(peer *network.Peer) {
	log.Warningf("Peer [%v] equivocated", peer)
}

func (p *Protocol) FinishQC(block *bc.Block) {
	//TODO Probably null votes map
	//TODO Aggregate signatures
	p.hqc = bc.CreateQuorumCertificate([]byte("new QC"), block.Header())
}

func (p *Protocol) HQC() *bc.QuorumCertificate {
	return p.hqc
}
