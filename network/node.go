package network

import (
	"context"
	"fmt"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-ds-leveldb"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/libp2p/go-libp2p-host"
	"github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/opts"
	"github.com/libp2p/go-libp2p-peerstore"
	"github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p-record"
	"github.com/libp2p/go-libp2p-routing"
	"github.com/multiformats/go-multiaddr"
	"github.com/poslibp2p/message"
	"path"
)

type Node struct {
	// Host is the main libp2p instance which handles all our networking.
	// It will require some configuration to set it up. Once set up we
	// can register new protocol handlers with it.
	Host host.Host

	// Routing is a Routing implementation which implements the PeerRouting,
	// ContentRouting, and ValueStore interfaces. In practice this will be
	// a Kademlia DHT.
	Routing routing.IpfsRouting

	// PubSub is an instance of gossipsub which uses the DHT save lists of
	// subscribers to topics which publishers can find via a DHT query and
	// publish messages to the topic using a gossip mechanism.
	PubSub *GossipDhtPubSub

	// PrivateKey is the identity private key for this node
	PrivateKey crypto.PrivKey

	// Datastore is a datastore implementation that we will use to store Routing
	// data.
	Datastore datastore.Datastore

	// Dispatcher for incoming messages which is used to wire messages and appropriate
	// handlers
	Dispatcher *message.Dispatcher

	bootstrapPeers []*message.Peer
}

func CreateNode(config *NodeConfig) (*Node, error) {
	opts := []libp2p.Option{
		// Listen on all interface on both IPv4 and IPv6.
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", config.Port)),
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip6/::/tcp/%d", config.Port)),
		libp2p.Identity(config.PrivateKey),
		libp2p.DisableRelay(),
	}

	// This function will initialize a new libp2p Host with our options plus a bunch of default options
	// The default options includes default transports, muxers, security, and peer store.
	peerHost, err := libp2p.New(context.Background(), opts...)
	if err != nil {
		return nil, err
	}
	log.Infof("I am %v", peerHost.Addrs)

	// Create a leveldb datastore
	dstore, err := leveldb.NewDatastore(path.Join(config.DataDir, "poslibp2p"), nil)
	if err != nil {
		return nil, err
	}

	// Create the DHT instance. It needs the Host and a datastore instance.
	rt, err := dht.New(
		context.Background(), peerHost,
		dhtopts.Datastore(dstore),
		dhtopts.Protocols("/poslibp2p/hotstuff/1.0.0"),
		dhtopts.Validator(record.NamespacedValidator{
			"pk": record.PublicKeyValidator{},
		}),
	)

	ps, err := pubsub.NewGossipSub(context.Background(), peerHost)
	if err != nil {
		return nil, err
	}

	//TODO get PubKey from message
	node := &Node{
		Host:           peerHost,
		Routing:        rt,
		PubSub:         &GossipDhtPubSub{Pubsub: ps, Host: peerHost, Routing: rt},
		PrivateKey:     config.PrivateKey,
		Datastore:      dstore,
		bootstrapPeers: config.Committee,
	}
	return node, nil
}

func (n *Node) GetPeerInfo() *peerstore.PeerInfo {
	join := multiaddr.Join(n.Host.Addrs()...)
	s := join.String() + fmt.Sprintf("/p2p/%v", n.Host.ID().Pretty())
	a, e := multiaddr.NewMultiaddr(s)
	info, e := peerstore.InfoFromP2pAddr(a)
	if e != nil {
		log.Error("Can't get peerInfo", e)
	}
	return info
}

// Will bootstrap the peer host using the provided bootstrap peers. Once the host
// has been bootstrapped it will proceed to bootstrap the DHT.
func (n *Node) Bootstrap() error {
	peers := n.bootstrapPeers
	return Bootstrap(n.Routing.(*dht.IpfsDHT), n.Host, bootstrapWithPeers(peers))
}

// Shutdown will cancel the context shared by the various components which will shut them all down
// disconnecting all peers in the process.
func (n *Node) Shutdown() {
	n.Host.Close()
}
