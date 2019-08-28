package state

import (
	"bytes"
	"github.com/davecgh/go-spew/spew"
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
	trie      *trie.FixedLengthHexKeyMerkleTrie
	pending   *Snapshot
	siblings  []*Snapshot
	hash      common.Hash
	forUpdate map[common.Address]*Account
	proposer  common.Address
}

func (snap *Snapshot) Pending() *Snapshot {
	return snap.pending
}

func (snap *Snapshot) SetPending(pending *Snapshot) {
	snap.pending = pending
}

func NewSnapshot(hash common.Hash, proposer common.Address) *Snapshot {
	return &Snapshot{
		trie:      trie.New(),
		hash:      hash,
		proposer:  proposer,
		forUpdate: make(map[common.Address]*Account),
	}
}

func NewSnapshotWithAccounts(hash common.Hash, proposer common.Address, acc map[common.Address]*Account) *Snapshot {
	s := &Snapshot{
		trie:      trie.New(),
		hash:      hash,
		proposer:  proposer,
		forUpdate: make(map[common.Address]*Account),
	}

	for k, v := range acc {
		s.Put(k, v)
	}
	return s
}

func (snap *Snapshot) NewPendingSnapshot(proposer common.Address) *Snapshot {
	s := &Snapshot{
		trie:      snap.trie.Copy(),
		forUpdate: make(map[common.Address]*Account),
		proposer:  proposer,
	}
	snap.siblings = append(snap.siblings, s)
	snap.SetPending(s)

	return s
}

func (snap *Snapshot) GetForRead(address common.Address) (acc *Account, found bool) {
	val, found := snap.trie.Get([]byte(strings.ToLower(address.Hex())))
	if found {
		account := DeserializeAccount(val)
		if account == nil {
			return nil, false
		}

		return account, true
	}

	return nil, false
}

func DeserializeAccount(serialized []byte) *Account {
	pbAcc := &pb.Account{}
	if err := proto.Unmarshal(serialized, pbAcc); err != nil {
		log.Error(err)
		return nil
	}
	balance := big.NewInt(0)
	var voters []common.Address
	for _, pbv := range pbAcc.Voters {
		voters = append(voters, common.BytesToAddress(pbv))
	}
	return &Account{nonce: pbAcc.Nonce, balance: balance.SetBytes(pbAcc.Value), origin: common.BytesToAddress(pbAcc.Origin), voters: voters}
}

func (snap *Snapshot) GetForUpdate(address common.Address) (acc *Account, found bool) {
	forUpdate, found := snap.forUpdate[address]
	if found { //already taken for update
		return forUpdate, true
	}

	account, found := snap.GetForRead(address)
	if found {
		snap.forUpdate[address] = account
	}

	return account, found
}

func (snap *Snapshot) Update(address common.Address, account *Account) {
	forUpdate, found := snap.forUpdate[address]
	if found { //already taken for update
		if forUpdate != account {
			spew.Dump(forUpdate)
			spew.Dump(account)
			log.Error("Can't update, value is already taken")
			return
		}
		snap.Put(address, account)
		delete(snap.forUpdate, address)
	} else { //trying to update
		log.Warning("Updating value that was not previously acquired for update, inserting it anyway")
		snap.Put(address, account)
	}
}

func (snap *Snapshot) Put(address common.Address, account *Account) {
	var addrBytes [][]byte
	for _, v := range account.voters {
		addrBytes = append(addrBytes, v.Bytes())
	}
	pbAcc := &pb.Account{Nonce: account.nonce, Value: account.balance.Bytes(), Origin: account.origin.Bytes(), Voters: addrBytes}
	b, e := proto.Marshal(pbAcc)
	if e != nil {
		log.Error("can't marshall balance", e)
		return
	}
	snap.trie.InsertOrUpdate([]byte(strings.ToLower(address.Hex())), b)
}

func (snap *Snapshot) IsApplicable(t *tx.Transaction) (err error) {
	sender, found := snap.GetForUpdate(t.From())
	if !found {
		return FutureTransactionError
	}
	if t.Nonce() < sender.nonce+1 {
		return ExpiredTransactionError
	}
	if t.Nonce() > sender.nonce+1 {
		return FutureTransactionError
	}
	cost := t.Fee()
	if sender.balance.Cmp(cost) < 0 {
		return InsufficientFundsError
	}
	return nil
}

