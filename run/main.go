package main

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	golog "github.com/ipfs/go-log"
	p2pcrypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/op/go-logging"
	cmn "github.com/poslibp2p/common"
	"github.com/poslibp2p/common/eth/common"
	"github.com/poslibp2p/common/eth/crypto"
	"github.com/poslibp2p/network"
	gologging "github.com/whyrusleeping/go-logging"
	"io/ioutil"
	"os"
	"path"
	"strconv"
)

var stdoutLogFormat = logging.MustStringFormatter(
	`%{color:reset}%{color}%{time:15:04:05.000} [%{shortfunc}] [%{level}] %{message}`,
)

var log = logging.MustGetLogger("main")

func main() {
	// LibP2P code uses golog to log messages. They log with different
	// string IDs (i.e. "swarm"). We can control the verbosity level for
	// all loggers with:
	golog.SetAllLoggers(gologging.INFO)

	// Parse options from the command line
	ind := flag.Int("l", -1, "peer index")
	flag.Parse()

	if *ind == -1 {
		log.Fatal("Please provide peer index with -l")
	}

	var v map[string]interface{}
	index := strconv.Itoa(*ind)
	bytes, e := ioutil.ReadFile("static/peer" + index + ".json")
	if e != nil {
		log.Fatal(e)
	}
	e = json.Unmarshal(bytes, &v)
	if e != nil {
		log.Fatal(e)
	}

	// First let's create a new identity key pair for our node. If this was your
	// application you would likely save this private key to a database and load
	// it from the db on subsequent start ups.
	pkpeer := v["pkpeer"].(string)
	addr := v["addr"].(string)
	pk := v["pk"].(string)

	decodeString, e := hex.DecodeString(pkpeer)
	privKey, err := p2pcrypto.UnmarshalPrivateKey(decodeString)
	if err != nil {
		log.Fatal(err)
	}

	var loader cmn.CommitteeLoader = &cmn.CommitteeLoaderImpl{}
	committee := loader.LoadFromFile("static/peers.json")
	// Next we'll create the node config
	cfg := &network.NodeConfig{
		PrivateKey: privKey,
		Port:       9080 + uint16(*ind),
		DataDir:    path.Join(os.TempDir(), strconv.Itoa(*ind)),
		Committee:  committee[0:2],
	}

	var me *cmn.Peer
	for _, p := range committee {
		if common.HexToAddress(addr) == p.GetAddress() {
			me = p
			pkbytes, e := hex.DecodeString(pk)
			if e != nil {
				log.Error(e)
				return
			}
			key, e := crypto.ToECDSA(pkbytes)
			if e != nil {
				log.Error(e)
				return
			}
			p.SetPrivateKey(key)
			break
		}
	}

	ctx := CreateContext(cfg, me)

	// Ok now we can bootstrap the node. This could take a little bit if we're
	// running on a live network.
	ctx.Bootstrap()

	select {}

}
