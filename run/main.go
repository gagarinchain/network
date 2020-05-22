package main

import (
	"flag"
	cmn "github.com/gagarinchain/common"
	"github.com/gagarinchain/common/api"
	"github.com/gagarinchain/network/network"
	golog "github.com/ipfs/go-log"
	"github.com/multiformats/go-multiaddr"
	"github.com/op/go-logging"
	"os"
	"path"
	"path/filepath"
	"plugin"
	"strconv"
)

var stdoutLogFormat = logging.MustStringFormatter(
	`%{time:15:04:05.000} [%{shortfile}] [%{level}] %{message}`,
)

var log = logging.MustGetLogger("main")

func main() {
	// LibP2P code uses golog to log messages. They log with different
	// string IDs (i.e. "swarm"). We can control the verbosity level for
	// all loggers with:
	level, _ := golog.LevelFromString("INFO")
	golog.SetAllLoggers(level)

	backend := logging.NewLogBackend(os.Stdout, "", 0)
	errBackend := logging.NewLogBackend(os.Stderr, "", 0)
	backendFormatter := logging.NewBackendFormatter(backend, stdoutLogFormat)
	errBackendFormatter := logging.NewBackendFormatter(errBackend, stdoutLogFormat)
	backendLeveled := logging.AddModuleLevel(errBackendFormatter)
	errBackendLeveled := logging.AddModuleLevel(backend)
	backendLeveled.SetLevel(logging.INFO, "")
	errBackendLeveled.SetLevel(logging.ERROR, "")

	logging.SetBackend(backendLeveled, backendFormatter)
	logging.SetBackend(errBackendLeveled, errBackendFormatter)

	ind := -1
	env, found := os.LookupEnv("GN_IND")
	if found {
		i, err := strconv.ParseInt(env, 10, 32)
		if err != nil {
			log.Debug("No GN_IND is found")
		} else {
			ind = int(i)
		}
	} else {
		// Parse options from the command line
		ind = *flag.Int("l", -1, "peer index")
		flag.Parse()
		if ind == -1 {
			log.Fatal("Please provide peer index with -l")
		}
	}

	extAddr, found := os.LookupEnv("EXT_ADDR")
	var extMA multiaddr.Multiaddr
	if found {
		extMA, _ = multiaddr.NewMultiaddr(extAddr)
	}

	plugPath, found := os.LookupEnv("GN_PLUGIN")
	if found {
		if plug, err := plugin.Open(plugPath); err == nil {
			if lookup, err := plug.Lookup("RollupsPlugin"); err == nil {
				onBlockCommit := lookup.(api.OnBlockCommit)
				onBlockCommit.OnBlockCommit(nil, nil, nil)
			} else {
				log.Error("RollupsPlugin export is not found")
			}
		} else {
			log.Error("Can't open plugin", err)
		}
	}

	s := cmn.ReadSettings()

	index := strconv.Itoa(ind)
	var loader cmn.CommitteeLoader = &cmn.CommitteeLoaderImpl{}
	peersPath := filepath.Join(s.Static.Dir, "peers.json")
	peerPath := path.Join(s.Static.Dir, "peer"+index+".json")
	committee := loader.LoadPeerListFromFile(peersPath)
	peerKey, err := loader.LoadPeerFromFile(peerPath, committee[ind])

	if err != nil {
		log.Fatal("Could't load peer credentials")
	}

	// Next we'll create the node config
	cfg := &network.NodeConfig{
		PrivateKey:        peerKey,
		Port:              9080,
		DataDir:           path.Join(s.Storage.Dir, strconv.Itoa(ind)),
		ExternalMultiaddr: extMA,
		Committee:         committee[0:s.Hotstuff.N],
	}

	ctx := CreateContext(cfg, committee[0:s.Hotstuff.N], committee[ind], s)

	// Ok now we can bootstrap the node. This could take a little bit if we're
	// running on a live network.
	ctx.Bootstrap(s)

	select {}

}
