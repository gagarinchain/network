package state

import (
	"bytes"
	"errors"
	"fmt"
	cmn "github.com/gagarinchain/common"
	"github.com/gagarinchain/common/api"
	"github.com/gagarinchain/common/eth/common"
	pb "github.com/gagarinchain/common/protobuff"
	"github.com/gagarinchain/common/trie/sparse"
	"github.com/gagarinchain/network/storage"
	"github.com/golang/protobuf/proto"
	"math/big"
)

type RecordPersister struct {
	storage storage.Storage
}

func (p *RecordPersister) Put(r api.Record) error {
	return p.storage.Put(storage.Record, r.Hash().Bytes(), r.Serialize())
}

func (p *RecordPersister) Contains(hash common.Hash) bool {
	return p.storage.Contains(storage.Record, hash.Bytes())
}
func (p *RecordPersister) Get(hash common.Hash) (value []byte, e error) {
	return p.storage.Get(storage.Record, hash.Bytes())
}
func (p *RecordPersister) Delete(hash common.Hash) (e error) {
	return p.storage.Delete(storage.Record, hash.Bytes())
}

func (p *RecordPersister) Hashes() (hashes []common.Hash) {
	keys := p.storage.Keys(storage.Record, nil)
	for _, key := range keys {
		hashes = append(hashes, common.BytesToHash(key))
	}
	return hashes
}

type RecordImpl struct {
	snap      *Snapshot
	pending   api.Record
	parent    api.Record
	siblings  []api.Record
	forUpdate map[common.Address]api.Account
	committee []common.Address
	bus       cmn.EventBus
}

func (r *RecordImpl) Siblings() []api.Record {
	return r.siblings
}

func (r *RecordImpl) AddSibling(sibling api.Record) {
	r.siblings = append(r.siblings, sibling)
}

func (r *RecordImpl) Trie() *sparse.SMT {
	return r.snap.trie
}
func (r *RecordImpl) Hash() common.Hash {
	if r.snap == nil {
		return common.Hash{}
	}
	return r.snap.hash
}
func (r *RecordImpl) SetHash(pending common.Hash) {
	r.snap.hash = pending
}
func (r *RecordImpl) Parent() api.Record {
	return r.parent
}

func (r *RecordImpl) SetParent(parent api.Record) {
	r.parent = parent
}

func NewRecord(snap *Snapshot, parent api.Record, committee []common.Address, bus cmn.EventBus) *RecordImpl {
	return &RecordImpl{snap: snap, parent: parent, committee: committee, forUpdate: make(map[common.Address]api.Account), bus: bus}
}

func (r *RecordImpl) Pending() api.Record {
	return r.pending
}

func (r *RecordImpl) SetPending(pending api.Record) {
	r.pending = pending
}

func (r *RecordImpl) NewPendingRecord(proposer common.Address) api.Record {
	s := &Snapshot{
		trie:     sparse.NewSMT(256),
		proposer: proposer,
	}
	pending := NewRecord(s, r, r.committee, r.bus)
	r.SetPending(pending)

	return pending
}

