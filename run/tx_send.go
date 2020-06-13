package run

import (
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/gagarinchain/common"
	"github.com/gagarinchain/common/api"
	"github.com/gagarinchain/common/message"
	pb "github.com/gagarinchain/common/protobuff"
	"github.com/gagarinchain/network/blockchain/tx"
	"github.com/gagarinchain/network/network"
	protoio "github.com/gogo/protobuf/io"
	"github.com/golang/protobuf/ptypes"
	ds "github.com/ipfs/go-datastore"
	dsync "github.com/ipfs/go-datastore/sync"
	ctxio "github.com/jbenet/go-context/io"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	net "github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"path"
	"strconv"

	"github.com/multiformats/go-multiaddr"
	"golang.org/x/net/context"
	"math/big"
	"path/filepath"
	"sync"
	"time"
)

type State map[string]int

func (s State) Copy() State {
	return s
}

func TxSend(s *common.Settings) {

	var loader common.CommitteeLoader = &common.CommitteeLoaderImpl{}
	peersPath := filepath.Join(s.Static.Dir, "peers.json")
	committee := loader.LoadPeerListFromFile(peersPath)

	priv, _, _ := crypto.GenerateSecp256k1Key(rand.Reader)
	opts := []libp2p.Option{
		// Listen on all interface on both IPv4 and IPv6.
		libp2p.DisableRelay(),
		libp2p.Identity(priv),
	}

	// This function will initialize a new libp2p Host with our options plus a bunch of default options
	// The default options includes default transports, muxers, security, and peer store.
	peerHost, _ := libp2p.New(context.Background(), opts...)

	id, _ := peer.IDFromPrivateKey(priv)

	spew.Printf("I am %v", peerHost.Addrs())
	spew.Printf("\nId %v", id.Pretty())

	// This function will initialize a new libp2p Host with our options plus a bunch of default options
	// The default options includes default transports, muxers, security, and peer store.
	peerHost, err := libp2p.New(context.Background(), opts...)
	if err != nil {
		log.Error(err)
	}
	log.Infof("I am %v", peerHost.Addrs())

	// connect to the chosen ipfs nodes
	a, err := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/9181/ws/p2p/16Uiu2HAmRfSdSFGboNKPYwcWEPXWtnoanBLMVeY6Ak6A31uC5BVm")
	info, err := peer.AddrInfoFromP2pAddr(a)
	addrInfos := []peer.AddrInfo{*info}
	err = bootstrapConnect(context.Background(), peerHost, addrInfos)
	if err != nil {
		log.Error(err)
	}

	dstore := dsync.MutexWrap(ds.NewMapDatastore())
	//d, _ := dht.New(context.Background(), peerHost, dht.Datastore(dstore), dht.Mode(dht.ModeServer))
	d := dht.NewDHTClient(context.Background(), peerHost, dstore)

	// Bootstrap the host
	err = d.Bootstrap(context.Background())
	if err != nil {
		log.Error(err)
	}

	if err != nil {
		panic(err)
	}

	messages := createTransactions(committee, s, loader)

	timeout, _ := context.WithTimeout(context.Background(), 3*time.Second)
	cid, _ := network.NewTopicCid("/tx").CID()
	provs, err := d.FindProviders(timeout, *cid)
	if err != nil {
		log.Error(err)
	}
	log.Debugf("found %v peers", len(provs))

	var streams []net.Stream
	for _, prov := range provs {
		stream, err := peerHost.NewStream(context.Background(), prov.ID, "/gagarin/tx/1.0.0")
		if err != nil {
			log.Error(err)
		}

		streams = append(streams, stream)
	}

	for _, m := range messages {
		for _, s := range streams {
			writer := ctxio.NewWriter(context.Background(), s)
			dw := protoio.NewDelimitedWriter(writer)

			log.Debugf("sending to %v", s.Conn().RemotePeer().Pretty())
			spew.Dump(s.Stat())
			err = dw.WriteMsg(m)
			if err != nil {
				log.Error(err)
			}

		}

	}
	//stream, err := peerHost.NewStream(context.Background(), committee[1].GetPeerInfo().ID, pubsub.GossipSubID)

	spew.Dump(peerHost.Addrs())
	//stream.Write(t.Serialized())
	select {}
}

func createTransactions(committee []*common.Peer, s *common.Settings, loader common.CommitteeLoader) []*message.Message {
	var messages []*message.Message
	for i := 0; i < 10; i++ {
		t := tx.CreateTransaction(api.Payment, committee[2].GetAddress(), committee[i].GetAddress(), 1, big.NewInt(1),
			big.NewInt(1), []byte("www"))
		peerPath := path.Join(s.Static.Dir, "peer"+strconv.Itoa(i)+".json")
		_, _ = loader.LoadPeerFromFile(peerPath, committee[i])
		t.Sign(committee[i].GetPrivateKey())
		log.Debug(t.Hash().Hex())
		any, _ := ptypes.MarshalAny(t.GetMessage())
		m := message.CreateMessage(pb.Message_TRANSACTION, any, committee[i])

		messages = append(messages, m)
		time.Sleep(time.Second)
	}
	return messages
}

// This code is borrowed from the go-ipfs bootstrap process
func bootstrapConnect(ctx context.Context, ph host.Host, peers []peer.AddrInfo) error {
	if len(peers) < 1 {
		return errors.New("not enough bootstrap peers")
	}

	errs := make(chan error, len(peers))
	var wg sync.WaitGroup
	for _, p := range peers {

		// performed asynchronously because when performed synchronously, if
		// one `Connect` call hangs, subsequent calls are more likely to
		// fail/abort due to an expiring context.
		// Also, performed asynchronously for dial speed.

		wg.Add(1)
		go func(p peer.AddrInfo) {
			defer wg.Done()
			defer log.Debug(ctx, "bootstrapDial", ph.ID(), p.ID)
			log.Debugf("%s bootstrapping to %s", ph.ID(), p.ID)

			ph.Peerstore().AddAddrs(p.ID, p.Addrs, peerstore.PermanentAddrTTL)
			if err := ph.Connect(ctx, p); err != nil {
				log.Debugf("bootstrapDialFailed", p.ID)
				log.Debugf("failed to bootstrap with %v: %s", p.ID, err)
				errs <- err
				return
			}
			log.Debug(ctx, "bootstrapDialSuccess", p.ID)
			log.Debugf("bootstrapped with %v", p.ID)
		}(p)
	}
	wg.Wait()

	// our failure condition is when no connection attempt succeeded.
	// So drain the errs channel, counting the results.
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
