package run

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/prysmaticlabs/go-ssz"
	"testing"
	"time"
)

//todo fix it
//func TestInitializing(t *testing.T) {
//	privKey, _, err := p2pcrypto.GenerateECDSAKeyPair(rand.Reader)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Next we'll create the node config
//	cfg := &bus.NodeConfig{
//		PrivateKey: privKey,
//		Port:       uint16(8081),
//		DataDir:    path.Join(os.TempDir(), strconv.Itoa(8081)),
//	}
//
//	ctx := CreateContext(cfg, generateIdentity(nil))
//
//	assert.Equal(t, int32(1), ctx.HotStuff().getCurrentView())
//	assert.Equal(t, ctx.Node().Host.ID().Pretty(), ctx.Node().GetPeerInfo().ID.Pretty())
//	assert.Equal(t, 10, len(ctx.Pacer().Committee()))
//
//}

//func TestScenarioParse(t *testing.T) {
//	s := GetScenarioFromFile("/Users/dabasov/Projects/gagarin/bus/static/scenario.yaml")
//
//	spew.Dump(s)
//}

func TestSsz(t *testing.T) {
	type T struct {
		From  int32
		To    int32
		Value uint64
		//non serialized field for map

	}
	type P struct {
		Hash      []byte
		Height    uint32
		Timestamp uint64
		Txs       []*T
		Data      []byte
	}

	p := P{
		Hash:      []byte{0x33, 0x34, 0x35, 0x36, 0x34, 0x35, 0x36},
		Height:    1,
		Timestamp: uint64(time.Now().Unix()),
		Data:      []byte{0x11, 0x12, 0x13, 0x14, 0x15},
		Txs: []*T{{
			From:  -1,
			To:    -1,
			Value: 10,
		}, {
			From:  10,
			To:    12,
			Value: 4,
		}, {
			From:  4,
			To:    43,
			Value: 33,
		}},
	}

	marshal, err := ssz.Marshal(p)

	spew.Dump(err)
	spew.Dump(marshal)
}
