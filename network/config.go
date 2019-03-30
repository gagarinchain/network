package network

import (
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/poslibp2p/message"
)

// NodeConfig contains basic configuration information that we'll need to
// start our node.
type NodeConfig struct {
	// Params represents the Bitcoin Cash network that this node will be using.
	//Params *chaincfg.Params

	// Port specifies the port use for incoming connections.
	Port uint16

	// PrivateKey is the key to initialize the node with. Typically
	// this will be persisted somewhere and loaded from disk on
	// startup.
	PrivateKey crypto.PrivKey

	// DataDir is the path to a directory to store node data.
	DataDir string

	Committee []*message.Peer
}
