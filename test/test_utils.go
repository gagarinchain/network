package test

import (
	"context"
	"github.com/gagarinchain/common"
	"github.com/gagarinchain/common/api"
	common2 "github.com/gagarinchain/common/eth/common"
	"github.com/gagarinchain/common/eth/crypto"
	msg "github.com/gagarinchain/common/message"
	cmocks "github.com/gagarinchain/common/mocks"
	pb "github.com/gagarinchain/common/protobuff"
	"github.com/gagarinchain/network/blockchain"
	"github.com/gagarinchain/network/blockchain/state"
	"github.com/gagarinchain/network/blockchain/tx"
	"github.com/gagarinchain/network/hotstuff"
	"github.com/gagarinchain/network/mocks"
	"github.com/gagarinchain/network/storage"
	"github.com/golang/protobuf/ptypes"
	"github.com/stretchr/testify/mock"
	"math/big"
	"strconv"
	"testing"
	"time"
)

type BlockFactory struct {
	committee []*common.Peer
	bc        api.Blockchain
	pool      tx.TransactionPool
}

func NewBlockFactory(committee []*common.Peer) *BlockFactory {
	bc, pool := createBlockchain(committee)
	return &BlockFactory{
		committee: committee,
		bc:        bc,
		pool:      pool,
	}
}

func (bf *BlockFactory) CreateProposalForParent(peerNumber int, parent api.Block, qc api.QuorumCertificate, data []byte) (p api.Proposal, m *msg.Message) {
	block, err := bf.bc.NewBlock(parent, qc, data)
	if err != nil {
		log.Error(err)
		return nil, nil
	}

	proposal := hotstuff.CreateProposal(block, bf.committee[peerNumber], qc)
	proposal.Sign(bf.committee[peerNumber].GetPrivateKey())
	any, _ := ptypes.MarshalAny(proposal.GetMessage())

	return proposal, msg.CreateMessage(pb.Message_PROPOSAL, any, bf.committee[peerNumber])
}
func (bf *BlockFactory) CreateProposalForParentWithSC(peerNumber int, parent api.Block, qc api.QuorumCertificate, data []byte) (p api.Proposal, m *msg.Message) {
	block, err := bf.bc.NewBlock(parent, qc, data)
	if err != nil {
		log.Error(err)
		return nil, nil
	}

	sc := bf.CreateSC(parent.Height())
	proposal := hotstuff.CreateProposal(block, bf.committee[peerNumber], sc)
	proposal.Sign(bf.committee[peerNumber].GetPrivateKey())
	any, _ := ptypes.MarshalAny(proposal.GetMessage())

	return proposal, msg.CreateMessage(pb.Message_PROPOSAL, any, bf.committee[peerNumber])
}

func (bf *BlockFactory) CreateProposalForBlock(peerNumber int, block api.Block, qc api.Certificate) (p api.Proposal, m *msg.Message) {
	proposal := hotstuff.CreateProposal(block, bf.committee[peerNumber], qc)
	proposal.Sign(bf.committee[peerNumber].GetPrivateKey())
	any, _ := ptypes.MarshalAny(proposal.GetMessage())

	return proposal, msg.CreateMessage(pb.Message_PROPOSAL, any, bf.committee[peerNumber])
}

func (bf *BlockFactory) CreateBlock(parent api.Block, qc api.QuorumCertificate, data []byte) api.Block {
	block, err := bf.bc.NewBlock(parent, qc, data)

	if err != nil {
		log.Error(err)
		return nil
	}

	return block
}

func (bf *BlockFactory) CreateChainNoGaps(height int32) (blocks []api.Block) {
	parentBlock := bf.bc.GetGenesisBlock()
	parentCert := bf.bc.GetGenesisCert()

	for i := 0; i < int(height); i++ {
		parentCert = bf.CreateQC(parentBlock)
		block, err := bf.bc.NewBlock(parentBlock, parentCert, []byte(""))
		if err != nil {
			log.Error(err)
			return nil
		}
		parentBlock = block
		blocks = append(blocks, parentBlock)
	}

	return blocks
}

func (bf *BlockFactory) waitPoolTransactions(timeout context.Context, count int) chan struct{} {
	fromPool := make(chan struct{})
	ticker := time.NewTicker(50 * time.Millisecond)
	go func() {
		for {
			select {
			case <-ticker.C:
				it := bf.pool.Iterator()
				i := 0
				for it.HasNext() {
					it.Next()
					i++
				}
				if i >= count {
					fromPool <- struct{}{}
				}

			case <-timeout.Done():
				close(fromPool)
				return
			}
		}
	}()
	return fromPool
}

