package hotstuff

import "github.com/gagarinchain/common/api"

//Implementations of slashing should be specific to different crypto's
type Slashing interface {
}

//1. Address of signer is among committee (wrong signature on network slash peer)
//2. Vote only once at height -- business (DoubleVoteEquivocation)
//3. Hash of parent exists (block withholding)   -- when loading
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

type DoubleVoteEquivocation struct {
	firstVote  api.Vote
	secondVote api.Vote
}

type InvalidProposedBlock struct {
	proposal api.Proposal
}
