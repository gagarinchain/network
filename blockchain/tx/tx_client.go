package tx

import (
	"context"
	crand "crypto/rand"
	"errors"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/gagarinchain/common"
	"github.com/gagarinchain/common/api"
	cmn "github.com/gagarinchain/common/eth/common"
	"github.com/gagarinchain/common/eth/crypto"
	"github.com/gagarinchain/common/message"
	pb "github.com/gagarinchain/common/protobuff"
	protoio "github.com/gagarinchain/common/protobuff/io"
	"github.com/gagarinchain/common/rpc"
	"github.com/golang/protobuf/ptypes"
	ctxio "github.com/jbenet/go-context/io"
	"github.com/libp2p/go-libp2p"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	net "github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/multiformats/go-multiaddr"
	"math/big"
	"math/rand"
	"sync"
	"time"
)

type TxSend struct {
	streams   map[cmn.Address]net.Stream
	committee []*common.Peer
	peerHost  host.Host
	client    *rpc.CommonClient
}

type TransactOpts struct {
	From           cmn.Address // Ethereum account to send the transaction from
	PrivateKey     *crypto.PrivateKey
	Nonce          uint64         // Nonce to use for the transaction execution (nil = use pending state)
	Peers          []*common.Peer //Peers to receive transaction, if nil then send all
	PeersThreshold int
	Fee            *big.Int        // Fee to pay for transaction
	Context        context.Context // Network context to support cancellation and timeouts (nil = no timeout)
}

//TODO make txSend work with rpc and without it. nonce and committee should be static
func CreateTxSend(rpcPath string) *TxSend {
	client := rpc.InitCommonClient(rpcPath)
	background := context.Background()
	timeout, _ := context.WithTimeout(background, time.Second)

	pbCommittee, err := client.Pbc().GetCommittee(timeout, &pb.GetCommitteeRequest{})
	if err != nil {
		log.Error(err)
		return nil
	}
	var committee []*common.Peer
	for _, pbPeer := range pbCommittee.Peer {
		committee = append(committee, common.CreatePeerFromStorage(pbPeer))
	}
	log.Debugf("Loaded committee of %v peers", len(committee))

	var addrs []*peer.AddrInfo
	for _, p := range committee {
		addrs = append(addrs, p.GetPeerInfo())
	}

	return &TxSend{
		committee: committee,
		client:    client,
	}
}

func (c *TxSend) Start() error {
	priv, _, _ := p2pcrypto.GenerateSecp256k1Key(crand.Reader)
	opts := []libp2p.Option{
		// Listen on all interface on both IPv4 and IPv6.
		libp2p.DisableRelay(),
		libp2p.Identity(priv),
	}

	// This function will initialize a new libp2p Host with our options plus a bunch of default options
	// The default options includes default transports, muxers, security, and peer store.
	c.peerHost, _ = libp2p.New(context.Background(), opts...)

	timeout, _ := context.WithTimeout(context.Background(), time.Second)
	if err := bootstrapConnect(timeout, c.peerHost, local()); err != nil {
		return err
	}

	c.streams = c.createCommitteeStreams()

	return nil
}

//BOOTSTRAP ADDRESSES
func local() (res []*peer.AddrInfo) {
	strs := []string{
		"/ip4/127.0.0.1/tcp/9080/p2p/16Uiu2HAmGgeX9Sr75ofG4rQbUhRiUH2AuKii5QCdD9h8NT83afo4",
		"/ip4/127.0.0.1/tcp/9081/p2p/16Uiu2HAmRfSdSFGboNKPYwcWEPXWtnoanBLMVeY6Ak6A31uC5BVm",
		"/ip4/127.0.0.1/tcp/9082/p2p/16Uiu2HAmTXCmPX1jpwGMXU3oNHrV8Q6RWVgDegSZiuehG2hwdxdC",
	}
	for _, str := range strs {
		newMultiaddr, _ := multiaddr.NewMultiaddr(str)
		addr, _ := peer.AddrInfoFromP2pAddr(newMultiaddr)
		res = append(res, addr)
	}
	return res
}

