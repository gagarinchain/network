package network

import (
	"context"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
	"github.com/libp2p/go-libp2p-pubsub"
	"sync"
	"time"
)

// Wrapper for libp2p Pubsub to add DHT Routing logic for peer discovery.
// See vyzo comment: https://github.com/ipfs/go-ipfs/issues/5569#issuecomment-427556556
type GossipDhtPubSub struct {
	Pubsub  *pubsub.PubSub
	Routing routing.Routing
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

	id, e := NewTopicCid(topic).CID()
	if e != nil {
		return nil, e
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
	var notConnected []peer.AddrInfo
	for p := range provs {
		if ps.Host.Network().Connectedness(p.ID) != network.Connected {
			notConnected = append(notConnected, p)
		}
	}

	for _, prov := range notConnected {
		wg.Add(1)
		go func(pi peer.AddrInfo) {
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
