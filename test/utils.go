package test

import (
	"github.com/gagarinchain/common"
	"github.com/gagarinchain/common/api"
	"github.com/gagarinchain/common/eth/crypto"
	cmocks "github.com/gagarinchain/common/mocks"
	"github.com/gagarinchain/network/mocks"
	"github.com/gagarinchain/network/storage"
	"github.com/stretchr/testify/mock"
	"math/big"
	"strconv"
	"testing"
)

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
	p1, _ := crypto.GenerateKey()
	p2, _ := crypto.GenerateKey()
	p3, _ := crypto.GenerateKey()
	p4, _ := crypto.GenerateKey()
	return []*common.Peer{
		common.CreatePeer(p1.PublicKey(), p1, nil),
		common.CreatePeer(p2.PublicKey(), p2, nil),
		common.CreatePeer(p3.PublicKey(), p3, nil),
		common.CreatePeer(p4.PublicKey(), p4, nil),
	}
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
