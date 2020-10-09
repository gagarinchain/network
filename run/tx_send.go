package run

import (
	"github.com/gagarinchain/common"
	"github.com/gagarinchain/common/api"
	cmn "github.com/gagarinchain/common/eth/common"
	crypto2 "github.com/gagarinchain/common/eth/crypto"
	pb "github.com/gagarinchain/common/protobuff"
	"github.com/gagarinchain/common/rpc"
	"github.com/gagarinchain/network/blockchain/tx"
	"golang.org/x/net/context"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"math/big"
	"strconv"
	"time"
)

type State map[string]int

func (s State) Copy() State {
	return s
}

//func TxSend(s *common.Settings) {
//
//	var loader common.CommitteeLoader = &common.CommitteeLoaderImpl{}
//	peersPath := filepath.Join(s.Static.Dir, "peers.json")
//	committee := loader.LoadPeerListFromFile(peersPath)
//
//	priv, _, _ := crypto.GenerateSecp256k1Key(crand.Reader)
//	opts := []libp2p.Option{
//		// Listen on all interface on both IPv4 and IPv6.
//		libp2p.DisableRelay(),
//		libp2p.Identity(priv),
//	}
//
//	// This function will initialize a new libp2p Host with our options plus a bunch of default options
//	// The default options includes default transports, muxers, security, and peer store.
//	peerHost, _ := libp2p.New(context.Background(), opts...)
//
//	id, _ := peer.IDFromPrivateKey(priv)
//
//	spew.Printf("I am %v", peerHost.Addrs())
//	spew.Printf("\nId %v", id.Pretty())
//
//	// This function will initialize a new libp2p Host with our options plus a bunch of default options
//	// The default options includes default transports, muxers, security, and peer store.
//	peerHost, err := libp2p.New(context.Background(), opts...)
//	if err != nil {
//		log.Error(err)
//	}
//	log.Infof("I am %v", peerHost.Addrs())
//
//	// connect to the chosen ipfs nodes
//	a, err := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/9181/ws/p2p/16Uiu2HAmRfSdSFGboNKPYwcWEPXWtnoanBLMVeY6Ak6A31uC5BVm")
//	info, err := peer.AddrInfoFromP2pAddr(a)
//	addrInfos := []*peer.AddrInfo{info}
//	err = bootstrapConnect(context.Background(), peerHost, addrInfos)
//	if err != nil {
//		log.Error(err)
//	}
//
//	dstore := dsync.MutexWrap(ds.NewMapDatastore())
//	//d, _ := dht.New(context.Background(), peerHost, dht.Datastore(dstore), dht.Mode(dht.ModeServer))
//	d := dht.NewDHTClient(context.Background(), peerHost, dstore)
//
//	// Bootstrap the host
//	err = d.Bootstrap(context.Background())
//	if err != nil {
//		log.Error(err)
//	}
//
//	if err != nil {
//		panic(err)
//	}
//
//	messages := createTransactions(committee, s, loader)
//	streams := createStreams(d, peerHost)
//	for _, m := range messages {
//		sendMessage(streams, m)
//	}
//	//stream, err := peerHost.NewStream(context.Background(), committee[1].GetPeerInfo().ID, pubsub.GossipSubID)
//
//	spew.Dump(peerHost.Addrs())
//	//stream.Write(t.Serialized())
//	select {}
//}

//
//func createStreams(d *dht.IpfsDHT, peerHost host.Host) []net.Stream {
//	timeout, _ := context.WithTimeout(context.Background(), 3*time.Second)
//	cid, _ := network.NewTopicCid("/tx").CID()
//	provs, err := d.FindProviders(timeout, *cid)
//	if err != nil {
//		log.Error(err)
//	}
//	log.Debugf("found %v peers", len(provs))
//
//	var streams []net.Stream
//	for _, prov := range provs {
//		stream, err := peerHost.NewStream(context.Background(), prov.ID, "/gagarin/tx/1.0.0")
//		if err != nil {
//			log.Error(err)
//			continue;
//		}
//
//		streams = append(streams, stream)
//	}
//	return streams
//}

