package main

import (
	"crypto/rand"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-protocol"
	"golang.org/x/net/context"
)

type State map[string]int

func (s State) Copy() State {
	return s
}

func main2() {
	//var loader common.CommitteeLoader = &common.CommitteeLoaderImpl{}
	//committee := loader.LoadPeerListFromFile("static/peers.json")
	//peerKey, _ := loader.LoadPeerFromFile("static/peer1.json", committee[1])
	//
	//spew.Dump(peerKey.Type())

	priv, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	opts := []libp2p.Option{
		// Listen on all interface on both IPv4 and IPv6.
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/9081")),
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/9181/ws")),
		libp2p.DisableRelay(),
		libp2p.Identity(priv),
	}

	// This function will initialize a new libp2p Host with our options plus a bunch of default options
	// The default options includes default transports, muxers, security, and peer store.
	peerHost, _ := libp2p.New(context.Background(), opts...)

	id, _ := peer.IDFromPrivateKey(priv)

	spew.Printf("I am %v", peerHost.Addrs())
	spew.Printf("\nId %v", id.Pretty())

	peerHost.SetStreamHandler(protocol.ID("/Libp2pProtocol/2.0.0"), func(stream network.Stream) {
		spew.Dump(stream)
	})

	select {}
}
