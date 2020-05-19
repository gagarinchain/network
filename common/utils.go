package common

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/gagarinchain/network/common/eth/crypto"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/op/go-logging"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"strconv"
)

var log = logging.MustGetLogger("hotstuff")

type Settings struct {
	Hotstuff struct {
		N          int `yaml:"N"`
		Delta      int `yaml:"Delta"`
		BlockDelta int `yaml:"BlockDelta"`
	} `yaml:"Hotstuff"`
	Network struct {
		MinPeerThreshold  int `yaml:"MinPeerThreshold"`
		ReconnectPeriod   int `yaml:"ReconnectPeriod"`
		ConnectionTimeout int `yaml:"ConnectionTimeout"`
	} `yaml:"Network"`
	Storage struct {
		Dir string `yaml:"Dir"`
	} `yaml:"Storage"`
	Static struct {
		Dir string `yaml:"Dir"`
	} `yaml:"Static"`
}

func GenerateIdentities() {
	committee := CommitteeData{}

	for i := 0; i < 10; i++ {
		pk, _ := crypto.GenerateKey()
		pkbytes := pk.V().Serialize()
		pubbytes := pk.PublicKey().Bytes()
		pkstring := hex.EncodeToString(pkbytes[:])
		pubstring := hex.EncodeToString(pubbytes[:])
		address := crypto.PubkeyToAddress(pk.PublicKey())

		privKey, _, _ := p2pcrypto.GenerateSecp256k1Key(rand.Reader)
		id, _ := peer.IDFromPrivateKey(privKey)
		b, _ := p2pcrypto.MarshalPrivateKey(privKey)

		v := map[string]interface{}{"addr": address, "pk": pkstring, "pub": pubstring, "id": id.Pretty(), "pkpeer": hex.EncodeToString(b)}

		marshal, _ := json.Marshal(v)
		var out bytes.Buffer
		if err := json.Indent(&out, marshal, "", "\t"); err != nil {
			panic(err)
		}

		log.Info(out.String())
		err := ioutil.WriteFile("static/peer"+strconv.Itoa(i)+".json", out.Bytes(), 0644)

		if err != nil {
			panic(err)
		}

		multiaddr := fmt.Sprintf("/ip4/127.0.0.1/tcp/908%d/p2p/%s", i, id.Pretty())

		p := PeerData{
			Address:      address.Hex(),
			Pub:          pubstring,
			MultiAddress: multiaddr,
		}

		committee.Peers = append(committee.Peers, p)
	}

	marshal, e := json.Marshal(committee)
	if e != nil {
		panic(e)
	}

	var out bytes.Buffer
	if err := json.Indent(&out, marshal, "", "\t"); err != nil {
		panic(err)
	}

	if err := ioutil.WriteFile("static/peers.json", out.Bytes(), 0644); err != nil {
		panic(err)
	}

}

func ReadSettings() (s *Settings) {
	settingsPath, found := os.LookupEnv("GN_SETTINGS")
	if !found {
		settingsPath = "static/settings.yaml"
	}

	file, e := os.Open(settingsPath)
	if e != nil {
		log.Error("Can't load settings, using default", e)
	} else {
		defer file.Close()
		s = &Settings{}
		byteValue, _ := ioutil.ReadAll(file)
		if err := yaml.Unmarshal(byteValue, s); err != nil {
			log.Error("Can't load settings, using default", e)
		}
	}
	if s == nil {
		s = &Settings{
			Hotstuff: struct {
				N          int `yaml:"N"`
				Delta      int `yaml:"Delta"`
				BlockDelta int `yaml:"BlockDelta"`
			}{N: 10, Delta: 5000, BlockDelta: 10},
			Network: struct {
				MinPeerThreshold  int `yaml:"MinPeerThreshold"`
				ReconnectPeriod   int `yaml:"ReconnectPeriod"`
				ConnectionTimeout int `yaml:"ConnectionTimeout"`
			}{MinPeerThreshold: 3, ReconnectPeriod: 10000, ConnectionTimeout: 3000},
			Storage: struct {
				Dir string `yaml:"Dir"`
			}{Dir: os.TempDir()},
			Static: struct {
				Dir string `yaml:"Dir"`
			}{Dir: "/Users/dabasov/Projects/gagarin/network/static"},
		}
	}
	return
}
