package hotstuff

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	comm "github.com/gagarinchain/common"
	"github.com/gagarinchain/common/api"
	"github.com/gagarinchain/common/eth/common"
	"github.com/gagarinchain/common/eth/crypto"
	msg "github.com/gagarinchain/common/message"
	"github.com/gagarinchain/common/network"
	"github.com/gagarinchain/common/protobuff"
	bc "github.com/gagarinchain/network/blockchain"
	"github.com/gagarinchain/network/storage"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/op/go-logging"
	"time"
)

var (
	log = logging.MustGetLogger("hotstuff")
)

type ProtocolConfig struct {
	F                 int
	Delta             time.Duration
	Blockchain        api.Blockchain
	Me                *comm.Peer
	Srv               network.Service
	Sync              bc.Synchronizer
	Pacer             api.Pacer
	Validators        []api.Validator
	Storage           storage.Storage
	Committee         []*comm.Peer
	InitialState      *InitialState
	OnReceiveProposal api.OnReceiveProposal
	OnVoteReceived    api.OnVoteReceived
	OnProposal        api.OnProposal
	OnBlockCommit     api.OnBlockCommit
}

type InitialState struct {
	View              int32
	VHeight           int32
	LastExecutedBlock api.Header
	HQC               api.QuorumCertificate
	HSC               api.SynchronizeCertificate
}

func DefaultState(bc api.Blockchain) *InitialState {
	return &InitialState{
		View:              int32(0),
		VHeight:           0,
		LastExecutedBlock: bc.GetGenesisBlock().Header(),
		HQC:               nil,
		HSC:               nil,
	}
}

/**
* we do not store b_lock, instead we use HQC.QREF_BLOCK.QC.QREF_BLOCK instead.
* This slightly differs from specification v6, since b_lock is updated only when block.justify updates.
* Nevertheless, HQC.qref_block will always be used as b_leaf and always will be on our next proposal path.
 */
type Protocol struct {
	f                 int
	blockchain        api.Blockchain
	vheight           int32
	lastVote          *msg.Message
	votes             map[common.Address]api.Vote
	lastExecutedBlock api.Header
	hqc               api.QuorumCertificate
	hc                api.Certificate
	me                *comm.Peer
	pacer             api.Pacer
	validators        []api.Validator
	srv               network.Service
	sync              bc.Synchronizer
	persister         *ProtocolPersister
	onReceiveProposal api.OnReceiveProposal
	onVoteReceived    api.OnVoteReceived
	onProposal        api.OnProposal
	onBlockCommit     api.OnBlockCommit
}

func (p *Protocol) HC() api.Certificate {
	return p.hc
}

type ProtocolPersister struct {
	Storage storage.Storage
}

func (pp *ProtocolPersister) PutVHeight(vheight int32) error {
	vhb := storage.Int32ToByte(vheight)
	return pp.Storage.Put(storage.VHeight, nil, vhb)
}
func (pp *ProtocolPersister) GetVHeight() (int32, error) {
	value, err := pp.Storage.Get(storage.VHeight, nil)
	if err != nil {
		return storage.DefaultIntValue, err
	}
	return storage.ByteToInt32(value)
}

func (pp *ProtocolPersister) PutLastExecutedBlockHash(hash common.Hash) error {
	return pp.Storage.Put(storage.LastExecutedBlock, nil, hash.Bytes())
}
func (pp *ProtocolPersister) GetLastExecutedBlockHash() (common.Hash, error) {
	value, err := pp.Storage.Get(storage.LastExecutedBlock, nil)
	if err != nil {
		return common.Hash{}, err
	}
	return common.BytesToHash(value), nil
}

func (pp *ProtocolPersister) PutHQC(hqc api.QuorumCertificate) error {
	m := hqc.GetMessage()
	bytes, e := proto.Marshal(m)
	if e != nil {
		return e
	}
	return pp.Storage.Put(storage.HQC, nil, bytes)
}
func (pp *ProtocolPersister) GetHQC() (api.QuorumCertificate, error) {
	value, err := pp.Storage.Get(storage.HQC, nil)
	if err != nil {
		return nil, err
	}
	pbqc := &pb.QuorumCertificate{}
	if err := proto.Unmarshal(value, pbqc); err != nil {
		return nil, err
	}
	return bc.CreateQuorumCertificateFromMessage(pbqc), nil
}

func (pp *ProtocolPersister) PutHC(hc api.Certificate) error {
	b, e := hc.ToBytes()
	if e != nil {
		return e
	}
	byteType := storage.Int32ToByte(int32(hc.Type()))

	return pp.Storage.Put(storage.HC, nil, append(b, byteType...))
}

