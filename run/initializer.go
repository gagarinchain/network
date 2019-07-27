package main

import (
	"context"
	"github.com/poslibp2p"
	"github.com/poslibp2p/blockchain"
	"github.com/poslibp2p/blockchain/state"
	"github.com/poslibp2p/common"
	"github.com/poslibp2p/common/message"
	"github.com/poslibp2p/hotstuff"
	"github.com/poslibp2p/network"

	"time"
)

type Context struct {
	node              *network.Node
	blockProtocol     *blockchain.BlockProtocol
	hotStuff          *hotstuff.Protocol
	pacer             *hotstuff.StaticPacer
	srv               network.Service
	txService         *blockchain.TxService
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

func CreateContext(cfg *network.NodeConfig, committee []*common.Peer, me *common.Peer) *Context {
	validators := []poslibp2p.Validator{
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

	hotstuffSrv := network.CreateService(context.Background(), node, dispatcher, txDispatcher)
	txService := blockchain.NewService(blockchain.NewTransactionValidator(), pool)
	storage, _ := blockchain.NewStorage(cfg.DataDir, nil)
	bsrv := blockchain.NewBlockService(hotstuffSrv)
	db := state.NewStateDB()
	bc := blockchain.CreateBlockchainFromStorage(storage, bsrv, pool, db)
	synchr := blockchain.CreateSynchronizer(me, bsrv, bc)
	protocol := blockchain.CreateBlockProtocol(hotstuffSrv, bc, synchr)

	config := &hotstuff.ProtocolConfig{
		F:          4,
		Delta:      10 * time.Second,
		Blockchain: bc,
		Me:         me,
		Srv:        hotstuffSrv,
		Storage:    storage,
		Sync:       synchr,
		Validators: validators,
		Committee:  committee,
	}

	pacer := hotstuff.CreatePacer(config)
	config.Pacer = pacer
	p := hotstuff.CreateProtocol(config)
	log.Debugf("%+v\n", p)
	return &Context{
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
	}
}

func filterSelf(peers []*common.Peer, self *common.Peer) (res []*common.Peer) {
	for _, p := range peers {
		if !p.Equals(self) {
			res = append(res, p)
		}
	}

	return res
}

func (c *Context) Bootstrap() {
	rootCtx := context.Background()
	statusChan, errChan := c.node.Bootstrap(rootCtx)

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
	go c.pacer.Run(rootCtx, c.hotstuffChan, c.epochChan)
}
