package main

import (
	"flag"
	golog "github.com/ipfs/go-log"
	"github.com/op/go-logging"
	cmn "github.com/poslibp2p/common"
	"github.com/poslibp2p/network"
	gologging "github.com/whyrusleeping/go-logging"
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
	index := strconv.Itoa(*ind)
	var loader cmn.CommitteeLoader = &cmn.CommitteeLoaderImpl{}
	committee := loader.LoadPeerListFromFile("static/peers.json")
	peerKey, err := loader.LoadPeerFromFile("static/peer"+index+".json", committee[*ind])

	if err != nil {
		log.Fatal("Could't load peer credentials")
	}

	// Next we'll create the node config
	cfg := &network.NodeConfig{
		PrivateKey: peerKey,
		Port:       9080 + uint16(*ind),
		DataDir:    path.Join(os.TempDir(), strconv.Itoa(*ind)),
		Committee:  committee[0:4],
	}

	ctx := CreateContext(cfg, committee[*ind])

	// Ok now we can bootstrap the node. This could take a little bit if we're
	// running on a live network.
	ctx.Bootstrap()

	select {}

}
