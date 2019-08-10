package state

import (
	"github.com/gagarinchain/network"
	"github.com/gagarinchain/network/common/eth/common"
	"github.com/gagarinchain/network/common/protobuff"
	"github.com/gagarinchain/network/common/trie"
	"github.com/gagarinchain/network/common/tx"
	"github.com/gogo/protobuf/proto"
	"math/big"
	"strings"
)

type SnapshotPersister struct {
	storage gagarinchain.Storage
}

func (p *SnapshotPersister) Put(s *Snapshot) error {
	return p.storage.Put(gagarinchain.Snapshot, s.hash.Bytes(), s.Serialize())
}

func (p *SnapshotPersister) Contains(hash common.Hash) bool {
	return p.storage.Contains(gagarinchain.Snapshot, hash.Bytes())
}
func (p *SnapshotPersister) Get(hash common.Hash) (value []byte, e error) {
	return p.storage.Get(gagarinchain.Snapshot, hash.Bytes())
}
func (p *SnapshotPersister) Delete(hash common.Hash) (e error) {
	return p.storage.Delete(gagarinchain.Snapshot, hash.Bytes())
}

func (p *SnapshotPersister) Hashes() (hashes []common.Hash) {
	keys := p.storage.Keys(gagarinchain.Snapshot, nil)
	for _, key := range keys {
		hashes = append(hashes, common.BytesToHash(key))
	}
	return hashes
}

type Snapshot struct {
	trie     *trie.FixedLengthHexKeyMerkleTrie
	pending  *Snapshot
	siblings []*Snapshot
	hash     common.Hash
}

func (snap *Snapshot) Pending() *Snapshot {
	return snap.pending
}

func (snap *Snapshot) SetPending(pending *Snapshot) {
	snap.pending = pending
}

func NewSnapshot(hash common.Hash) *Snapshot {
	return &Snapshot{
		trie: trie.New(),
		hash: hash,
	}
}

func NewSnapshotWithAccounts(hash common.Hash, acc map[common.Address]*Account) *Snapshot {
	s := &Snapshot{
		trie: trie.New(),
		hash: hash,
	}

	for k, v := range acc {
		s.Put(k, v)
	}
	return s
}

func (snap *Snapshot) NewSnapshot(hash common.Hash) *Snapshot {
	s := NewSnapshot(hash)
	snap.siblings = append(snap.siblings, s)

	return s
}

func (snap *Snapshot) Get(address common.Address) (acc *Account, found bool) {
	val, found := snap.trie.Get([]byte(strings.ToLower(address.Hex())))
	if found {
		pbAcc := &pb.Account{}
		if err := proto.Unmarshal(val, pbAcc); err != nil {
			log.Error(err)
			return nil, false
		}
		balance := big.NewInt(0)
		return &Account{nonce: pbAcc.Nonce, balance: balance.SetBytes(pbAcc.Value)}, true
	}

	return nil, false
}

func (snap *Snapshot) Put(address common.Address, account *Account) {
	pbAcc := &pb.Account{Nonce: account.nonce, Value: account.balance.Bytes()}
	bytes, e := proto.Marshal(pbAcc)
	if e != nil {
		log.Error("can't marshall balance", e)
		return
	}
	snap.trie.InsertOrUpdate([]byte(strings.ToLower(address.Hex())), bytes)
}

func (snap *Snapshot) ApplyTransaction(tx *tx.Transaction) (err error) {
	sender, found := snap.Get(tx.From())
	if !found {
		log.Info(tx.From().Hex())
		return FutureTransactionError
	}

	receiver, found := snap.Get(tx.To())
	if !found {
		receiver = NewAccount(0, big.NewInt(0))
	}

	proposer, found := snap.Get(Me)
	if !found {
		proposer = NewAccount(0, big.NewInt(0))
	}

	cost := big.NewInt(0)
	cost = cost.Add(tx.Value(), tx.Fee())
	if tx.Nonce() < sender.nonce+1 {
		return ExpiredTransactionError
	}
	if tx.Nonce() > sender.nonce+1 {
		return FutureTransactionError
	}

	if sender.balance.Cmp(cost) < 0 {
		return InsufficientFundsError
	}

	sender.balance.Sub(sender.balance, cost)
	sender.nonce += 1
	receiver.balance.Add(receiver.balance, tx.Value())
	proposer.balance.Add(proposer.balance, tx.Fee())

	//TODO optimize it with batching several db updates and executing atomic
	snap.Put(tx.From(), sender)
	snap.Put(tx.To(), receiver)
	snap.Put(Me, proposer)

	return nil
}

func (snap *Snapshot) Proof() common.Hash {
	return snap.trie.Proof()
}

func (snap *Snapshot) Serialize() []byte {
	var hashes [][]byte
	for _, sibl := range snap.siblings {
		hashes = append(hashes, sibl.hash.Bytes())
	}

	var entries []*pb.Entry
	for _, e := range snap.trie.Entries() {
		entries = append(entries, &pb.Entry{
			Address: e.Key,
			Account: e.Value,
		})
	}

	pbsnap := &pb.Snapshot{
		Hash:     snap.hash.Bytes(),
		Siblings: hashes,
		Entries:  entries,
	}

	bytes, err := proto.Marshal(pbsnap)
	if err != nil {
		return nil
	}

	return bytes
}

func FromProtoWithoutSiblings(m *pb.Snapshot) (snap *Snapshot, e error) {
	merkleTrie := trie.New()
	snap = &Snapshot{
		hash: common.BytesToHash(m.Hash),
		trie: merkleTrie,
	}

	for _, e := range m.GetEntries() {
		merkleTrie.InsertOrUpdate(e.Address, e.Account)
	}

	return snap, nil
}
