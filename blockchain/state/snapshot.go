package state

import (
	"github.com/gagarinchain/common/api"
	"github.com/gagarinchain/common/eth/common"
	"github.com/gagarinchain/common/protobuff"
	"github.com/gagarinchain/common/trie/sparse"
	"github.com/gagarinchain/network/storage"
	"github.com/golang/protobuf/proto"
	"math/big"
)

type SnapshotPersister struct {
	storage storage.Storage
}

func (p *SnapshotPersister) Put(s *Snapshot) error {
	return p.storage.Put(storage.Snapshot, s.hash.Bytes(), s.Serialize())
}

func (p *SnapshotPersister) Contains(hash common.Hash) bool {
	return p.storage.Contains(storage.Snapshot, hash.Bytes())
}
func (p *SnapshotPersister) Get(hash common.Hash) (value []byte, e error) {
	return p.storage.Get(storage.Snapshot, hash.Bytes())
}
func (p *SnapshotPersister) Delete(hash common.Hash) (e error) {
	return p.storage.Delete(storage.Snapshot, hash.Bytes())
}

func (p *SnapshotPersister) Hashes() (hashes []common.Hash) {
	keys := p.storage.Keys(storage.Snapshot, nil)
	for _, key := range keys {
		hashes = append(hashes, common.BytesToHash(key))
	}
	return hashes
}

// Snapshot is wrapper around SMT, which is used for storing blockchain specific state.
// Snapshot knows how to adapt account to be properly stored in SMT, persists state to db and implements other utility functions.
type Snapshot struct {
	trie     *sparse.SMT
	proposer common.Address
	hash     common.Hash
}

func NewSnapshot(hash common.Hash, proposer common.Address) *Snapshot {
	return &Snapshot{
		trie:     sparse.NewSMT(256),
		hash:     hash,
		proposer: proposer,
	}
}

func NewSnapshotWithAccounts(hash common.Hash, proposer common.Address, acc map[common.Address]api.Account) *Snapshot {
	s := &Snapshot{
		trie:     sparse.NewSMT(256),
		hash:     hash,
		proposer: proposer,
	}

	for k, v := range acc {
		s.Put(k, v)
	}
	return s
}

// Puts <address, account> pair to state storage
func (snap *Snapshot) Put(address common.Address, account api.Account) {
	serialized := account.Serialize()
	snap.trie.Add(address.Big(), serialized)
}

//Returns merkle proof for address
func (snap *Snapshot) Proof(address common.Address) (proof *sparse.Proof, found bool) {
	return snap.trie.Proof(address.Big())
}

//Returns root node hash, can be used as state tree proof
func (snap *Snapshot) RootProof() common.Hash {
	bytes, b := snap.trie.GetById(snap.trie.Root())
	if !b {
		log.Error("Root hash is not found, strange situation")
		return common.Hash{}
	}

	return common.BytesToHash(bytes)
}

type Entry struct {
	Key   common.Address
	Value api.Account
}

//returns accounts stored in state storage
func (snap *Snapshot) Entries() (entries []*Entry) {
	for _, e := range snap.trie.Entries() {
		entry := &Entry{
			Key:   common.BigToAddress(e.Key),
			Value: DeserializeAccount(e.Val),
		}
		entries = append(entries, entry)
	}

	return
}

func (snap *Snapshot) Serialize() []byte {
	var entries []*pb.Entry
	for _, e := range snap.trie.Entries() {
		entries = append(entries, &pb.Entry{
			Address: e.Key.Bytes(),
			Account: e.Val,
		})
	}

	pbsnap := &pb.Snapshot{
		Hash:     snap.hash.Bytes(),
		Proposer: snap.proposer.Bytes(),
		Entries:  entries,
	}

	bytes, err := proto.Marshal(pbsnap)
	if err != nil {
		return nil
	}

	return bytes
}

// returns account associated with address
func (snap *Snapshot) Get(address common.Address) (api.Account, bool) {
	val, found := snap.trie.Get(address.Big())
	if !found {
		return nil, false
	}

	return DeserializeAccount(val), true
}

func SnapshotFromProto(bytes []byte) (snap *Snapshot, e error) {
	m := &pb.Snapshot{}
	if err := proto.Unmarshal(bytes, m); err != nil {
		return nil, err
	}

	merkleTrie := sparse.NewSMT(256)
	snap = &Snapshot{
		hash:     common.BytesToHash(m.Hash),
		proposer: common.BytesToAddress(m.Proposer),
		trie:     merkleTrie,
	}

	for _, e := range m.GetEntries() {
		merkleTrie.Add(big.NewInt(0).SetBytes(e.GetAddress()), e.Account)
	}

	return snap, nil
}
