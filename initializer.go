package main

import (
	"context"
	"fmt"
	"github.com/libp2p/go-libp2p-peerstore"
	"github.com/poslibp2p/blockchain"
	"github.com/poslibp2p/eth/crypto"
	"github.com/poslibp2p/hotstuff"
	"github.com/poslibp2p/message"
	"github.com/poslibp2p/message/protobuff"
	"github.com/poslibp2p/network"

	"time"
)

type Context struct {
	node          *network.Node
	blockProtocol *blockchain.BlockProtocol
	hotStuff      *hotstuff.Protocol
	pacer         *hotstuff.StaticPacer
}

func CreateContext(cfg *network.NodeConfig) *Context {
	handlers := make(map[pb.Message_MessageType]message.Handler)
	dispatcher := &message.Dispatcher{Handlers: handlers, MsgChan: make(chan *message.Message, 1024)}

	node, err := network.CreateNode(cfg)
	if err != nil {
		log.Fatal(err)
	}
	node.Dispatcher = dispatcher

	log.Infof("This is my addrs %v", node.Host.Addrs())
	fullAddr := fmt.Sprintf("/ip4/127.0.0.1/tcp/%d/p2p/%s", cfg.Port, node.Host.ID().Pretty())
	log.Infof("Now run \"./poslibp2p -l %d -d %s\" on a different terminal\n", cfg.Port+1, fullAddr)

	me := generateIdentity(node.GetPeerInfo())

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
	go c.hotStuff.Run(msgChan)
	go c.pacer.Run(context.Background())
}

func generateIdentity(pi *peerstore.PeerInfo) *message.Peer {
	privateKey, e := crypto.GenerateKey() //Load keys here
	if e != nil {
		log.Error("failed to generate key")
	}
	return message.CreatePeer(&privateKey.PublicKey, privateKey, pi)
}
