package run

import (
	"context"
	"github.com/gagarinchain/common"
	"github.com/gagarinchain/common/api"
	common2 "github.com/gagarinchain/common/eth/common"
	"github.com/gagarinchain/common/message"
	"github.com/gagarinchain/common/network"
	pb "github.com/gagarinchain/common/protobuff"
	"github.com/gagarinchain/network/blockchain"
	"github.com/gagarinchain/network/blockchain/state"
	"github.com/gagarinchain/network/blockchain/tx"
	"github.com/gagarinchain/network/hotstuff"
	"github.com/gagarinchain/network/rpc"
	"github.com/gagarinchain/network/storage"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/multiformats/go-multiaddr"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strconv"
	"time"
)

type Context struct {
	me                *common.Peer
	node              *network.Node
	blockProtocol     *blockchain.BlockProtocol
	hotStuff          *hotstuff.Protocol
	pacer             *hotstuff.StaticPacer
	srv               network.Service
	txService         *tx.TxService
	eventBuss         *network.GagarinEventBus
	rpc               *rpc.Service
	storage           storage.Storage
	hotstuffChan      chan *message.Message
	epochChan         chan *message.Message
	blockProtocolChan chan *message.Message
	txChan            chan *message.Message
}

func (c *Context) Pacer() *hotstuff.StaticPacer {
	return c.pacer
}

func (c *Context) HotStuff() *hotstuff.Protocol {
	return c.hotStuff
}

func (c *Context) BlockProtocol() *blockchain.BlockProtocol {
	return c.blockProtocol
}

func (c *Context) Node() *network.Node {
	return c.node
}

