package state

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	gagarinchain "github.com/gagarinchain/network"
	"github.com/gagarinchain/network/common/eth/common"
	pb "github.com/gagarinchain/network/common/protobuff"
	"github.com/gagarinchain/network/common/trie/sparse"
	"github.com/gagarinchain/network/common/tx"
	"github.com/gogo/protobuf/proto"
	"math/big"
)

type RecordPersister struct {
	storage gagarinchain.Storage
}

func (p *RecordPersister) Put(r *Record) error {
	return p.storage.Put(gagarinchain.Record, r.snap.hash.Bytes(), r.Serialize())
}

func (p *RecordPersister) Contains(hash common.Hash) bool {
	return p.storage.Contains(gagarinchain.Record, hash.Bytes())
}
func (p *RecordPersister) Get(hash common.Hash) (value []byte, e error) {
	return p.storage.Get(gagarinchain.Record, hash.Bytes())
}
func (p *RecordPersister) Delete(hash common.Hash) (e error) {
	return p.storage.Delete(gagarinchain.Record, hash.Bytes())
}

func (p *RecordPersister) Hashes() (hashes []common.Hash) {
	keys := p.storage.Keys(gagarinchain.Record, nil)
	for _, key := range keys {
		hashes = append(hashes, common.BytesToHash(key))
	}
	return hashes
}

type Record struct {
	snap     *Snapshot
	pending  *Record
	parent   *Record
	siblings []*Record

	forUpdate map[common.Address]*Account
}

func NewRecord(snap *Snapshot, parent *Record) *Record {
	return &Record{snap: snap, parent: parent, forUpdate: make(map[common.Address]*Account)}
}

func (r *Record) Pending() *Record {
	return r.pending
}

func (r *Record) SetPending(pending *Record) {
	r.pending = pending
}

func (r *Record) NewPendingRecord(proposer common.Address) *Record {
	s := &Snapshot{
		trie:     sparse.NewSMT(256),
		proposer: proposer,
	}
	pending := NewRecord(s, r)
	r.SetPending(pending)

	return pending
}

func (r *Record) Get(address common.Address) (acc *Account, found bool) {
	acc, found = r.snap.Get(address)
	if !found {
		if r.parent != nil {
			return r.parent.Get(address)
		} else {
			return nil, false
		}
	}

	return acc, found
}

func (r *Record) GetForUpdate(address common.Address) (acc *Account, found bool) {
	acc, found = r.forUpdate[address]
	if found {
		return acc, found
	}

	acc, found = r.Get(address)

	if found {
		r.forUpdate[address] = acc
	}
	return
}

// Updates taken value forUpdate.
// Copies from previous snapshots latest versions of needed hashes to calculate proof for current account
func (r *Record) Update(address common.Address, account *Account) error {
	forUpdate, found := r.forUpdate[address]
	if found { //already taken for update
		if forUpdate != account {
			spew.Dump(forUpdate)
			spew.Dump(account)
			return errors.New("can't update, value is already taken")
		}
		r.lookupAccountAndAddNodes(address)
		r.snap.Put(address, account)
		delete(r.forUpdate, address)
	} else { //trying to update
		log.Warningf("Updating account %v that was not previously acquired for update, inserting it anyway", address.Hex())
		r.lookupAccountAndAddNodes(address)
		r.snap.Put(address, account)
	}

	return nil
}

//LookUp all needed for proof calculation nodes in all known snapshots and add them to current record state tree
func (r *Record) lookupAccountAndAddNodes(address common.Address) {
	path := sparse.GetPath(address.Big(), 256)
	relativePath := sparse.GetRelativePath(path) //nodes we need to calculate proof
	ids, hashes := lookUp(r, relativePath)
	for i, id := range ids {
		r.snap.trie.PutNode(id, hashes[i].Bytes())
	}
}

//LookUp recursively all state tree to find toLookup nodes, returns found ids and its values in state smt
func lookUp(record *Record, toLookup []*sparse.NodeId) (ids []*sparse.NodeId, hashes []common.Hash) {
	var reduced []*sparse.NodeId
	for _, id := range toLookup {
		b, found := record.snap.trie.GetById(id) //todo probably should hide under snapshot this trie manipulation
		if found {
			ids = append(ids, id)
			hashes = append(hashes, common.BytesToHash(b))
		} else {
			reduced = append(reduced, id)
		}
	}

	if record == record.parent {
		log.Error("Self reference in records")
		return
	}

	parent := record.parent
	if parent == nil || reduced == nil { //we are done if we iterated all known states
		return
	}
	retIds, retHashes := lookUp(parent, reduced)

	return append(ids, retIds...), append(hashes, retHashes...)
}