//func createTransactions(committee []*common.Peer, s *common.Settings, loader common.CommitteeLoader) []*message.Message {
//	var messages []*message.Message
//	for i := 0; i < 10; i++ {
//		t := tx.CreateTransaction(api.Payment, committee[2].GetAddress(), committee[i].GetAddress(), 1, big.NewInt(1),
//			big.NewInt(1), []byte("www"))
//		peerPath := path.Join(s.Static.Dir, "peer"+strconv.Itoa(i)+".json")
//		_, _ = loader.LoadPeerFromFile(peerPath, committee[i])
//		t.Sign(committee[i].GetPrivateKey())
//		log.Debug(t.Hash().Hex())
//		any, _ := ptypes.MarshalAny(t.GetMessage())
//		m := message.CreateMessage(pb.Message_TRANSACTION, any, committee[i])
//
//		messages = append(messages, m)
//		time.Sleep(time.Second)
//	}
//	return messages
//}

type Execution struct {
	client   *rpc.CommonClient
	scenario Scenario
	senders  []*crypto2.PrivateKey
	viewChan chan int32
	nonces   map[cmn.Address]uint64
	txSend   *tx.TxSend
}

type Settings struct {
	ScenarioPath string
	RpcPath      string
	SendersPath  string
}

func CreateExecution(s *Settings) *Execution {
	background := context.Background()
	client := rpc.InitCommonClient(s.RpcPath)

	withCancel, _ := context.WithCancel(background)

	view := client.PollView(withCancel)

	scenario := getScenarioFromFile(s.ScenarioPath)
	txSend := tx.CreateTxSend(s.RpcPath)
	if err := txSend.Start(); err != nil {
		log.Error(err)
		return nil
	}

	e := &Execution{
		client:   client,
		scenario: scenario,
		senders:  getSendersFromFile(s.SendersPath),
		viewChan: view,
		nonces:   make(map[cmn.Address]uint64),
		txSend:   txSend,
	}
	return e
}

type Scenario map[int]Peers

type Peers map[string][]struct {
	Type  string `yaml:"type,omitempty"`
	Fee   int64  `yaml:"fee"`
	From  int    `yaml:"from"`
	To    string `yaml:"to"`
	Value int64  `yaml:"value"`
	Nonce uint64 `yaml:"nonce,omitempty"`
}

func getScenarioFromFile(path string) Scenario {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		log.Error(err)
		return nil
	}

	s := Scenario{}
	//out := make(map[string]string)
	err = yaml.Unmarshal(file, s)

	if err != nil {
		log.Error(err)
		return nil
	}
	return s
}

func getSendersFromFile(path string) []*crypto2.PrivateKey {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		log.Error(err)
		return nil
	}

	out := make([]string, 256)
	err = yaml.Unmarshal(file, &out)

	if err != nil {
		log.Error(err)
		return nil
	}
	var res []*crypto2.PrivateKey
	for _, key := range out {
		b := cmn.Hex2Bytes(key)
		res = append(res, crypto2.PkFromBytes(b))
	}

	return res
}

