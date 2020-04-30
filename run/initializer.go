package main

import (
	"context"
	net "github.com/gagarinchain/network"
	"github.com/gagarinchain/network/blockchain"
	"github.com/gagarinchain/network/blockchain/state"
	"github.com/gagarinchain/network/common"
	"github.com/gagarinchain/network/common/message"
	pb "github.com/gagarinchain/network/common/protobuff"
	"github.com/gagarinchain/network/hotstuff"
	"github.com/gagarinchain/network/network"
	"path"
	"time"
)

type Context struct {
	me                *common.Peer
	node              *network.Node
	blockProtocol     *blockchain.BlockProtocol
	hotStuff          *hotstuff.Protocol
	pacer             *hotstuff.StaticPacer
	srv               network.Service
	txService         *blockchain.TxService
	eventBuss         *network.GagarinEventBus
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

func CreateContext(cfg *network.NodeConfig, committee []*common.Peer, me *common.Peer, s *Settings) *Context {
	validators := []net.Validator{
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

	cfg.Committee = filterSelf(cfg.Committee, me)
	node, err := network.CreateNode(cfg)
	if err != nil {
		log.Fatal(err)
	}

	log.Infof("This is my id %v", node.Host.ID().Pretty())
	pool := blockchain.NewTransactionPool()

	events := make(chan *common.Event)
	bus := network.NewGagarinEventBus(events)

	hotstuffSrv := network.CreateService(context.Background(), node, dispatcher, txDispatcher, bus)
	storage, _ := common.NewStorage(cfg.DataDir, nil)
	txValidator := blockchain.NewTransactionValidator(committee)
	headerValidator := &blockchain.HeaderValidator{}
	bsrv := blockchain.NewBlockService(hotstuffSrv, blockchain.NewBlockValidator(committee, txValidator, headerValidator), headerValidator)
	db := state.NewStateDB(storage)
	seed := blockchain.SeedFromFile(path.Join(s.Static.Dir, "seed.json"))
	bc := blockchain.CreateBlockchainFromStorage(&blockchain.BlockchainConfig{
		Seed:           seed,
		BlockPerister:  &blockchain.BlockPersister{Storage: storage},
		ChainPersister: &blockchain.BlockchainPersister{Storage: storage},
		Pool:           pool,
		Db:             db,
		Storage:        storage,
		Delta:          time.Duration(s.Hotstuff.BlockDelta) * time.Millisecond,
		EventBus:       bus,
	})
	synchr := blockchain.CreateSynchronizer(me, bsrv, bc, -1, 20, 3, 3, 5, int32(2*s.Hotstuff.N))
	protocol := blockchain.CreateBlockProtocol(hotstuffSrv, bc, synchr)

	initialState := getInitialState(storage, bc)
	reqDispatcher := blockchain.NewRequestHandler(bc, db)
	bus.AddHandler(pb.Request_ACCOUNT, reqDispatcher.HandleAccountRequest)
	bus.AddHandler(pb.Request_BLOCK, reqDispatcher.HandleBlockRequest)

	config := &hotstuff.ProtocolConfig{
		F:            s.Hotstuff.N,
		Delta:        time.Duration(s.Hotstuff.Delta) * time.Millisecond,
		Blockchain:   bc,
		Me:           me,
		Srv:          hotstuffSrv,
		InitialState: initialState,
		Sync:         synchr,
		Validators:   validators,
		Committee:    committee,
		Storage:      storage,
	}

	txService := blockchain.NewService(txValidator, pool, hotstuffSrv, bc, me)

	pacer := hotstuff.CreatePacer(config)
	config.Pacer = pacer
	p := hotstuff.CreateProtocol(config)
	log.Debugf("%+v\n", p)
	bc.SetProposerGetter(pacer)
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
	}
}

func getInitialState(storage net.Storage, bc blockchain.Blockchain) *hotstuff.InitialState {
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

func (c *Context) Bootstrap(s *Settings) {
	rootCtx := context.Background()
	config := &network.BootstrapConfig{
		Period:            time.Duration(s.Network.ReconnectPeriod) * time.Millisecond,
		MinPeerThreshold:  s.Network.MinPeerThreshold,
		ConnectionTimeout: time.Duration(s.Network.ConnectionTimeout) * time.Millisecond,
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
	ints, errors := c.srv.Bootstrap(rootCtx)
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
		func(event hotstuff.Event) {
			if event.T == hotstuff.EpochStarted {
				epoch := event.Payload.(int32)
				vMsg := &pb.EpochStartedPayload{Epoch: epoch}
				ev := &common.Event{T: common.EpochStarted, Payload: vMsg}
				c.eventBuss.FireEvent(ev)
			}
			if event.T == hotstuff.ChangedView {
				view := event.Payload.(int32)
				vMsg := &pb.ViewChangedPayload{View: view}
				ev := &common.Event{T: common.ChangedView, Payload: vMsg}
				c.eventBuss.FireEvent(ev)
			}
		},
		map[hotstuff.EventType]interface{}{
			hotstuff.EpochStarted: struct{}{},
			hotstuff.ChangedView:  struct{}{},
		})
	go func() {
		for {
			c.eventBuss.Run(rootCtx)
		}
	}()
	go c.pacer.Run(rootCtx, c.hotstuffChan, c.epochChan)
}
