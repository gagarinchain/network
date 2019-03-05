package hotstuff

import (
	"errors"
	"fmt"
	"github.com/op/go-logging"
	bc "github.com/poslibp2p/blockchain"
	"github.com/poslibp2p/eth/common"
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
	Synchronizer    bc.Synchronizer
}

type Protocol struct {
	f          int
	blockchain *bc.Blockchain
	// height of last voted block, the height of genesis block is 2
	vheight           int32
	votes             map[common.Address]*bc.Vote
	lastExecutedBlock *bc.Header
	hqc               *bc.QuorumCertificate
	me                *network.Peer
	currentProposer   *network.Peer
	nextProposer      *network.Peer
	srv               network.Service
	synchronizer      bc.Synchronizer
}

func CreateProtocol(cfg *ProtocolConfig) *Protocol {
	return &Protocol{
		f:                 cfg.F,
		blockchain:        cfg.Blockchain,
		vheight:           2,
		votes:             make(map[common.Address]*bc.Vote),
		lastExecutedBlock: cfg.Blockchain.GetGenesisBlock().Header(),
		hqc:               cfg.Blockchain.GetGenesisCert(),
		me:                cfg.Me,
		currentProposer:   cfg.CurrentProposer,
		nextProposer:      cfg.NextProposer,
		srv:               cfg.Srv,
		synchronizer:      cfg.Synchronizer,
	}
}

//We return qref(qref(HQC_Block))
func (p *Protocol) GetPref() *bc.Block {
	_, one, _ := p.blockchain.GetThreeChainForTwo(p.hqc.QrefBlock().Hash())
	return one
}

func (p *Protocol) CheckCommit() bool {
	log.Info("Check commit for", common.Bytes2Hex(p.hqc.QrefBlock().Hash().Bytes()))
	zero, one, two := p.blockchain.GetThreeChainForTwo(p.hqc.QrefBlock().Hash())
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
		if !p.blockchain.Contains(qc.QrefBlock().Hash()) {
			p.synchronizer.RequestBlockWithParent(qc.QrefBlock())
		}

		log.Infof("Got new HQC block[%v], updating height [%v] -> [%v]",
			qc.QrefBlock().Hash(), p.hqc.QrefBlock().Height(), qc.QrefBlock().Height())

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

func (p *Protocol) OnReceiveProposal(proposal *bc.Proposal) {
	p.Update(proposal.HQC)
	if proposal.NewBlock.Header().Height() > p.vheight && p.blockchain.IsSibling(proposal.NewBlock.Header(), p.GetPref().Header()) {
		p.vheight = proposal.NewBlock.Header().Height()

		newBlock := p.blockchain.GetBlockByHash(proposal.NewBlock.Header().Hash())
		vote := bc.CreateVote(newBlock, p.hqc, p.me)

		msg, _ := vote.GetMessage()
		p.srv.SendMessage(proposal.Sender, msg)
	}
}

func (p *Protocol) OnReceiveVote(vote *bc.Vote) error {
	if !vote.Sender.Equals(p.currentProposer) {
		p.equivocate(vote.Sender)
		return errors.New("peer equivocated")
	}

	p.Update(vote.HQC)

	id := vote.Sender.GetAddress()
	_, i := p.votes[id]
	if i {
		return nil
	}

	p.votes[id] = vote
	if len(p.votes) >= 2*p.f+1 {
		p.FinishQC(vote.NewBlock)
	}

	return nil
}

func (p *Protocol) OnPropose(data []byte) {
	block := p.blockchain.NewBlock(p.blockchain.GetHead(), p.hqc, data)
	proposal := &bc.Proposal{Sender: p.me, NewBlock: block, HQC: p.hqc}

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