func (pp *ProtocolPersister) GetHC() (api.Certificate, error) {
	value, err := pp.Storage.Get(storage.HC, nil)
	if err != nil {
		return nil, err
	}
	t := value[len(value)-binary.MaxVarintLen32:]
	toInt32, err := storage.ByteToInt32(t)
	if err != nil {
		return nil, err
	}
	certType := api.CertType(toInt32)

	var cert api.Certificate
	switch certType {
	case api.Empty:
		fallthrough
	case api.QRef:
		cert = &bc.QuorumCertificateImpl{}
	case api.SC:
		cert = &bc.SynchronizeCertificateImpl{}
	default:
		return nil, errors.New("unknown certificate type")
	}

	return cert, cert.FromBytes(value[:len(value)-binary.MaxVarintLen32])
}

func (p *Protocol) Vheight() int32 {
	return p.vheight
}

func CreateProtocol(cfg *ProtocolConfig) *Protocol {
	p := &Protocol{
		f:                 cfg.F,
		blockchain:        cfg.Blockchain,
		vheight:           cfg.InitialState.VHeight,
		votes:             make(map[common.Address]api.Vote),
		lastExecutedBlock: cfg.InitialState.LastExecutedBlock,
		hqc:               cfg.InitialState.HQC,
		me:                cfg.Me,
		pacer:             cfg.Pacer,
		validators:        cfg.Validators,
		srv:               cfg.Srv,
		sync:              cfg.Sync,
		persister:         &ProtocolPersister{Storage: cfg.Storage},
	}

	if cfg.OnReceiveProposal == nil {
		p.onReceiveProposal = api.NullOnReceiveProposal{}
	} else {
		p.onReceiveProposal = cfg.OnReceiveProposal
	}

	if cfg.OnVoteReceived == nil {
		p.onVoteReceived = api.NullOnVoteReceived{}
	} else {
		p.onVoteReceived = cfg.OnVoteReceived
	}

	if cfg.OnBlockCommit == nil {
		p.onBlockCommit = api.NullOnBlockCommit{}
	} else {
		p.onBlockCommit = cfg.OnBlockCommit
	}
	if cfg.OnProposal == nil {
		p.onProposal = api.NullOnProposal{}
	} else {
		p.onProposal = cfg.OnProposal
	}

	return p
}

//We return qref(qref(HQC_Block))
func (p *Protocol) GetPref() api.Block {
	_, one, _ := p.blockchain.GetThreeChain(p.hqc.QrefBlock().Hash())
	return one
}