func CreateContext(s *common.Settings) *Context {
	var extMA multiaddr.Multiaddr
	if s.Network.ExtAddr != "" {
		extMA, _ = multiaddr.NewMultiaddr(s.Network.ExtAddr)
	}

	var loader common.CommitteeLoader = &common.CommitteeLoaderImpl{}
	committee := loadCommittee(s, loader, s.Hotstuff.CommitteeSize)

	peerKey := loadMyKey(s, loader, committee)

	me := committee[s.Hotstuff.Me]

	// Next we'll create the node config
	validators := []api.Validator{
		hotstuff.NewEpochStartValidator(committee),
		hotstuff.NewProposalValidator(committee),
		hotstuff.NewVoteValidator(committee),
	}
	msgChan := make(chan *message.Message)
	epochChan := make(chan *message.Message)
	blockChan := make(chan *message.Message)
	txChan := make(chan *message.Message)
	dispatcher := message.NewHotstuffDispatcher(msgChan, epochChan, blockChan)
	txDispatcher := message.NewTxDispatcher(txChan)

	dataDir := path.Join(s.Storage.Dir, strconv.Itoa(s.Hotstuff.Me))
	node, err := network.CreateNode(&network.NodeConfig{
		PrivateKey:        peerKey,
		Port:              9080,
		DataDir:           dataDir,
		ExternalMultiaddr: extMA,
		Committee:         filterSelf(committee, me),
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Infof("This is my id %v", node.Host.ID().Pretty())
	pool := tx.NewTransactionPool()

	events := make(chan *common.Event)
	bus := network.NewGagarinEventBus(events)

	hotstuffSrv := network.CreateService(context.Background(), node, dispatcher, txDispatcher, bus)
	storage, _ := storage.NewStorage(dataDir, nil)
	txValidator := blockchain.NewTransactionValidator(committee)
	headerValidator := &blockchain.HeaderValidator{}
	bsrv := blockchain.NewBlockService(hotstuffSrv, blockchain.NewBlockValidator(committee, txValidator, headerValidator), headerValidator)
	var addresses []common2.Address
	for _, peer := range committee {
		addresses = append(addresses, peer.GetAddress())
	}
	db := state.NewStateDB(storage, addresses, bus)
	seed := blockchain.SeedFromFile(path.Join(s.Static.Dir, "seed.json"))

	plugins := rpc.NewPluginAdapter(me)
	for _, plugin := range s.Plugins {
		if plugin.Address != "" {
			plugins.AddPlugin(plugin)
		}
	}

	bc := blockchain.CreateBlockchainFromStorage(&blockchain.BlockchainConfig{
		Seed:              seed,
		BlockPerister:     &blockchain.BlockPersister{Storage: storage},
		ChainPersister:    &blockchain.BlockchainPersister{Storage: storage},
		Pool:              pool,
		Db:                db,
		Storage:           storage,
		Delta:             time.Duration(s.Hotstuff.BlockDelta) * time.Millisecond,
		EventBus:          bus,
		OnNewBlockCreated: plugins,
	})

	//todo move parameters to settings and add config structure
	synchr := blockchain.CreateSynchronizer(me, bsrv, bc, -1, 20, 3, 3, 5, int32(2*s.Hotstuff.CommitteeSize))
	protocol := blockchain.CreateBlockProtocol(hotstuffSrv, bc, synchr)

	initialState := getInitialState(storage, bc)
	reqDispatcher := blockchain.NewRequestHandler(bc, db)
	bus.AddHandler(pb.Request_ACCOUNT, reqDispatcher.HandleAccountRequest)
	bus.AddHandler(pb.Request_BLOCK, reqDispatcher.HandleBlockRequest)

	config := &hotstuff.ProtocolConfig{
		F:                 s.Hotstuff.CommitteeSize,
		Delta:             time.Duration(s.Hotstuff.Delta) * time.Millisecond,
		Blockchain:        bc,
		Me:                me,
		Srv:               hotstuffSrv,
		Sync:              synchr,
		Validators:        validators,
		Storage:           storage,
		Committee:         committee,
		InitialState:      initialState,
		OnReceiveProposal: plugins,
		OnVoteReceived:    plugins,
		OnProposal:        plugins,
		OnBlockCommit:     plugins,
	}

	txService := tx.NewService(txValidator, pool, hotstuffSrv, bc, me)

	pacer := hotstuff.CreatePacer(config)
	config.Pacer = pacer
	p := hotstuff.CreateProtocol(config)
	log.Debugf("%+v\n", p)
	bc.SetProposerGetter(pacer)

	var rpcService *rpc.Service
	if s.Rpc.Address != "" {
		rpcService = rpc.NewService(bc, pacer, db)
	}

	return &Context{
		me:                me,
		node:              node,
		blockProtocol:     protocol,
		hotStuff:          p,
		pacer:             pacer,
		srv:               hotstuffSrv,
		hotstuffChan:      msgChan,
		epochChan:         epochChan,
		blockProtocolChan: blockChan,
		txService:         txService,
		txChan:            txChan,
		eventBuss:         bus,
		rpc:               rpcService,
		storage:           storage,
	}
}

func loadMyKey(s *common.Settings, loader common.CommitteeLoader, committee []*common.Peer) crypto.PrivKey {
	peerPath := path.Join(s.Static.Dir, "peer"+strconv.Itoa(s.Hotstuff.Me)+".json")
	peerKey, err := loader.LoadPeerFromFile(peerPath, committee[s.Hotstuff.Me])
	if err != nil {
		log.Fatal("Could't load peer credentials")
	}
	return peerKey
}

func loadCommittee(s *common.Settings, loader common.CommitteeLoader, size int) []*common.Peer {
	peersPath := filepath.Join(s.Static.Dir, "peers.json")
	committee := loader.LoadPeerListFromFile(peersPath)

	if len(committee) < size {
		log.Fatal("Can't load all committee info")
	}
	if len(committee) > size {
		committee = committee[:size]
	}
	return committee
}

func getInitialState(storage storage.Storage, bc api.Blockchain) *hotstuff.InitialState {
	initialState := &hotstuff.InitialState{
		View:              int32(0),
		Epoch:             int32(-1),
		VHeight:           0,
		LastExecutedBlock: bc.GetGenesisBlock().Header(),
		HQC:               bc.GetGenesisBlock().QC(),
	}
	persister := &hotstuff.PacerPersister{Storage: storage}
	p := &hotstuff.ProtocolPersister{Storage: storage}
	epoch, e1 := persister.GetCurrentEpoch()
	view, e2 := persister.GetCurrentView()
	vheight, e3 := p.GetVHeight()
	last, e4 := p.GetLastExecutedBlockHash()
	hqc, e5 := p.GetHQC()
	if e1 != nil {
		log.Debug("no epoch is stored")
	} else if e2 != nil {
		log.Debug("no view is stored")
	} else if e3 != nil {
		log.Debug("no vheight is stored")
	} else if e4 != nil {
		log.Debug("no last executed block is stored")
	} else if e5 != nil {
		log.Debug("no hqc is stored")
	} else {
		initialState = &hotstuff.InitialState{
			View:              view,
			Epoch:             epoch,
			VHeight:           vheight,
			LastExecutedBlock: bc.GetBlockByHash(last).Header(),
			HQC:               hqc,
		}
	}
	return initialState
}

func filterSelf(peers []*common.Peer, self *common.Peer) (res []*common.Peer) {
	for _, p := range peers {
		if !p.Equals(self) {
			res = append(res, p)
		}
	}

	return res
}

func (c *Context) Bootstrap(s *common.Settings) {
	rootCtx, cancel := context.WithCancel(context.Background())
	c.handleInterrupt(cancel)

	config := &network.BootstrapConfig{
		Period:            time.Duration(s.Network.ReconnectPeriod) * time.Millisecond,
		MinPeerThreshold:  s.Network.MinPeerThreshold,
		ConnectionTimeout: time.Duration(s.Network.ConnectionTimeout) * time.Millisecond,
		RendezvousNs:      "/hotstuff",
	}
	statusChan, errChan := c.node.Bootstrap(rootCtx, config)

	for {
		select {
		case <-statusChan:
			log.Debug("Node bootstrapped successfully")
			goto END
		case e := <-errChan:
			log.Error(e)
		}
	}
END:
	ints, errors := c.srv.Bootstrap(rootCtx, config)
	select {
	case <-ints:
		log.Debug("Network service bootstrapped successfully")
	case e := <-errors:
		log.Error(e)
	}

	go c.txService.Run(rootCtx, c.txChan)
	go c.blockProtocol.Run(rootCtx, c.blockProtocolChan)

	respChan, errChans := c.blockProtocol.Bootstrap(rootCtx)
	for {
		select {
		case <-respChan:
			log.Debug("Block protocol bootstrapped successfully")
			goto END_BP
		case e := <-errChans:
			log.Error(e)
		}
	}
END_BP:
	c.pacer.Bootstrap(rootCtx, c.hotStuff)
	c.pacer.SubscribeEvents(
		rootCtx,
		func(event api.Event) {
			if event.T == api.EpochStarted {
				epoch := event.Payload.(int32)
				vMsg := &pb.EpochStartedPayload{Epoch: epoch}
				ev := &common.Event{T: common.EpochStarted, Payload: vMsg}
				c.eventBuss.FireEvent(ev)
			}
			if event.T == api.ChangedView {
				view := event.Payload.(int32)
				vMsg := &pb.ViewChangedPayload{View: view}
				ev := &common.Event{T: common.ViewChanged, Payload: vMsg}
				c.eventBuss.FireEvent(ev)
			}
		},
		map[api.EventType]interface{}{
			api.EpochStarted: struct{}{},
			api.ChangedView:  struct{}{},
		})
	go func() {
		c.eventBuss.Run(rootCtx)
	}()
	go c.pacer.Run(rootCtx, c.hotstuffChan, c.epochChan)
	if s.Rpc.Address != "" {
		if err := c.rpc.Bootstrap(rpc.Config{
			Address:              s.Rpc.Address,
			MaxConcurrentStreams: s.Rpc.MaxConcurrentStreams,
		}); err != nil {
			log.Fatal("Can't start grpc service", err)
		}
	}
}

func (c *Context) handleInterrupt(cancel context.CancelFunc) chan bool {
	res := make(chan bool)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	go func() { // Block until a signal is received.
		sig := <-sigChan
		log.Infof("Received %v signal. Shutting down Gagarin.bus", sig)
		cancel()
		c.node.Shutdown()
		c.storage.Close()

		log.Info("Shut down completed")
		os.Exit(0)
	}()

	return res
}