func (r *RecordImpl) Get(address common.Address) (acc api.Account, found bool) {
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

//ForUpdate is used to protect us from loading instance of account several times from db. This method helps serialize updates.
// We have to load call this method every time we want to change account, otherwise we can erase change during data duplication
func (r *RecordImpl) GetForUpdate(address common.Address) (acc api.Account, found bool) {
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
func (r *RecordImpl) Update(address common.Address, account api.Account) error {
	forUpdate, found := r.forUpdate[address]
	if found { //already taken for update
		if forUpdate != account { //there is another copy of this record
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

	var old *pb.AccountE
	if found {
		old = &pb.AccountE{
			Address: address.Bytes(),
			//hash can be nil, possibly we have no block hash yet6 and it's always pending account snapshot
			Block: r.snap.hash.Bytes(),
			Nonce: forUpdate.Nonce(),
			Value: forUpdate.Balance().Uint64(),
		}
	}
	r.bus.FireEvent(&cmn.Event{
		T: cmn.BalanceUpdated,
		Payload: &pb.AccountUpdatedPayload{
			Old: old,
			New: &pb.AccountE{
				Address: address.Bytes(),
				//hash can be nil, possibly we have no block hash yet6 and it's always pending account snapshot
				Block: r.snap.hash.Bytes(),
				Nonce: account.Nonce(),
				Value: account.Balance().Uint64(),
			},
		},
	})
	return nil
}

//LookUp all needed for proof calculation nodes in all known snapshots and add them to current record state tree
func (r *RecordImpl) lookupAccountAndAddNodes(address common.Address) {
	path := sparse.GetPath(address.Big(), 256)
	relativePath := sparse.GetRelativePath(path) //nodes we need to calculate proof
	ids, hashes := lookUp(r, relativePath)
	for i, id := range ids {
		r.snap.trie.PutNode(id, hashes[i].Bytes())
	}
}

//LookUp recursively all state tree to find toLookup nodes, returns found ids and its values in state smt
func lookUp(record api.Record, toLookup []*sparse.NodeId) (ids []*sparse.NodeId, hashes []common.Hash) {
	var reduced []*sparse.NodeId
	for _, id := range toLookup {
		b, found := record.Trie().GetById(id) //todo probably should hide under snapshot this trie manipulation
		if found {
			ids = append(ids, id)
			hashes = append(hashes, common.BytesToHash(b))
		} else {
			reduced = append(reduced, id)
		}
	}

	if record == record.Parent() {
		log.Error("Self reference in records")
		return
	}

	parent := record.Parent()
	if parent == nil || reduced == nil { //we are done if we iterated all known states
		return
	}
	retIds, retHashes := lookUp(parent, reduced)

	return append(ids, retIds...), append(hashes, retHashes...)
}

func (r *RecordImpl) Proof(address common.Address) (proof *sparse.Proof) {
	p, found := r.snap.Proof(address)
	if !found {
		return nil
	}
	return p
}

func (r *RecordImpl) RootProof() common.Hash {
	return r.snap.RootProof()
}

func (r *RecordImpl) IsApplicable(t api.Transaction) (err error) {
	sender, found := r.GetForUpdate(t.From())
	if !found {
		return FutureTransactionError
	}
	if t.Nonce() < sender.Nonce()+1 {
		return ExpiredTransactionError
	}
	if t.Nonce() > sender.Nonce()+1 {
		return FutureTransactionError
	}
	cost := t.Fee()
	if sender.Balance().Cmp(cost) < 0 {
		return InsufficientFundsError
	}
	return nil
}

func (r *RecordImpl) ApplyTransaction(t api.Transaction) (receipts []api.Receipt, err error) {
	if err := r.IsApplicable(t); err != nil {
		return nil, err
	}

	sender, found := r.GetForUpdate(t.From())
	if !found {
		log.Infof("Sender is not found %v", t.From().Hex())
		return nil, FutureTransactionError
	}
	withFee := true
	if bytes.Equal(r.snap.proposer.Bytes(), t.From().Bytes()) {
		log.Debugf("Current proposer %v, tx from %v", r.snap.proposer.Hex(), t.From().Hex())
		withFee = false
	}

	sender.IncrementNonce()

	proposer, found := r.GetForUpdate(r.snap.proposer)
	if !found {
		proposer = NewAccount(0, big.NewInt(0))
	}

	var receiver api.Account
	to := t.To()

	switch t.TxType() {
	case api.Payment:
		cost := big.NewInt(0).Add(t.Value(), t.Fee())
		if sender.Balance().Cmp(cost) < 0 {
			return nil, InsufficientFundsError
		}

		receiver, found = r.GetForUpdate(t.To())
		if !found {
			receiver = NewAccount(0, big.NewInt(0))
		}
		sender.Balance().Sub(sender.Balance(), cost)
		receiver.Balance().Add(receiver.Balance(), t.Value())
		receipts = append(receipts, NewReceipt(t.Hash(), 0, t.From(), to, t.Value(), receiver.Balance(), sender.Balance()))
		proposer.Balance().Add(proposer.Balance(), t.Fee())

		if withFee {
			receipts = append(receipts, NewReceipt(t.Hash(), 0, t.From(), r.snap.proposer, t.Fee(), proposer.Balance(), sender.Balance()))
		}
	case api.Settlement:
		cost := t.Fee()
		if sender.Balance().Cmp(cost) < 0 {
			return nil, InsufficientFundsError
		}

		receiver, found = r.GetForUpdate(t.To())
		if !found {
			value := big.NewInt(0).Add(t.Value(), big.NewInt(api.DefaultSettlementReward))
			receiver = NewAccount(0, value) //store reward at account while assets are not separate
			receipts = append(receipts, NewReceipt(t.Hash(), 0, t.From(), to, value, receiver.Balance(), sender.Balance()))
		}

		realFee := big.NewInt(0).Sub(t.Fee(), big.NewInt(api.DefaultSettlementReward))
		proposer.Balance().Add(proposer.Balance(), realFee)
		if withFee {
			receipts = append(receipts, NewReceipt(t.Hash(), 0, t.From(), r.snap.proposer, t.Fee(), proposer.Balance(), sender.Balance()))
		}
		sender.Balance().Sub(sender.Balance(), cost)
		receiver.SetOrigin(t.From())

	case api.Agreement:
		cost := t.Fee()
		if sender.Balance().Cmp(cost) < 0 {
			return nil, InsufficientFundsError
		}

		receiver, found = r.GetForUpdate(t.To())
		if !found {
			return nil, FutureTransactionError
		}
		sender.Balance().Sub(sender.Balance(), cost)
		proposer.Balance().Add(proposer.Balance(), t.Fee())
		if withFee {
			receipts = append(receipts, NewReceipt(t.Hash(), 0, t.From(), r.snap.proposer, t.Fee(), proposer.Balance(), sender.Balance()))
		}
	case api.Proof:
		cost := big.NewInt(0).Add(t.Value(), t.Fee())
		if sender.Balance().Cmp(cost) < 0 {
			return nil, InsufficientFundsError
		}
		proposer.Balance().Add(proposer.Balance(), t.Fee())
		if withFee {
			receipts = append(receipts, NewReceipt(t.Hash(), 0, t.From(), r.snap.proposer, t.Fee(), proposer.Balance(), sender.Balance()))
		}

		sender.Balance().Sub(sender.Balance(), t.Fee())

		receiver, found = r.GetForUpdate(t.To())
		if !found {
			log.Infof("Address %v not found", t.From().Hex())
			return nil, FutureTransactionError
		}

		if !bytes.Equal(t.From().Bytes(), receiver.Origin().Bytes()) {
			return nil, WrongProofOrigin
		}

		origin, found := r.GetForUpdate(receiver.Origin())
		if !found {
			log.Infof("Origin %v not found", t.From().Hex())
			return nil, FutureTransactionError
		}

		provers, err := t.RecoverProvers(r.committee)
		if err != nil {
			return nil, err
		}

		n := len(provers)
		for _, v := range provers {
			voter, f := r.GetForUpdate(v)
			if !f {
				voter = NewAccount(0, big.NewInt(0))
			}
			fraction := big.NewInt(0).Div(big.NewInt(api.DefaultSettlementReward), big.NewInt(int64(n)))
			voter.Balance().Add(voter.Balance(), fraction)
			receipts = append(receipts, NewReceipt(t.Hash(), 0, t.To(), v, fraction, receiver.Balance(), voter.Balance()))

			receiver.Balance().Sub(receiver.Balance(), fraction)

			if err := r.Update(v, voter); err != nil {
				return nil, err
			}
		}

		if receiver.Balance().Cmp(big.NewInt(0)) > 0 {
			origin.Balance().Add(origin.Balance(), receiver.Balance())
			receipts = append(receipts, NewReceipt(t.Hash(), 0, t.To(), receiver.Origin(), receiver.Balance(),
				receiver.Balance(), origin.Balance()))
			receiver.Balance().Set(big.NewInt(0))

		}

		if err := r.Update(receiver.Origin(), origin); err != nil {
			return nil, err
		}

	}

	//TODO optimize it with batching several db updates and executing atomic
	if err := r.Update(to, receiver); err != nil {
		return nil, err
	}
	if err := r.Update(t.From(), sender); err != nil {
		return nil, err
	}
	if err := r.Update(r.snap.proposer, proposer); err != nil {
		return nil, err
	}

	return receipts, nil
}

func (r *RecordImpl) Put(address common.Address, account api.Account) {
	_, found := r.Get(address)
	if found {
		log.Errorf("Account for %v is already stored in db, update it instead!", address.Hex())
		return
	}

	r.snap.Put(address, account)
}

func (r *RecordImpl) Serialize() []byte {
	var siblMess [][]byte
	for _, s := range r.siblings {
		siblMess = append(siblMess, s.Hash().Bytes())
	}

	var parent []byte
	if r.parent != nil {
		parent = r.parent.Hash().Bytes()
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
func FromProto(recPb *pb.Record, snapshots map[common.Hash]*Snapshot, committee []common.Address, bus cmn.EventBus) (*RecordImpl, error) {
	hash := common.BytesToHash(recPb.Snap)
	s, f := snapshots[hash]
	if !f {
		return nil, fmt.Errorf("snapshot %v is not found in db", hash.Hex())
	}
	return NewRecord(s, nil, committee, bus), nil
}

func SetParentFromProto(r api.Record, recPb *pb.Record, records map[common.Hash]api.Record) {
	parentHash := common.BytesToHash(recPb.Parent)
	parent, f := records[parentHash]
	if !f {
		log.Errorf("parent record %v is not found in db", parentHash.Hex())
	}
	r.SetParent(parent)

	for _, shash := range recPb.Siblings {
		sibl, f := records[common.BytesToHash(shash)]
		if !f {
			log.Errorf("sibling record %v is not found in db", common.Bytes2Hex(shash))
		}
		r.AddSibling(sibl)
	}
}
