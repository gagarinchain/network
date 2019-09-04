package network

import (
	"context"
	"errors"
	"fmt"
	"github.com/gagarinchain/network/common"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/libp2p/go-libp2p-kad-dht"
	"github.com/op/go-logging"
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
	InitialPeers []peer.AddrInfo
}

//Default parameters for bootstrapping
var DefaultBootstrapConfig = BootstrapConfig{
	MinPeerThreshold:  3,
	Period:            10 * time.Second,
	ConnectionTimeout: (10 * time.Second) / 3,
}

func bootstrapWithPeers(committee []*common.Peer) BootstrapConfig {
	peers := make([]peer.AddrInfo, len(committee))
	for i, c := range committee {
		peers[i] = *c.GetPeerInfo()
	}

	cfg := DefaultBootstrapConfig
	cfg.InitialPeers = peers
	return cfg
}

//Start network services here
//Note that now we simply connect to every node in committee, and actually don't use peer discovery via DHT.
//It is considered that sooner we will have small amount of bootstrapped nodes and bigger amount of committee which can be found using DHT routing tables and connected to
func Bootstrap(ctx context.Context, routing *dht.IpfsDHT, peerHost host.Host, cfg BootstrapConfig) (statusChan chan int, errChan chan error) {
	statusChan = make(chan int)
	errChan = make(chan error)

	watchdog := &Watchdog{
		host: peerHost,
		cfg:  cfg,
		res:  statusChan,
		err:  errChan,
	}
	go watchdog.watch(ctx)
	//Start DHT
	err := routing.BootstrapWithConfig(ctx, dht.DefaultBootstrapConfig)
	if err != nil {
		go func() {
			errChan <- err
		}()
	}

	return statusChan, errChan
}

type Watchdog struct {
	host host.Host
	cfg  BootstrapConfig
	res  chan int
	err  chan error
}

func (w *Watchdog) watch(ctx context.Context) {
	t := time.NewTicker(w.cfg.Period)
	go func() {
		for {
			select {
			case <-t.C:
				if err := bootstrapTick(ctx, w.host, w.cfg); err != nil {
					w.err <- err
				} else {
					w.res <- 0
				}
			case <-ctx.Done():
				t.Stop()
				return
			}
		}
	}()
	if err := bootstrapTick(ctx, w.host, w.cfg); err != nil {
		w.err <- err
	}
}

func bootstrapTick(ctx context.Context, host host.Host, cfg BootstrapConfig) error {
	ctx, cancel := context.WithTimeout(ctx, cfg.ConnectionTimeout)
	defer cancel()
	id := host.ID()
	peers := cfg.InitialPeers

	//manage bootstrap connections
	connected := host.Network().Peers()
	mandatoryToConnect := cfg.MinPeerThreshold - len(connected)
	if mandatoryToConnect <= 0 {
		log.Debugf("Connected to %d (> %d) nodes", len(connected), cfg.MinPeerThreshold)
	}
	numToDial := len(cfg.InitialPeers) - len(connected)

	threshold := len(cfg.InitialPeers) - cfg.MinPeerThreshold

	// filter out connected nodes
	var notConnected []peer.AddrInfo
	for _, p := range peers {
		if host.Network().Connectedness(p.ID) != network.Connected {
			notConnected = append(notConnected, p)
		}
	}

	if len(notConnected)-mandatoryToConnect < 0 {
		log.Debugf("%s no more bootstrap peers to create %d connections", id, numToDial)
		return ErrNotEnoughPeers
	}

	log.Debugf("%s bootstrapping to %d nodes: %v", id, numToDial, notConnected)
	return bootstrapConnect(ctx, host, notConnected, threshold)
}

func bootstrapConnect(ctx context.Context, ph host.Host, peers []peer.AddrInfo, threshold int) error {
	if len(peers) < 1 {
		return nil
	}

	errs := make(chan error, len(peers))
	var wg sync.WaitGroup
	for _, p := range peers {
		wg.Add(1)

		//running startup routines
		go func(p peer.AddrInfo) {
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
	if count > threshold {
		return fmt.Errorf("failed to bootstrap. %s", err)
	}
	return nil
}

func IntMin(x, y int) int {
	if x < y {
		return x
	}
	return y
}
