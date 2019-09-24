package gagarinchain

import (
	"github.com/gagarinchain/network/common/protobuff"
	"github.com/syndtr/goleveldb/leveldb"
)

//Known validations
//1. Address of signer is among committee
//2. Vote only once at height -- business (onReceiveVote)
//3. Hash of parent exists    -- when loading
//4. Hash of block is real    -- common
//5. Propose at appropriate height -- business (onReceiveProposal)
//6. Propose in order              -- business (onReceiveProposal)
//7. Send vote to next proposer    -- ???
//8. Vote with your keys, not with other signature -- tricky one mb need nonce, but at worst we will receive the same vote again
//9. Voter is among committee                      -- same as 1
//10. Propose same block to all peers              -- equivocation only with proofs
//11. Extend pref block head                       -- equivocation only with proofs
//12. Propose with different QC (withheld QC)      -- same as 10
//13. Block far in the future                      -- if several epochs from the future? (possible when we missed start epoch and now on)
//14. QC block exists and on the fork we get for message
//15. QC signature is valid
type Validator interface {
	IsValid(entity interface{}) (bool, error)
	Supported(mType pb.Message_MessageType) bool
	GetId() interface{}
}

type Persistable interface {
	Persist(storage Storage) error
}

type ResourceType byte

const Block = ResourceType(0x0)
const HeightIndex = ResourceType(0x1)
const CurrentEpoch = ResourceType(0x2)
const CurrentView = ResourceType(0x3)
const TopCommittedHeight = ResourceType(0x4)
const CurrentTopHeight = ResourceType(0x5)
const Snapshot = ResourceType(0x6)
const Record = ResourceType(0x7)

type Storage interface {
	Put(rtype ResourceType, key []byte, value []byte) error
	Get(rtype ResourceType, key []byte) (value []byte, err error)
	Contains(rtype ResourceType, key []byte) bool
	Delete(rtype ResourceType, key []byte) error
	Keys(rtype ResourceType, keyPrefix []byte) (keys [][]byte)
	Stats() *leveldb.DBStats
}
