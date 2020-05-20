package api

import (
	"github.com/gagarinchain/network/common/eth/common"
	"github.com/gagarinchain/network/common/trie/sparse"
	"math/big"
)

type Account interface {
	Balance() *big.Int
	Nonce() uint64
	Copy() Account
	Voters() []common.Address
	Origin() common.Address
	SetOrigin(origin common.Address)
	AddVoters(from common.Address)
	IncrementNonce()
}

//record stores information about version tree structure and provides base account processing logic
type Record interface {
	Pending() Record
	SetPending(pending Record)
	NewPendingRecord(proposer common.Address) Record
	Get(address common.Address) (acc Account, found bool)
	GetForUpdate(address common.Address) (acc Account, found bool)
	Proof(address common.Address) (proof *sparse.Proof)
	RootProof() common.Hash
	SetParent(parent Record)
	Parent() Record
	Serialize() []byte
	Hash() common.Hash
	Trie() *sparse.SMT
	AddSibling(record Record)
	SetHash(pending common.Hash)
	Siblings() []Record
	ApplyTransaction(t Transaction) (err error)
	Update(address common.Address, account Account) error
	Put(address common.Address, account Account)
}
