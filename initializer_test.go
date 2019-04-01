package main

import (
	"crypto/rand"
	p2pcrypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/poslibp2p/network"
	"github.com/stretchr/testify/assert"
	"os"
	"path"
	"strconv"
	"testing"
)

func TestInitializing(t *testing.T) {
	privKey, _, err := p2pcrypto.GenerateECDSAKeyPair(rand.Reader)
	if err != nil {
		log.Fatal(err)
	}

	// Next we'll create the node config
	cfg := &network.NodeConfig{
		PrivateKey: privKey,
		Port:       uint16(8081),
		DataDir:    path.Join(os.TempDir(), strconv.Itoa(8081)),
	}

	ctx := CreateContext(cfg)

	assert.Equal(t, int32(3), ctx.hotStuff.GetCurrentView())
	assert.Equal(t, ctx.node.Host.ID().Pretty(), ctx.node.GetPeerInfo().ID.Pretty())
	assert.Equal(t, 10, len(ctx.pacer.Committee()))

}