func (e *Execution) Execute() {
	for {
		select {
		case nextView := <-e.viewChan:
			log.Debugf("New view %v started", nextView)
			stage, f := e.scenario[int(nextView)]
			if !f {
				continue
			}

			log.Debugf("Processing %v stages", len(stage))
			for p, mess := range stage {
				var curPeers []*common.Peer
				peerThreshold := 0

				switch p {
				case "a": //any
				case "p":
					timeout, _ := context.WithTimeout(context.Background(), time.Second)
					proposer, err := e.client.Pbc().GetProposerForView(timeout, &pb.GetProposerForViewRequest{
						View: nextView,
					})
					if err != nil {
						log.Error("can't get proposer for next view", err)
						continue
					}
					p := common.CreatePeerFromStorage(proposer.Peer)
					curPeers = []*common.Peer{p}
				default:
					curPeers = nil
					peerThreshold, err := strconv.Atoi(p)
					if err != nil {
						log.Errorf("Unknown peer literal %v", peerThreshold)
						continue
					}

				}
				for _, dto := range mess {
					ttype := api.Payment
					if dto.Type != "" {
						switch dto.Type {
						case "Payment":
							ttype = api.Payment
						case "Agreement":
							ttype = api.Agreement
						case "Redeem":
							ttype = api.Redeem
						}
					}

					to := e.getAddressTo(dto.To)

					from, _ := e.getAddressAndPeerFrom(dto.From)
					nonce := dto.Nonce
					if dto.Nonce == 0 {
						nonce = e.getNonceAndIncrement(from)
					}

					opts := &tx.TransactOpts{
						From:           from,
						PrivateKey:     e.senders[dto.From],
						Nonce:          nonce,
						Peers:          curPeers,
						PeersThreshold: peerThreshold,
						Fee:            big.NewInt(1),
						Context:        context.Background(),
					}

					if err := e.txSend.Transact(opts, ttype, to, big.NewInt(dto.Value)); err != nil {
						log.Errorf("Can't send transaction %v", err)
						continue
					}

				}
			}
		}
	}
}

//func (e *Execution) createMessages(s Scenario) {
//	res := make(map[int]map[string][]*message.Message)
//
//	for h, p := range s {
//		groupedMessages := make(map[string][]*message.Message)
//		for peer, list := range p {
//			var messages []*message.Message
//			for _, dto := range list {
//				ttype := api.Payment
//				if dto.Type != "" {
//					switch dto.Type {
//					case "Payment":
//						ttype = api.Payment
//					case "Agreement":
//						ttype = api.Agreement
//					case "Redeem":
//						ttype = api.Redeem
//					}
//				}
//
//				to := e.getAddressTo(dto.To)
//
//				from, p := e.getAddressAndPeerFrom(dto.From)
//				nonce := dto.Nonce
//				if dto.Nonce == 0 {
//					nonce = e.getNonceAndIncrement(from)
//				}
//
//				t := tx.CreateTransaction(ttype, to, from, nonce, big.NewInt(dto.Value),
//					big.NewInt(dto.Fee), nil)
//				t.Sign(e.senders[dto.From])
//				log.Debug(t.Hash().Hex())
//				getMessage := t.GetMessage()
//				any, _ := ptypes.MarshalAny(getMessage)
//				m := message.CreateMessage(pb.Message_TRANSACTION, any, p)
//
//				messages = append(messages, m)
//			}
//			groupedMessages[peer] = messages
//		}
//		res[h] = groupedMessages
//	}
//
//	e.messages = res
//}

func (e *Execution) getAddressTo(a string) cmn.Address {
	atoi, err := strconv.Atoi(a)
	to := cmn.Address{}
	if err == nil {
		pk := e.senders[atoi]
		if pk == nil {
			log.Error("Can't find peer %v in te list", atoi)
			return cmn.Address{}
		}
		to = crypto2.PubkeyToAddress(pk.PublicKey())

	} else {
		to = cmn.HexToAddress(a)
	}
	return to
}
func (e *Execution) getAddressAndPeerFrom(a int) (cmn.Address, *common.Peer) {
	to := cmn.Address{}

	pk := e.senders[a]
	if pk == nil {
		log.Error("Can't find peer %v in te list", a)
		return cmn.Address{}, nil
	}
	to = crypto2.PubkeyToAddress(pk.PublicKey())

	return to, common.CreatePeer(pk.PublicKey(), pk, nil)
}

func (e *Execution) getNonceAndIncrement(from cmn.Address) uint64 {
	nonce, f := e.nonces[from]
	if !f {
		account, err := e.client.Pbc().GetAccount(context.Background(), &pb.GetAccountRequest{
			Address: from.Bytes(),
		})
		if err != nil {
			log.Error()
			return 0
		}

		nonce = account.GetAccount().GetNonce()
	}
	nonce += 1
	e.nonces[from] = nonce
	return nonce
}
