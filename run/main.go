package main

import (
	"flag"
	"github.com/davecgh/go-spew/spew"
	cmn "github.com/gagarinchain/network/common"
	"github.com/gagarinchain/network/network"
	golog "github.com/ipfs/go-log"
	"github.com/op/go-logging"
	gologging "github.com/whyrusleeping/go-logging"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path"
	"strconv"
)

var stdoutLogFormat = logging.MustStringFormatter(
	`%{time:15:04:05.000} [%{shortfile}] [%{level}] %{message}`,
)

var log = logging.MustGetLogger("main")

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
		DataDir string `yaml:"DataDir"`
	} `yaml:"Storage"`
}

func main() {
	// LibP2P code uses golog to log messages. They log with different
	// string IDs (i.e. "swarm"). We can control the verbosity level for
	// all loggers with:
	golog.SetAllLoggers(gologging.INFO)

	backend := logging.NewLogBackend(os.Stderr, "", 0)
	backendFormatter := logging.NewBackendFormatter(backend, stdoutLogFormat)
	backendLeveled := logging.AddModuleLevel(backend)
	backendLeveled.SetLevel(logging.INFO, "")

	logging.SetBackend(backendLeveled, backendFormatter)

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

	s := readSettings()

	index := strconv.Itoa(ind)
	var loader cmn.CommitteeLoader = &cmn.CommitteeLoaderImpl{}
	committee := loader.LoadPeerListFromFile("static/peers.json")
	spew.Dump(committee)
	peerKey, err := loader.LoadPeerFromFile("static/peer"+index+".json", committee[ind])

	if err != nil {
		log.Fatal("Could't load peer credentials")
	}

	// Next we'll create the node config
	cfg := &network.NodeConfig{
		PrivateKey: peerKey,
		Port:       9080,
		DataDir:    path.Join(s.Storage.DataDir, strconv.Itoa(ind)),
		Committee:  committee[0:s.Hotstuff.N],
	}

	ctx := CreateContext(cfg, committee[0:s.Hotstuff.N], committee[ind], s)

	// Ok now we can bootstrap the node. This could take a little bit if we're
	// running on a live network.
	ctx.Bootstrap(s)

	select {}

}

func readSettings() (s *Settings) {
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
				DataDir string `yaml:"DataDir"`
			}{DataDir: os.TempDir()},
		}
	}
	return
}
