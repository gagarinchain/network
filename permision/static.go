package permision

import (
	"github.com/poslibp2p/hotstuff"
	"github.com/poslibp2p/message"
)

//Static pacer that store validator set in file and round-robin elect proposer each 2 Delta-periods
type StaticPacer struct {
	committee []*message.Peer
	protocol  *hotstuff.Protocol
}
