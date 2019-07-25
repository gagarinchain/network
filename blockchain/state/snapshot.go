package state

import (
	"github.com/gogo/protobuf/proto"
	"github.com/poslibp2p/common/eth/common"
	"github.com/poslibp2p/common/protobuff"
	"github.com/poslibp2p/common/trie"
	"github.com/poslibp2p/common/tx"
	"math/big"
)

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

	val, found := snap.trie.Get([]byte(address.Hex()))
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
	snap.trie.InsertOrUpdate([]byte(address.Hex()), bytes)
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
