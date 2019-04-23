package main

import (
	"context"
	"github.com/poslibp2p"
	"github.com/poslibp2p/blockchain"
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
	hotstuffChan      chan *message.Message
	blockProtocolChan chan *message.Message
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

func CreateContext(cfg *network.NodeConfig, me *common.Peer) *Context {
	validators := []poslibp2p.Validator{
		hotstuff.NewEpochStartValidator(cfg.Committee),
		hotstuff.NewProposalValidator(cfg.Committee),
		hotstuff.NewVoteValidator(cfg.Committee),
	}
	msgChan := make(chan *message.Message)
	blockChan := make(chan *message.Message)
	dispatcher := message.NewDispatcher(validators, msgChan, blockChan)

	node, err := network.CreateNode(cfg)
	if err != nil {
		log.Fatal(err)
	}

	log.Infof("This is my id %v", node.Host.ID().Pretty())

	srv := network.CreateService(context.Background(), node, dispatcher)
	storage, _ := blockchain.NewStorage(cfg.DataDir, nil)
	bsrv := blockchain.NewBlockService(srv)
	bc := blockchain.CreateBlockchainFromStorage(storage, bsrv)
	synchr := blockchain.CreateSynchronizer(me, bsrv, bc)
	protocol := blockchain.CreateBlockProtocol(srv, bc, synchr)

	config := &hotstuff.ProtocolConfig{
		F:           4,
		Delta:       10 * time.Second,
		Blockchain:  bc,
		Me:          me,
		Srv:         srv,
		Storage:     storage,
		Sync:        synchr,
		Committee:   cfg.Committee,
		ControlChan: make(chan hotstuff.Command),
	}

	pacer := hotstuff.CreatePacer(config)
	config.Pacer = pacer
	p := hotstuff.CreateProtocol(config)
	log.Debugf("%+v\n", p)
	pacer.SetViewGetter(p)
	pacer.SetEventNotifier(p)

	return &Context{
		node:              node,
		blockProtocol:     protocol,
		hotStuff:          p,
		pacer:             pacer,
		srv:               srv,
		hotstuffChan:      msgChan,
		blockProtocolChan: blockChan,
	}
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
	go c.blockProtocol.Run(rootCtx, c.blockProtocolChan)

	ints, errors := c.srv.Bootstrap(rootCtx)
	select {
	case <-ints:
		log.Debug("Network service bootstrapped successfully")
	case e := <-errors:
		log.Error(e)
	}

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

	go c.hotStuff.Run(c.hotstuffChan)
	go c.pacer.Run(rootCtx)
}
