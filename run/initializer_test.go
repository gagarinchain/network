package main

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"gopkg.in/yaml.v2"
	"testing"
)

var data = `
Hotstuff:
  N: 10
  Delta: 5000
  BlockDelta: 10
Network:
  MinPeerThreshold: 3
  ReconnectPeriod: 10000
  ConnectionTimeout: 3000
`

var data2 = `
a: Easy!
b:
  c: 2
  d: [3, 4]
`

type T struct {
	A string
	B struct {
		RenamedC int   `yaml:"c"`
		D        []int `yaml:",flow"`
	}
}

//todo fix it
func TestInitializing(t *testing.T) {
	s := &Settings{}

	if err := yaml.Unmarshal([]byte(data), &s); err != nil {
		t.Error(err)
	}

	spew.Dump(s)

	tt := T{}

	if err := yaml.Unmarshal([]byte(data2), &tt); err != nil {
		t.Error(err)
	}

	fmt.Printf("--- t:\n%v\n\n", tt)

	//privKey, _, err := p2pcrypto.GenerateECDSAKeyPair(rand.Reader)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	//// Next we'll create the node config
	//cfg := &network.NodeConfig{
	//	PrivateKey: privKey,
	//	Port:       uint16(8081),
	//	DataDir:    path.Join(os.TempDir(), strconv.Itoa(8081)),
	//}
	//
	//ctx := CreateContext(cfg, generateIdentity(nil))
	//
	//assert.Equal(t, int32(1), ctx.HotStuff().getCurrentView())
	//assert.Equal(t, ctx.Node().Host.ID().Pretty(), ctx.Node().GetPeerInfo().ID.Pretty())
	//assert.Equal(t, 10, len(ctx.Pacer().Committee()))

}
