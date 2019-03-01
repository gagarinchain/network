package blockchain

import (
	"github.com/poslibp2p/message"
	"github.com/poslibp2p/network"
)

type Vote struct {
	Sender   *network.Peer
	NewBlock *Block
	HQC      *QuorumCertificate
}

func CreateVote(newBlock *Block, hqc *QuorumCertificate, me *network.Peer) *Vote {
	return &Vote{Sender: me, NewBlock: newBlock, HQC: hqc}
}

func (v *Vote) GetMessage() (*message.Message, error) {
	log.Warning("Construct message from vote here", v)
	return &message.Message{}, nil
}