func (bf *BlockFactory) CreateQC(block api.Block) api.QuorumCertificate {
	var signs []*crypto.Signature
	signsByAddress := make(map[common2.Address]*crypto.Signature)
	for i := 0; i < 2*len(bf.committee)/3+1; i++ {
		s := block.Header().Sign(bf.committee[i].GetPrivateKey())
		signs = append(signs, s)
		signsByAddress[bf.committee[i].GetAddress()] = s
	}
	var adresses []common2.Address
	for _, peer := range bf.committee {
		adresses = append(adresses, peer.GetAddress())
	}

	bitmap := common.GetBitmap(signsByAddress, adresses)
	aggregate := crypto.AggregateSignatures(bitmap, signs)

	return blockchain.CreateQuorumCertificate(aggregate, block.Header(), api.QRef)
}
func (bf *BlockFactory) CreateEmptyQC(block api.Block) api.QuorumCertificate {
	var signs []*crypto.Signature
	signsByAddress := make(map[common2.Address]*crypto.Signature)
	for i := 0; i < 2*len(bf.committee)/3+1; i++ {
		hash := blockchain.CalculateSyncHash(block.Height(), true)
		s := crypto.Sign(hash.Bytes(), bf.committee[i].GetPrivateKey())

		signs = append(signs, s)
		signsByAddress[bf.committee[i].GetAddress()] = s
	}
	var adresses []common2.Address
	for _, peer := range bf.committee {
		adresses = append(adresses, peer.GetAddress())
	}

	bitmap := common.GetBitmap(signsByAddress, adresses)
	aggregate := crypto.AggregateSignatures(bitmap, signs)

	return blockchain.CreateQuorumCertificate(aggregate, block.Header(), api.Empty)
}

func (bf *BlockFactory) CreateSC(height int32) api.SynchronizeCertificate {
	var signs []*crypto.Signature
	signsByAddress := make(map[common2.Address]*crypto.Signature)
	for i := 0; i < 2*len(bf.committee)/3+1; i++ {
		hash := blockchain.CalculateSyncHash(height, false)
		s := crypto.Sign(hash.Bytes(), bf.committee[i].GetPrivateKey())

		signs = append(signs, s)
		signsByAddress[bf.committee[i].GetAddress()] = s
	}
	var adresses []common2.Address
	for _, peer := range bf.committee {
		adresses = append(adresses, peer.GetAddress())
	}

	bitmap := common.GetBitmap(signsByAddress, adresses)
	aggregate := crypto.AggregateSignatures(bitmap, signs)

	return blockchain.CreateSynchronizeCertificate(aggregate, height)
}

func (bf *BlockFactory) CreateBlockWithTxs(head api.Block, cert api.QuorumCertificate, bytes []byte, p []api.Transaction) api.Block {

	for _, t := range p {
		bf.pool.Add(t)
	}
	//header := blockchain.NewHeaderBuilderImpl().SetParent(head.Header().Hash()).SetHeight(head.Height() + 1).SetTimestamp(time.Now()).Build()
	//return blockchain.NewBlockBuilderImpl().SetHeader(header).SetQC(cert).SetTxs(p).Build()
	bf.waitPoolTransactions(context.Background(), len(p))

	block := bf.CreateBlock(head, cert, bytes)
	bf.pool.RemoveAll(p...)
	return block

}

func createBlockchain(committee []*common.Peer) (api.Blockchain, tx.TransactionPool) {
	storage := SoftStorageMock()

	bpersister := &blockchain.BlockPersister{storage}
	cpersister := &blockchain.BlockchainPersister{storage}

	pool := tx.NewTransactionPool()
	seed := blockchain.SeedFromFile("../static/seed.json")

	var addresses []common2.Address
	for _, p := range committee {
		addresses = append(addresses, p.GetAddress())
	}
	stateDb := state.NewStateDB(storage, addresses, &common.NullBus{})

	return blockchain.CreateBlockchainFromGenesisBlock(&blockchain.BlockchainConfig{
		Seed:           seed,
		BlockPerister:  bpersister,
		ProposerGetter: &SimpleProposerGetter{committee: committee},
		ChainPersister: cpersister,
		Pool:           pool,
		Db:             stateDb,
		Storage:        storage,
		Delta:          10 * time.Millisecond,
	}), pool
}

type SimpleProposerGetter struct {
	committee []*common.Peer
}

func (s *SimpleProposerGetter) ProposerForHeight(blockHeight int32) *common.Peer {
	return s.committee[int(blockHeight)%len(s.committee)]
}

