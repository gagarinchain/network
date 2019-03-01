package network

import (
	"context"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-host"
	"github.com/libp2p/go-libp2p-net"
	"github.com/libp2p/go-libp2p-peer"
	"github.com/libp2p/go-libp2p-peerstore"
	"github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p-routing"
	"sync"
	"time"
)

// Wrapper for libp2p Pubsub to add DHT Routing logic for peer discovery.
// See vyzo comment: https://github.com/ipfs/go-ipfs/issues/5569#issuecomment-427556556
type GossipDhtPubSub struct {
	Pubsub  *pubsub.PubSub
	Routing routing.IpfsRouting
	Host    host.Host
}

// Publish publishes data under the given topic
func (p *GossipDhtPubSub) Publish(ctx context.Context, topic string, data []byte) error {
	return p.Pubsub.Publish(topic, data)
}

// Subscribe returns a new Subscription for the given topic.
// While subscribing we calculate topics cid and provide this value to underlying DHT-router
func (p *GossipDhtPubSub) SubscribeAndProvide(ctx context.Context, topic string) (*pubsub.Subscription, error) {
	sub, err := p.Pubsub.Subscribe(topic)
	if err != nil {
		return nil, err
	}

	id, e := (&TopicCid{topic}).CID()
	if e != nil {
		return nil, err
	}

	go p.Routing.Provide(ctx, *id, true)

	// ticker for connecting
	tick := func() {
		p.connectToPubSubPeers(ctx, *id)
	}

	//t := time.NewTicker(30 * time.Second)
	//go func() {
	//	for {
	//		select {
	//		case <-t.C:
	//			log.Debugf("Connecting to peers...")
	//			tick()
	//		}
	//	}
	//}()
	tick()

	return sub, nil
}

// GetTopics returns the topics this node is subscribed to
func (p *GossipDhtPubSub) GetTopics() []string {
	return p.Pubsub.GetTopics()
}

// ListPeers returns a list of peers we are connected to.
func (p *GossipDhtPubSub) ListPeers(topic string) []peer.ID {
	return p.Pubsub.ListPeers(topic)
}

//
func (ps *GossipDhtPubSub) connectToPubSubPeers(ctx context.Context, cid cid.Cid) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	//take 10 providers to connect
	provs := ps.Routing.FindProvidersAsync(ctx, cid, 10)
	var wg sync.WaitGroup

	// filter out connected nodes
	var notConnected []peerstore.PeerInfo
	for p := range provs {
		if ps.Host.Network().Connectedness(p.ID) != net.Connected {
			notConnected = append(notConnected, p)
		}
	}

	for _, prov := range notConnected {
		wg.Add(1)
		go func(pi peerstore.PeerInfo) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(ctx, time.Second*10)
			defer cancel()

			err := ps.Host.Connect(ctx, pi)
			if err != nil {
				log.Error("Pubsub discover: ", err)
				return
			}
			log.Info("connected to Pubsub peer:", pi.ID)
		}(prov)
	}
	wg.Wait()
}
