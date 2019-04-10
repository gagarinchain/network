package network

import (
	"context"
	"errors"
	"fmt"
	"github.com/libp2p/go-libp2p-host"
	"github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-net"
	"github.com/libp2p/go-libp2p-peerstore"
	"github.com/op/go-logging"
	"github.com/poslibp2p/common"
	"math/rand"
	"sync"
	"time"
)

var ErrNotEnoughPeers = errors.New("not enough peers to bootstrap")
var log = logging.MustGetLogger("bootstrap")

type BootstrapConfig struct {
	//Minimum amount of peers needed to start correctly
	MinPeerThreshold int

	// Period duration for bootstrap repetition.
	// Should set up it carefully since bootstrap process could be resource heavy
	Period time.Duration

	// Simply connection timeout
	ConnectionTimeout time.Duration

	//Peers we started with, base peers used to discover additional ones
	InitialPeers []peerstore.PeerInfo
}

//Default parameters for bootstrapping
var DefaultBootstrapConfig = BootstrapConfig{
	MinPeerThreshold:  2,
	Period:            30 * time.Second,
	ConnectionTimeout: (30 * time.Second) / 3,
}

func bootstrapWithPeers(committee []*common.Peer) BootstrapConfig {
	peers := make([]peerstore.PeerInfo, len(committee))
	for i, c := range committee {
		peers[i] = *c.GetPeerInfo()
	}

	cfg := DefaultBootstrapConfig
	cfg.InitialPeers = peers
	return cfg
}

//We Start services here
func Bootstrap(routing *dht.IpfsDHT, peerHost host.Host, cfg BootstrapConfig) error {
	// ticker for bootstrapping
	tick := func() {
		if err := bootstrapTick(context.Background(), peerHost, cfg); err != nil {
			log.Debugf("bootstrap error: %s", err)
		}
	}

	t := time.NewTicker(cfg.Period)
	go func() {
		for {
			select {
			case <-t.C:
				log.Debugf("Ticking...")
				tick()
			}
		}
	}()
	tick()

	//Start DHT
	if _, err := routing.BootstrapWithConfig(dht.DefaultBootstrapConfig); err != nil {
		return err
	}
	return nil
}

func bootstrapTick(ctx context.Context, host host.Host, cfg BootstrapConfig) error {
	ctx, cancel := context.WithTimeout(ctx, cfg.ConnectionTimeout)
	defer cancel()
	id := host.ID()
	peers := cfg.InitialPeers

	//manage bootstrap connections
	connected := host.Network().Peers()
	if len(connected) >= cfg.MinPeerThreshold {
		log.Debugf("%s core bootstrap skipped -- connected to %d (> %d) nodes",
			id, len(connected), cfg.MinPeerThreshold)
		return nil
	}
	numToDial := cfg.MinPeerThreshold - len(connected)

	// filter out connected nodes
	var notConnected []peerstore.PeerInfo
	for _, p := range peers {
		if host.Network().Connectedness(p.ID) != net.Connected {
			notConnected = append(notConnected, p)
		}
	}

	if len(notConnected) < 1 {
		log.Debugf("%s no more bootstrap peers to create %d connections", id, numToDial)
		return ErrNotEnoughPeers
	}

	// connect to a random susbset of bootstrap candidates
	randSubset := randomSubsetOfPeers(notConnected, numToDial)

	log.Debugf("%s bootstrapping to %d nodes: %v", id, numToDial, randSubset)
	return bootstrapConnect(ctx, host, randSubset)
}

func bootstrapConnect(ctx context.Context, ph host.Host, peers []peerstore.PeerInfo) error {
	if len(peers) < 1 {
		return ErrNotEnoughPeers
	}

	errs := make(chan error, len(peers))
	var wg sync.WaitGroup
	for _, p := range peers {
		wg.Add(1)

		//running startup routines
		go func(p peerstore.PeerInfo) {
			defer wg.Done()
			log.Debugf("%s connecting %s", ph.ID(), p.ID)
			log.Debugf("Got addresses [%v] for peer [%s]", p.Addrs, p.ID)

			ph.Peerstore().AddAddrs(p.ID, p.Addrs, peerstore.PermanentAddrTTL)
			if err := ph.Connect(ctx, p); err != nil {
				log.Debugf("failed to connect with %v: %s", p.ID, err)
				errs <- err
				return
			}
			log.Infof("connected %v", p.ID)
		}(p)
	}
	wg.Wait()

	// cleaning error channel
	close(errs)
	count := 0
	var err error
	for err = range errs {
		if err != nil {
			count++
		}
	}
	if count == len(peers) {
		return fmt.Errorf("failed to bootstrap. %s", err)
	}
	return nil
}

func randomSubsetOfPeers(in []peerstore.PeerInfo, max int) []peerstore.PeerInfo {
	n := IntMin(max, len(in))
	var out []peerstore.PeerInfo
	for _, val := range rand.Perm(len(in)) {
		out = append(out, in[val])
		if len(out) >= n {
			break
		}
	}
	return out
}

func IntMin(x, y int) int {
	if x < y {
		return x
	}
	return y
}