func (r *Record) Proof(address common.Address) (proof *sparse.Proof) {
	p, found := r.snap.Proof(address)
	if !found {
		return nil
	}
	return p
}

func (r *Record) RootProof() common.Hash {
	return r.snap.RootProof()
}

func (r *Record) IsApplicable(t *tx.Transaction) (err error) {
	sender, found := r.GetForUpdate(t.From())
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

func (r *Record) ApplyTransaction(t *tx.Transaction) (err error) {
	if err := r.IsApplicable(t); err != nil {
		return err
	}

	sender, found := r.GetForUpdate(t.From())
	if !found {
		log.Infof("Sender is not found %v", t.From().Hex())
		return FutureTransactionError
	}

	sender.nonce += 1

	proposer, found := r.GetForUpdate(r.snap.proposer)
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

		receiver, found = r.GetForUpdate(t.To())
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

		receiver, found = r.GetForUpdate(t.To())
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

		receiver, found = r.GetForUpdate(t.To())
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

		receiver, found = r.GetForUpdate(t.To())
		if !found {
			log.Infof("Address %v not found", t.From().Hex())
			return FutureTransactionError
		}

		if !bytes.Equal(t.From().Bytes(), receiver.origin.Bytes()) {
			return WrongProofOrigin
		}

		origin, found := r.GetForUpdate(receiver.origin)
		if !found {
			log.Infof("Origin %v not found", t.From().Hex())
			return FutureTransactionError
		}

		n := len(receiver.voters)
		for _, v := range receiver.voters {
			voter, f := r.GetForUpdate(v)
			if !f {
				voter = NewAccount(0, big.NewInt(0))

			}
			fraction := big.NewInt(0).Div(big.NewInt(tx.DefaultSettlementReward), big.NewInt(int64(n)))
			voter.balance.Add(voter.balance, fraction)
			receiver.balance.Sub(receiver.balance, fraction)

			r.Update(v, voter)
		}

		if receiver.balance.Cmp(big.NewInt(0)) > 0 {
			origin.balance.Add(origin.balance, receiver.balance)
			receiver.balance.Set(big.NewInt(0))

		}

		r.Update(receiver.origin, origin)

	}

	//TODO optimize it with batching several db updates and executing atomic
	r.Update(to, receiver)
	r.Update(t.From(), sender)
	r.Update(r.snap.proposer, proposer)

	return nil
}

func (r *Record) Put(address common.Address, account *Account) {
	_, found := r.Get(address)
	if found {
		log.Errorf("Account for %v is already stored in db, update it instead!", address.Hex())
		return
	}

	r.snap.Put(address, account)
}

func (r *Record) Serialize() []byte {
	var siblMess [][]byte
	for _, s := range r.siblings {
		siblMess = append(siblMess, s.snap.hash.Bytes())
	}

	var parent []byte
	if r.parent != nil {
		parent = r.parent.snap.hash.Bytes()
	}
	record := &pb.Record{
		Snap:     r.snap.hash.Bytes(),
		Parent:   parent,
		Siblings: siblMess,
	}
	marshal, e := proto.Marshal(record)
	if e != nil {
		log.Error(e)
		return nil
	}

	return marshal
}

func GetProto(bytes []byte) (*pb.Record, error) {
	recPb := &pb.Record{}

	if err := proto.Unmarshal(bytes, recPb); err != nil {
		return nil, err
	}

	return recPb, nil
}

//returns record without parent and siblings
func FromProto(recPb *pb.Record, snapshots map[common.Hash]*Snapshot) (*Record, error) {
	hash := common.BytesToHash(recPb.Snap)
	s, f := snapshots[hash]
	if !f {
		return nil, fmt.Errorf("snapshot %v is not found in db", hash.Hex())
	}
	return NewRecord(s, nil), nil
}

func (r *Record) SetParentFromProto(recPb *pb.Record, records map[common.Hash]*Record) {
	parentHash := common.BytesToHash(recPb.Parent)
	parent, f := records[parentHash]
	if !f {
		log.Errorf("parent record %v is not found in db", parentHash.Hex())
	}
	r.parent = parent

	for _, shash := range recPb.Siblings {
		sibl, f := records[common.BytesToHash(shash)]
		if !f {
			log.Errorf("sibling record %v is not found in db", common.Bytes2Hex(shash))
		}
		r.siblings = append(r.siblings, sibl)
	}
}