func (c *TxSend) Transact(txOpts *TransactOpts, txType api.Type, to cmn.Address, value *big.Int, data []byte) error {
	var nonce = txOpts.Nonce
	if txOpts.Nonce == 0 {
		nonce = c.getNonceAndIncrement(txOpts.From)
	}

	//create transaction
	tx := CreateTransaction(txType, to, txOpts.From, nonce, value, txOpts.Fee, data)
	switch txType {
	case api.Payment:
	case api.Settlement:
	case api.Agreement:
		if err := tx.CreateProof(txOpts.PrivateKey); err != nil {
			return err
		}
	case api.Proof:
		return errors.New("can't send proof tx")
		//create proof
	case api.Redeem:
	}

	tx.Hash()
	tx.Sign(txOpts.PrivateKey)
	pbm := tx.GetMessage()
	any, _ := ptypes.MarshalAny(pbm)
	m := message.CreateMessage(pb.Message_TRANSACTION, any, nil)

	var streams []net.Stream
	if txOpts.Peers == nil || len(txOpts.Peers) == 0 {
		for _, stream := range c.streams {
			streams = append(streams, stream)
		}

		if txOpts.PeersThreshold != 0 {
			rand.Shuffle(len(streams), func(i, j int) {
				streams[i], streams[j] = streams[j], streams[i]
			})
			threshold := txOpts.PeersThreshold
			if threshold > len(c.streams) {
				threshold = len(c.streams)
			}
			streams = streams[:txOpts.PeersThreshold]
		}
	} else {
		for _, p := range txOpts.Peers {
			streams = append(streams, c.streams[p.GetAddress()])
		}
	}

	return c.sendMessage(streams, m)
}

func (c *TxSend) createCommitteeStreams() map[cmn.Address]net.Stream {
	streams := make(map[cmn.Address]net.Stream)
	withCancel, _ := context.WithCancel(context.Background())
	for i, prov := range c.committee {
		if err := c.peerHost.Connect(withCancel, *prov.GetPeerInfo()); err != nil {
			log.Error("Can't establish connection", err)
			continue
		}
		stream, err := c.peerHost.NewStream(withCancel, prov.GetPeerInfo().ID, "/gagarin/tx/1.0.0")
		if err != nil {
			log.Error(err)
			continue
		}

		streams[c.committee[i].GetAddress()] = stream
	}
	return streams
}

func bootstrapConnect(ctx context.Context, ph host.Host, peers []*peer.AddrInfo) error {
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
		go func(p *peer.AddrInfo) {
			defer wg.Done()
			defer log.Debug(ctx, "bootstrapDial", ph.ID(), p.ID)
			log.Debugf("%s bootstrapping to %s", ph.ID(), p.ID)

			ph.Peerstore().AddAddrs(p.ID, p.Addrs, peerstore.PermanentAddrTTL)
			if err := ph.Connect(ctx, *p); err != nil {
				log.Debugf("bootstrapDialFailed", p.ID)
				log.Debugf("failed to bootstrap with %v: %s", p.Addrs[0], err)
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

func (c *TxSend) sendMessage(streams []net.Stream, m *message.Message) error {
	for _, s := range streams {
		writer := ctxio.NewWriter(context.Background(), s)
		dw := protoio.NewDelimitedWriter(writer)

		log.Debugf("sending to %v", s.Conn().RemotePeer().Pretty())
		spew.Dump(s.Stat())
		err := dw.WriteMsg(m)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *TxSend) getNonceAndIncrement(from cmn.Address) uint64 {
	nonce := uint64(0)

	account, err := c.client.Pbc().GetAccount(context.Background(), &pb.GetAccountRequest{
		Address: from.Bytes(),
	})
	if err != nil {
		log.Error()
		return 1
	}

	nonce = account.GetAccount().GetNonce()
	nonce += 1
	return nonce
}