func (p *Protocol) CheckCommit() bool {
	log.Info("Check commit for", p.hqc.QrefBlock().Hash().Hex())
	zero, one, two := p.blockchain.GetThreeChain(p.hqc.QrefBlock().Hash())
	if bytes.Equal(two.Header().Parent().Bytes(), one.Header().Hash().Bytes()) &&
		bytes.Equal(one.Header().Parent().Bytes(), zero.Header().Hash().Bytes()) {
		log.Debugf("Committing block %v height %v", zero.Header().Hash().Hex(), zero.Height())
		toCommit, orphans, err := p.blockchain.OnCommit(zero)
		if err != nil {
			log.Error(err)
			return false
		}
		if err := p.onBlockCommit.OnBlockCommit(context.Background(), zero, orphans); err != nil {
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

func (p *Protocol) Update(cert api.Certificate) {
	//if new block has cert to block with greater number means that it is new HQC block
	// and we must consider it as new HEAD
	//if new block has cert to block with greater number means that it is new HQC block
	// and we must consider it as new HEAD
	if p.hc == nil {
		log.Infof("HCs new [%v], old is empty", cert.Height())
	} else {
		log.Infof("HCs new [%v], old[%v]", cert.Height(), p.hc.Height())
	}
	if cert.Type() == api.Empty || cert.Type() == api.QRef {
		hqc := cert.(api.QuorumCertificate)
		if p.hqc == nil {
			log.Infof("Qcs new [%v], old is empty", hqc.Height())
		} else {
			log.Infof("Qcs new [%v], old[%v]", hqc.Height(), p.hqc.Height())
		}

		if p.hqc == nil || hqc.Height() > p.hqc.Height() {
			if p.hqc == nil {
				log.Infof("Got new HQC [%v], updating number empty -> [%v]",
					hqc.GetHash().Hex(), hqc.Height())
			} else {
				log.Infof("Got new HQC [%v], updating number [%v] -> [%v]",
					hqc.GetHash().Hex(), p.hqc.Height(), hqc.Height())
			}

			p.hqc = hqc

			if err := p.persister.PutHQC(hqc); err != nil {
				log.Error(err)
				return
			}
			p.CheckCommit()
		}
	}

	if p.hc == nil || cert.Height() > p.hc.Height() {
		if p.hc == nil {
			log.Infof("Got new HC [%v], updating number empty -> [%v]",
				cert.GetHash().Hex(), cert.Height())
		} else {
			log.Infof("Got new HC [%v], updating number [%v] -> [%v]",
				cert.GetHash().Hex(), p.hc.Height(), cert.Height())
		}

		p.hc = cert

		if err := p.persister.PutHC(cert); err != nil {
			log.Error(err)
			return
		}
		p.pacer.FireEvent(api.Event{
			T:       api.HCUpdated,
			Payload: cert.Height(),
		})
	}

}

func (p *Protocol) OnReceiveProposal(ctx context.Context, proposal api.Proposal) error {
	//probably we should not update HQC from message here, only update qc of the block
	p.Update(proposal.Cert())
	p.Update(proposal.NewBlock().QC())

	log.Infof("current proposer %v", p.pacer.GetCurrent().GetAddress().Hex())

	//TODO move this two validations
	if !proposal.Sender().Equals(p.pacer.GetCurrent()) {
		log.Warningf("This proposer [%v] is not expected", proposal.Sender().GetAddress().Hex())
		p.equivocate(proposal.Sender())
		return errors.New("peer equivocated")
	}
	if proposal.NewBlock().Header().Height() != p.getCurrentView() {
		log.Warningf("This proposer [%v] is expected to propose at height [%v], not on [%v]",
			proposal.Sender().GetAddress().Hex(), p.getCurrentView(), proposal.NewBlock().Header().Height())
		p.equivocate(proposal.Sender())
		return errors.New("peer equivocated")
	}

	if !p.blockchain.IsSibling(proposal.NewBlock().Header(), proposal.NewBlock().QRef()) {
		log.Infof("Block must extend QC.block, but [%v] do not extend [%v]",
			proposal.NewBlock().Header().Hash().Hex(), proposal.NewBlock().QRef().Hash().Hex())
		return errors.New("peer equivocated")
	}

	if proposal.NewBlock().Header().Height() <= p.vheight {
		log.Infof("Received proposal for block [%v] with lower or equal number [%v] we already proposed from proposer [%v], skipping it",
			proposal.NewBlock().Header().Hash().Hex(), proposal.NewBlock().Header().Height(), proposal.Sender().GetAddress().Hex())
	} else if !p.blockchain.IsSibling(proposal.NewBlock().Header(), p.GetPref().Header()) {
		log.Infof("Received proposal for block [%v] with higher number [%v] from proposer [%v], "+
			"but it does not extend Pref block [%v] with number [%v] and it's QC.block.height < PREF.height skipping it",
			proposal.NewBlock().Header().Hash().Hex(), proposal.NewBlock().Header().Height(), proposal.Sender().GetAddress().Hex(),
			p.GetPref().Header().Hash().Hex(), p.GetPref().Header().Height())
	} else {
		log.Infof("Received proposal for block [%v] with higher number [%v] from proposer [%v], voting for it",
			proposal.NewBlock().Header().Hash().Hex(), proposal.NewBlock().Header().Height(), proposal.Sender().GetAddress().Hex())
		if err := p.onReceiveProposal.BeforeProposedBlockAdded(context.Background(), proposal); err != nil {
			return err
		}
		if receipts, err := p.blockchain.AddBlock(proposal.NewBlock()); err != nil {
			return err
		} else {
			if err := p.onReceiveProposal.AfterProposedBlockAdded(context.Background(), proposal, receipts); err != nil {
				return err
			}
		}

		p.vheight = proposal.NewBlock().Header().Height()
		if err := p.persister.PutVHeight(p.vheight); err != nil {
			return err
		}

		newBlock := p.blockchain.GetBlockByHash(proposal.NewBlock().Header().Hash())
		vote := CreateVote(newBlock.Header(), p.hc, p.me)
		vote.Sign(p.me.GetPrivateKey())

		vMsg := vote.GetMessage()
		any, e := ptypes.MarshalAny(vMsg)
		if e != nil {
			log.Error(e)
		}
		p.lastVote = msg.CreateMessage(pb.Message_VOTE, any, p.me)

		if err := p.onReceiveProposal.BeforeVoted(context.Background(), vote); err != nil {
			return err
		}
		p.Vote(ctx)
		if err := p.onReceiveProposal.AfterVoted(context.Background(), vote); err != nil {
			return err
		}
	}
	return nil
}

//Votes with last vote
//This call is fast and will never block. All underlying message sending must be done in async manner with default timeout
//We don't care about sending result since it has no effect on protocol
// ctx - parent context for all execution
func (p *Protocol) Vote(ctx context.Context) {
	if p.lastVote != nil {
		go p.srv.SendMessage(ctx, p.pacer.GetNext(), p.lastVote)
	}
	p.pacer.NotifyEvent(api.Event{
		T: api.Voted,
	})
}

func (p *Protocol) OnReceiveVote(ctx context.Context, vote api.Vote) error {
	log.Debugf("Received vote for block on height [%v] from [%v] peer", vote.Header().Height(), vote.Sender().GetAddress().Hex())
	p.Update(vote.Cert())

	//TODO mb we don't need this check
	//if p.me.GetAddress() != p.pacer.GetCurrent().GetAddress() && p.me.GetAddress() != p.pacer.GetNext().GetAddress() {
	//	return errors.New(fmt.Sprintf("Got unexpected vote from [%v], i'm not proposer now", vote.Sender().GetAddress().Hex()))
	//}

	addr := vote.Sender().GetAddress()

	//todo add vote validation
	if stored, ok := p.votes[addr]; ok {
		//check whether peer voted previously for block with higher number
		if stored.Header().Height() > vote.Header().Height() {
			p.equivocate(vote.Sender())
			return errors.New("peer voted for block with higher height")
		}
		if stored.Header().Height() == vote.Header().Height() &&
			stored.Header().Hash() != vote.Header().Hash() {
			p.equivocate(vote.Sender())
			return errors.New("peer voted for different blocks on the same height")
		}
	}

	p.votes[addr] = vote

	if p.CheckConsensus() {
		if !p.blockchain.Contains(vote.Header().Hash()) {
			e := p.sync.LoadFork(ctx, vote.Header().Height(), vote.Header().Hash(), vote.Sender())
			if e != nil {
				return e
			}
		}
		qc := p.FinishQC(vote.Header())
		p.Update(qc)

		p.pacer.NotifyEvent(api.Event{
			T:       api.VotesCollected,
			Payload: qc.QrefBlock().Height(),
		})
	}
	return nil
}

func (p *Protocol) CheckConsensus() bool {
	type stat struct {
		score  int
		header api.Header
	}
	stats := make(map[common.Hash]*stat, len(p.votes))
	for _, each := range p.votes {
		if s, ok := stats[each.Header().Hash()]; !ok {
			stats[each.Header().Hash()] = &stat{score: 1, header: each.Header()}
		} else {
			s.score += 1
		}
	}

	bestStat := &stat{}
	for _, v := range stats {
		if v.score > bestStat.score {
			bestStat = v
		}
	}

	if bestStat.score >= 2*(p.f/3)+1 && bestStat.header.Height() > p.HQC().QrefBlock().Height() {
		return true
	}

	return false
}

//We must propose block atop preferred block.  "It then chooses to extend a branch from the Preferred Block determined by it."
//In later versions of protocol this block is called SafeBlock, block on which we have locked certificate, simply 2-chain block.
//This call is fast and will never block. All underlying message sending must be done in async manner with default timeout
//We don't care about sending result since it has no effect on protocol
// ctx - parent context for all execution
func (p *Protocol) OnPropose(ctx context.Context) {
	if !p.pacer.GetCurrent().Equals(p.me) {
		log.Debug("Not my turn to propose, skipping")
		return
	}

	log.Debugf("We are proposer, proposing with HQC [%v]", p.HQC().QrefBlock().Height())
	//TODO    write test to fail on signature
	head := p.blockchain.GetBlockByHash(p.HQC().QrefBlock().Hash())
	blocksToAdd := p.getCurrentView() - 1 - head.Header().Height()

	log.Debugf("Padding with %v empty blocks", blocksToAdd)

	for i := 0; i < int(blocksToAdd); i++ {
		var err error
		head, err = p.blockchain.PadEmptyBlock(head, p.hqc)
		if err != nil {
			log.Error("Can't pad with empty block", err)
			return
		}
	}

	block, err := p.blockchain.NewBlock(head, p.hqc, []byte(""))
	if err != nil {
		log.Error("Error while creating new block", err)
		return
	}

	if _, err := p.blockchain.AddBlock(block); err != nil {
		log.Error("Error while adding new block", err)
		return
	}
	proposal := CreateProposal(block, p.me, p.hc)

	if err := p.onProposal.OnProposal(context.Background(), proposal); err != nil {
		log.Error(err)
		return
	}
	proposal.Sign(p.me.GetPrivateKey())
	payload := proposal.GetMessage()
	any, e := ptypes.MarshalAny(payload)
	if e != nil {
		log.Error(e)
		return
	}
	m := msg.CreateMessage(pb.Message_PROPOSAL, any, p.me)

	go p.srv.Broadcast(ctx, m)

	p.pacer.NotifyEvent(api.Event{
		T: api.Proposed,
	})
}

func (p *Protocol) EmptyBlockOnTop() (api.Block, error) {
	block := p.blockchain.GetBlockByHash(p.hqc.QrefBlock().Hash())
	return p.blockchain.PadEmptyBlock(block, p.hqc)
}

func (*Protocol) equivocate(peer *comm.Peer) {
	log.Warningf("Peer [%v] equivocated", peer.GetAddress().Hex())
}

func (p *Protocol) FinishQC(header api.Header) api.QuorumCertificate {
	var signs []*crypto.Signature
	signsByAddress := make(map[common.Address]*crypto.Signature)

	for k, v := range p.votes {
		if bytes.Equal(v.Header().Hash().Bytes(), header.Hash().Bytes()) {
			signs = append(signs, v.Signature())
			signsByAddress[k] = v.Signature()
		}
	}
	bitmap := p.pacer.GetBitmap(signsByAddress)
	aggregate := crypto.AggregateSignatures(bitmap, signs)
	qc := bc.CreateQuorumCertificate(aggregate, header, api.QRef)
	log.Debugf("Generated new QC for %v on height %v", header.Hash().Hex(), header.Height())

	//clean votes
	for k, v := range p.votes {
		if v.Header().Height() <= qc.QrefBlock().Height() {
			delete(p.votes, k)
		}
	}

	return qc
}

func (p *Protocol) FinishGenesisQCAndUpdate(aggregate *crypto.SignatureAggregate) {
	log.Debug("Creating genesis qc")
	p.blockchain.UpdateGenesisBlockQC(bc.CreateQuorumCertificate(aggregate, p.blockchain.GetGenesisBlock().Header(), api.Empty))
	p.Update(p.blockchain.GetGenesisCert())
}
func (p *Protocol) FinishSCAndUpdate(height int32, aggregate *crypto.SignatureAggregate) {
	log.Debug("Creating synchronize certificate")
	cert := bc.CreateSynchronizeCertificate(aggregate, height)
	p.Update(cert)
}
func (p *Protocol) FinishVotingSCAndUpdate(height int32, aggregate *crypto.SignatureAggregate) {
	log.Debug("Creating synchronize voting certificate")

	head := p.blockchain.GetBlockByHash(p.HQC().QrefBlock().Hash())
	blocksToAdd := height - head.Header().Height()

	log.Debugf("Padding with %v empty blocks", blocksToAdd)

	for i := 0; i < int(blocksToAdd); i++ {
		var err error
		head, err = p.blockchain.PadEmptyBlock(head, p.hqc)
		if err != nil {
			log.Error("Can't pad with empty block", err)
			return
		}
	}

	cert := bc.CreateQuorumCertificate(aggregate, head.Header(), api.Empty)

	var committee []*crypto.PublicKey
	for _, peer := range p.pacer.GetPeers() {
		committee = append(committee, peer.PublicKey())
	}
	p.Update(cert)
}

func (p *Protocol) HQC() api.QuorumCertificate {
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

func (p *Protocol) handleMessage(ctx context.Context, m *msg.Message) error {
	switch m.Type {
	case pb.Message_VOTE:
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
		pr, err := CreateProposalFromMessage(m)
		if err != nil {
			return err
		}

		if e := p.validateMessage(pr, pb.Message_PROPOSAL); e != nil {
			return e
		}

		parent := pr.NewBlock().Header().Parent()
		if p.blockchain.Contains(pr.NewBlock().Header().Hash()) {
			log.Info("Received proposal for block that we already has, seems we are creators of this proposal")
			//we use our block that we stored previously, because it has receipts
			stored := p.blockchain.GetBlockByHash(pr.NewBlock().Header().Hash())
			pr.newBlock = stored
		}
		if !p.blockchain.Contains(parent) {
			log.Debugf("Requesting for fork starting at proposal parent block %v at height %v", parent.Hex(), pr.NewBlock().Height())
			err := p.sync.LoadFork(ctx, pr.NewBlock().Header().Height()-1, pr.NewBlock().Header().Parent(), pr.Sender())
			if err != nil {
				return err
			}

			if !p.blockchain.Contains(pr.NewBlock().QRef().Hash()) {
				err := p.sync.LoadFork(ctx, pr.NewBlock().QRef().Height(), pr.NewBlock().QRef().Hash(), pr.Sender())
				if err != nil {
					return err
				}
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