func (snap *Snapshot) ApplyTransaction(t *tx.Transaction) (err error) {
	if err := snap.IsApplicable(t); err != nil {
		return err
	}

	sender, found := snap.GetForUpdate(t.From())
	if !found {
		log.Infof("Sender is not found %v", t.From().Hex())
		return FutureTransactionError
	}

	sender.nonce += 1

	proposer, found := snap.GetForUpdate(snap.proposer)
	if !found {
		proposer = NewAccount(0, big.NewInt(0))
	}

	var receiver *Account
	to := t.To()

	switch t.TxType() {
	case tx.Payment:
		cost := big.NewInt(0).Add(t.Value(), t.Fee())
		if sender.balance.Cmp(cost) < 0 {
			return InsufficientFundsError
		}

		receiver, found = snap.GetForUpdate(t.To())
		if !found {
			receiver = NewAccount(0, big.NewInt(0))
		}
		sender.balance.Sub(sender.balance, cost)
		receiver.balance.Add(receiver.balance, t.Value())
		proposer.balance.Add(proposer.balance, t.Fee())
	case tx.Settlement:
		cost := t.Fee()
		if sender.balance.Cmp(cost) < 0 {
			return InsufficientFundsError
		}

		receiver, found = snap.GetForUpdate(t.To())
		if !found {
			receiver = NewAccount(0, big.NewInt(0).Add(t.Value(), big.NewInt(tx.DefaultSettlementReward))) //store reward at account while assets are not separate
		}
		proposer.balance.Add(proposer.balance, big.NewInt(0).Sub(t.Fee(), big.NewInt(tx.DefaultSettlementReward)))
		sender.balance.Sub(sender.balance, cost)
		receiver.origin = t.From()

	case tx.Agreement:
		cost := t.Fee()
		if sender.balance.Cmp(cost) < 0 {
			return InsufficientFundsError
		}

		receiver, found = snap.GetForUpdate(t.To())
		if !found {
			return FutureTransactionError
		}
		sender.balance.Sub(sender.balance, cost)
		proposer.balance.Add(proposer.balance, t.Fee())
		receiver.voters = append(receiver.voters, t.From())
	case tx.Proof:
		cost := big.NewInt(0).Add(t.Value(), t.Fee())
		if sender.balance.Cmp(cost) < 0 {
			return InsufficientFundsError
		}
		proposer.balance.Add(proposer.balance, t.Fee())
		sender.balance.Sub(sender.balance, t.Fee())

		receiver, found = snap.GetForUpdate(t.To())
		if !found {
			log.Infof("Address %v not found", t.From().Hex())
			return FutureTransactionError
		}

		if !bytes.Equal(t.From().Bytes(), receiver.origin.Bytes()) {
			return WrongProofOrigin
		}

		origin, found := snap.GetForUpdate(receiver.origin)
		if !found {
			log.Infof("Origin %v not found", t.From().Hex())
			return FutureTransactionError
		}

		n := len(receiver.voters)
		for _, v := range receiver.voters {
			voter, f := snap.GetForUpdate(v)
			if !f {
				voter = NewAccount(0, big.NewInt(0))

			}
			fraction := big.NewInt(0).Div(big.NewInt(tx.DefaultSettlementReward), big.NewInt(int64(n)))
			voter.balance.Add(voter.balance, fraction)
			receiver.balance.Sub(receiver.balance, fraction)

			snap.Update(v, voter)
		}

		if receiver.balance.Cmp(big.NewInt(0)) > 0 {
			origin.balance.Add(origin.balance, receiver.balance)
			receiver.balance.Set(big.NewInt(0))

		}

		snap.Update(receiver.origin, origin)

	}

	//TODO optimize it with batching several db updates and executing atomic
	snap.Update(to, receiver)
	snap.Update(t.From(), sender)
	snap.Update(snap.proposer, proposer)

	return nil
}

func (snap *Snapshot) Proof() common.Hash {
	return snap.trie.Proof()
}

type Entry struct {
	Key   common.Address
	Value *Account
}

func (snap *Snapshot) Entries() (entries []*Entry) {
	for _, e := range snap.trie.Entries() {
		entry := &Entry{
			Key:   common.HexToAddress(string(e.Key)),
			Value: DeserializeAccount(e.Value),
		}
		entries = append(entries, entry)
	}

	return
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
		hash:      common.BytesToHash(m.Hash),
		trie:      merkleTrie,
		forUpdate: make(map[common.Address]*Account),
	}

	for _, e := range m.GetEntries() {
		merkleTrie.InsertOrUpdate(e.Address, e.Account)
	}

	return snap, nil
}
