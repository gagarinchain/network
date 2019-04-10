package run

import (
	"context"
	"github.com/libp2p/go-libp2p-peerstore"
	"github.com/poslibp2p/blockchain"
	"github.com/poslibp2p/common/eth/crypto"
	"github.com/poslibp2p/common/message"
	"github.com/poslibp2p/common/protobuff"
	"github.com/poslibp2p/hotstuff"
	"github.com/poslibp2p/network"

	"time"
)

type Context struct {
	node          *network.Node
	blockProtocol *blockchain.BlockProtocol
	hotStuff      *hotstuff.Protocol
	pacer         *hotstuff.StaticPacer
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
	handlers := make(map[pb.Message_MessageType]message.Handler)
	dispatcher := &message.Dispatcher{Handlers: handlers, MsgChan: make(chan *message.Message, 1024)}

	node, err := network.CreateNode(cfg)
	if err != nil {
		log.Fatal(err)
	}
	node.Dispatcher = dispatcher

	log.Infof("This is my id %v", node.Host.ID().Pretty())

	srv := network.CreateService(context.Background(), node, dispatcher)
	storage, _ := blockchain.NewStorage(cfg.DataDir, nil)
	bsrv := blockchain.NewBlockService(srv)
	bc := blockchain.CreateBlockchainFromGenesisBlock(storage, bsrv)
	synchr := blockchain.CreateSynchronizer(me, bsrv, bc)
	protocol := blockchain.CreateBlockProtocol(srv, bc, synchr)

	config := &hotstuff.ProtocolConfig{
		F:           10,
		Delta:       1 * time.Second,
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
	pacer.SetViewGetter(p)
	pacer.SetEventNotifier(p)

	return &Context{
		node:          node,
		blockProtocol: protocol,
		hotStuff:      p,
		pacer:         pacer,
	}
}

func (c *Context) Bootstrap() {
	if err := c.node.Bootstrap(); err != nil {
		log.Fatal("Can't start network services")
	}

	msgChan := make(chan *message.Message)
	go c.node.SubscribeAndListen(context.Background(), msgChan)
	go c.blockProtocol.Bootstrap(context.Background())
	go c.hotStuff.Run(msgChan)
	go c.pacer.Run(context.Background())
}

func generateIdentity(pi *peerstore.PeerInfo) *common.Peer {
	privateKey, e := crypto.GenerateKey() //Load keys here
	if e != nil {
		log.Error("failed to generate key")
	}
	return message.CreatePeer(&privateKey.PublicKey, privateKey, pi)
}