func (s *SimpleProposerGetter) GetBitmap(src map[common2.Address]*crypto.Signature) (bitmap *big.Int) {
	panic("implement me")
}

func (s *SimpleProposerGetter) GetPeers() []*common.Peer {
	return s.committee
}

func SoftStorageMock() storage.Storage {
	storage := &mocks.Storage{}

	storage.On("Put", mock.AnythingOfType("storage.ResourceType"), mock.AnythingOfType("[]uint8"), mock.AnythingOfType("[]uint8")).Return(nil)
	storage.On("Get", mock.AnythingOfType("storage.ResourceType"), mock.AnythingOfType("[]uint8")).Return(nil, nil)
	storage.On("Contains", mock.AnythingOfType("storage.ResourceType"), mock.AnythingOfType("[]uint8")).Return(false)
	storage.On("Delete", mock.AnythingOfType("storage.ResourceType"), mock.AnythingOfType("[]uint8")).Return(nil)
	storage.On("Keys", mock.AnythingOfType("storage.ResourceType"), mock.AnythingOfType("[]uint8")).Return(nil)

	return storage
}

func MockProposerForHeight() api.ProposerForHeight {
	proposer := &cmocks.ProposerForHeight{}
	proposer.On("ProposerForHeight", mock.AnythingOfType("int32")).Return(&common.Peer{})

	return proposer

}
func MockGoodBlockValidator() api.Validator {
	proposer := &mocks.Validator{}
	proposer.On("IsValid", mock.AnythingOfType("*blockchain.BlockImpl")).Return(true, nil)
	proposer.On("Supported", mock.AnythingOfType("pb.Message_MessageType")).Return(true)
	proposer.On("GetId").Return("Id")

	return proposer

}

func MockGoodHeaderValidator() api.Validator {
	proposer := &mocks.Validator{}
	proposer.On("IsValid", mock.AnythingOfType("*blockchain.HeaderImpl")).Return(true, nil)
	proposer.On("Supported", mock.AnythingOfType("pb.Message_MessageType")).Return(true)
	proposer.On("GetId").Return("Id")

	return proposer

}

func mockCommittee(t *testing.T) []*common.Peer {
	return common.GeneratePeers(4)
}

//func mockCommittee(t *testing.T) []*common.Peer {
//	return []*common.Peer {
//		generateIdentity(t, 0),
//		generateIdentity(t, 1),
//		generateIdentity(t, 2),
//		generateIdentity(t, 3),
//	}
//}

func mockSignatureAggregateValid(m []byte, c []*common.Peer) *crypto.SignatureAggregate {
	bitmap := 1<<(2*len(c)/3+1) - 1
	var signs []*crypto.Signature
	var pubs []*crypto.PublicKey
	for i := 0; i < 2*len(c)/3+1; i++ {
		sign := crypto.Sign(m, c[i].GetPrivateKey())
		signs = append(signs, sign)
		pubs = append(pubs, c[i].PublicKey())
	}

	signatures := crypto.AggregateSignatures(big.NewInt(int64(bitmap)), signs)
	return signatures
}
func mockSignatureAggregateNotValid(m []byte, c []*common.Peer) *crypto.SignatureAggregate {
	bitmap := 1<<len(c) - 1
	var signs []*crypto.Signature
	var pubs []*crypto.PublicKey
	for i := 0; i < 2*len(c)/3+1; i++ {
		sign := crypto.Sign(m, c[i].GetPrivateKey())
		signs = append(signs, sign)
		pubs = append(pubs, c[i].PublicKey())
	}

	signatures := crypto.AggregateSignatures(big.NewInt(int64(bitmap)), signs)
	return signatures
}

func mockSignatureAggregateNotEnough(m []byte, c []*common.Peer) *crypto.SignatureAggregate {
	bitmap := 1<<(2*len(c)/3) - 1
	var signs []*crypto.Signature
	for i := 0; i < 2*len(c)/3; i++ {
		signs = append(signs, crypto.Sign(m, c[i].GetPrivateKey()))
	}
	return crypto.AggregateSignatures(big.NewInt(int64(bitmap)), signs)
}

func generateIdentity(t *testing.T, ind int) *common.Peer {
	loader := &common.CommitteeLoaderImpl{}
	committee := loader.LoadPeerListFromFile("../static/peers.json")
	_, _ = loader.LoadPeerFromFile("../static/peer"+strconv.Itoa(ind)+".json", committee[ind])

	return committee[ind]
}
