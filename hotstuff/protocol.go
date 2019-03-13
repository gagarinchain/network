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
)

var log = logging.MustGetLogger("hotstuff")

type ProtocolConfig struct {
	F               int
	Blockchain      *bc.Blockchain
	Me              *msg.Peer
	CurrentProposer *msg.Peer
	NextProposer    *msg.Peer
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
	me                *msg.Peer
	currentProposer   *msg.Peer
	nextProposer      *msg.Peer
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
		vote.Sign(p.me.GetPrivateKey())

		vMsg := vote.GetMessage()
		any, e := ptypes.MarshalAny(vMsg)
		if e != nil {
			log.Error(e)
		}
		m := msg.CreateMessage(pb.Message_VOTE, any)

		p.srv.SendMessage(proposal.Sender, m)
	}

	return nil
}
func (p *Protocol) SetCurrentProposer(peer *msg.Peer) {
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
		log.Infof("Already got vote for block [%v] from [%v]", v.NewBlock.Header().Hash().Hex(), id.Hex())

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
		//TODO we can prepare block here and propose it
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

	if secondBestStat.score >= (p.f/3)*2+1 {
		panic(fmt.Sprintf("at list two blocks [%v], [%v]  got consensus score, reload chain than go on",
			bestStat.block, secondBestStat.block))
	}

	if bestStat.score >= (p.f/3)*2+1 {
		return true
	}

	return false
}

//We must propose block atop preferred block.  "It then chooses to extend a branch from the Preferred Block
//determined by it."
func (p *Protocol) OnPropose(data []byte) {
	//TODO    write test to fail on signature
	block := p.blockchain.NewBlock(p.blockchain.GetHead(), p.hqc, data)
	proposal := CreateProposal(block, p.hqc, p.me)

	proposal.Sign(p.me.GetPrivateKey())

	payload := proposal.GetMessage()
	any, e := ptypes.MarshalAny(payload)
	if e != nil {
		log.Error(e)
	}
	m := msg.CreateMessage(pb.Message_PROPOSAL, any)
	p.srv.Broadcast(m)
}

func (*Protocol) equivocate(peer *msg.Peer) {
	log.Warningf("Peer [%v] equivocated", peer.GetAddress().Hex())
}

func (p *Protocol) FinishQC(block *bc.Block) {
	//Simply concatenate votes for now
	var aggregate []byte
	for _, v := range p.votes {
		aggregate = append(aggregate, v.Signature...)
	}
	p.hqc = bc.CreateQuorumCertificate(aggregate, block.Header())

	//we get new QC, it means it is a good time to propose
}

func (p *Protocol) HQC() *bc.QuorumCertificate {
	return p.hqc
}

func (p *Protocol) OnProposerChange(peer *msg.Peer) {
	p.nextProposer = peer
	p.votes = make(map[common.Address]*Vote)
}

func (p *Protocol) Run(msgChan chan *msg.Message, nextProposerChan chan *msg.Peer) {
	for {
		select {
		case m := <-msgChan:
			switch m.Type {
			case pb.Message_VOTE:
				//TODO remove this shitty code
				vp := &pb.VotePayload{}
				if err := ptypes.UnmarshalAny(m.Payload, vp); err != nil {
					log.Error("Couldn't unmarshal response", err)
				}
				block := p.blockchain.GetBlockByHash(common.BytesToHash(vp.BlockHash))

				v, err := CreateVoteFromMessage(m, block, &msg.Peer{})
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
			}
		case nextProposer := <-nextProposerChan:
			p.OnProposerChange(nextProposer)
		}
	}

}
