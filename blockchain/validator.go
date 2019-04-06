package blockchain

import "github.com/poslibp2p/message/protobuff"

type Validator interface {
	Validate(payload *pb.Message) bool
	Supported(mType *pb.Message_MessageType) bool
}

//Known validations
//1. Address of signer is among committee
//2. Vote only once at height
//3. Hash of parent exists
//4. Hash of block is real
//5. Propose at appropriate height
//6. Propose in order
//7. Send vote to next proposer
//8. Vote with your keys, not with other signature
//9. Voter is among committee
//10. Propose same block to all peers
//11. Extend pref block head
//12. Propose with different QC (withheld QC different block 10)
//13. Block far in the future
type EpochStartValidator struct {
}

func (ev *EpochStartValidator) Validate(payload *pb.Message) bool {
	return false
}
