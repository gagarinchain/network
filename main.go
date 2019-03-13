package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	golog "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/libp2p/go-libp2p-peer"
	"github.com/libp2p/go-libp2p-peerstore"
	"github.com/multiformats/go-multiaddr"
	"github.com/op/go-logging"
	"github.com/poslibp2p/message"
	"github.com/poslibp2p/message/protobuff"
	"github.com/poslibp2p/network"
	gologging "github.com/whyrusleeping/go-logging"
	"os"
	"path"
	"strconv"
)

var stdoutLogFormat = logging.MustStringFormatter(
	`%{color:reset}%{color}%{time:15:04:05.000} [%{shortfunc}] [%{level}] %{message}`,
)

var log = logging.MustGetLogger("main")

func main() {
	// LibP2P code uses golog to log messages. They log with different
	// string IDs (i.e. "swarm"). We can control the verbosity level for
	// all loggers with:
	golog.SetAllLoggers(gologging.INFO) // Change to DEBUG for extra info

	// Parse options from the command line
	listenF := flag.Int("l", 0, "wait for incoming connections")
	target := flag.String("d", "", "target peer to dial")
	flag.Parse()

	if *listenF == 0 {
		log.Fatal("Please provide a port to bind on with -l")
	}

	// First let's create a new identity key pair for our node. If this was your
	// application you would likely save this private key to a database and load
	// it from the db on subsequent start ups.
	privKey, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		log.Fatal(err)
	}

	// Next we'll create the node config
	cfg := network.NodeConfig{
		PrivateKey: privKey,

		Port: uint16(*listenF),

		// For this we will just use a temp directory.
		DataDir: path.Join(os.TempDir(), strconv.Itoa(*listenF)),
	}

	// If the target address is provided let's add it as a bootstrap peer in the config
	if *target != "" {
		// Parse the target address
		peerInfo, err := ParseBootstrapPeer(*target)
		if err != nil {
			log.Fatal(err)
		}
		log.Debugf("Got addresses [%v] for addr[%s]", peerInfo.Addrs, *target)
		cfg.BootstrapPeers = []peerstore.PeerInfo{peerInfo}
	}

	// Now create our node object
	node, err := network.CreateNode(&cfg)
	if err != nil {
		log.Fatal(err)
	}

	node.Dispatcher.Handlers = map[pb.Message_MessageType]message.Handler{}

	//log.Infof("This is my addrs %v", node.Host.Addrs())
	//// If this is the listening dht node then just hang here.
	//if *target == "" {
	//	log.Info("listening for connections")
	//	fullAddr := fmt.Sprintf("/ip4/127.0.0.1/tcp/%d/p2p/%s", *listenF, node.Host.ID().Pretty())
	//	log.Infof("Now run \"go run main.go -l %d -d %s\" on a different terminal\n", *listenF+1, fullAddr)
	//
	//	// Subscribe to the topic
	//	sub, err := node.PubSub.SubscribeAndProvide(context.Background(),"shard")
	//	if err != nil {
	//		log.Fatal(err)
	//	}
	//	for {
	//		msg, err := sub.Next(context.Background())
	//		if err == io.EOF || err == context.Canceled {
	//			break
	//		} else if err != nil {
	//			break
	//		}
	//		pid, err := peer.IDFromBytes(msg.From)
	//		if err != nil {
	//			log.Fatal(err)
	//		}
	//		log.Infof("Received Pubsub message: %s from %s\n", string(msg.Data), pid.Pretty())
	//	}
	//}

	log.Infof("This is my addrs %v", node.Host.Addrs())
	fullAddr := fmt.Sprintf("/ip4/127.0.0.1/tcp/%d/p2p/%s", *listenF, node.Host.ID().Pretty())
	log.Infof("Now run \"./poslibp2p -l %d -d %s\" on a different terminal\n", *listenF+1, fullAddr)

	go node.SubscribeAndListen("shard")

	// Ok now we can bootstrap the node. This could take a little bit if we we're
	// running on a live network.
	err = node.Bootstrap()
	if err != nil {
		log.Fatal(err)
	}

	// Publish to the topic
	//var text = "Hello"
	//any, e := ptypes.MarshalAny(&pb.HelloPayload{HelloMessage: text})
	//if e != nil {
	//	log.Fatal(err)
	//}
	//
	//var msg = &pb.Message{Type: pb.Message_HELLO, Payload: any}
	//marshalled, e := proto.Marshal(msg)
	//if e != nil {
	//	log.Fatal(err)
	//}

	//log.Infof("Publishing message [%s] to topic..", marshalled)
	//
	//tick := func() {
	//	err = node.PubSub.Publish(context.Background(), "shard", marshalled)
	//	if err != nil {
	//		log.Fatal(err)
	//	}
	//}
	//
	//t := time.NewTicker(11 * time.Second)
	//go func() {
	//	for {
	//		select {
	//		case <-t.C:
	//			log.Debugf("Sending...")
	//			tick()
	//		}
	//	}
	//}()
	//tick()

	// hang
	select {}

}

func ParseBootstrapPeer(addr string) (peerstore.PeerInfo, error) {
	p2pAddr, err := multiaddr.NewMultiaddr(addr)
	if err != nil {
		return peerstore.PeerInfo{}, err
	}

	pid, err := p2pAddr.ValueForProtocol(multiaddr.P_P2P)
	if err != nil {
		return peerstore.PeerInfo{}, err
	}

	peerid, err := peer.IDB58Decode(pid)
	if err != nil {
		return peerstore.PeerInfo{}, err
	}

	targetPeerAddr, _ := multiaddr.NewMultiaddr(
		fmt.Sprintf("/p2p/%s", peer.IDB58Encode(peerid)))
	targetAddr := p2pAddr.Decapsulate(targetPeerAddr)

	return peerstore.PeerInfo{
		Addrs: []multiaddr.Multiaddr{
			targetAddr,
		},
		ID: peerid,
	}, nil
}
