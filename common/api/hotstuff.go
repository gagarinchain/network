package api

import (
	"github.com/gagarinchain/network/common"
	cmn "github.com/gagarinchain/network/common/eth/common"
	"github.com/gagarinchain/network/common/eth/crypto"
	pb "github.com/gagarinchain/network/common/protobuff"
	"math/big"
)

type ProposerForHeight interface {
	ProposerForHeight(blockHeight int32) *common.Peer
	GetBitmap(src map[cmn.Address]*crypto.Signature) (bitmap *big.Int)
	GetPeers() []*common.Peer
}

type EventNotifier interface {
	SubscribeProtocolEvents(sub chan Event)
	FireEvent(event Event)
}

type Pacer interface {
	EventNotifier
	ProposerForHeight
	GetCurrentView() int32
	GetCurrent() *common.Peer
	GetNext() *common.Peer
}

type Proposal interface {
	Sender() *common.Peer
	NewBlock() Block
	Signature() *crypto.Signature
	HQC() QuorumCertificate
	GetMessage() *pb.ProposalPayload
	Sign(key *crypto.PrivateKey)
}

type Vote interface {
	Sign(key *crypto.PrivateKey)
	GetMessage() *pb.VotePayload
	Sender() *common.Peer
	Header() Header
	Signature() *crypto.Signature
	HQC() QuorumCertificate
}

type EventPayload interface{}
type EventType int

const (
	TimedOut            EventType = iota
	EpochStarted        EventType = iota
	EpochStartTriggered EventType = iota
	Voted               EventType = iota
	VotesCollected      EventType = iota
	Proposed            EventType = iota
	ChangedView         EventType = iota
)

// We intentionally duplicated subset of common.Events here, simply to isolate protocol logic
type Event struct {
	Payload EventPayload
	T       EventType
}

type EventHandler = func(event Event)
