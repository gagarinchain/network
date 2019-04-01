package main

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	golog "github.com/ipfs/go-log"
	p2pcrypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/op/go-logging"
	"github.com/poslibp2p/message"
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
	ind := flag.Int("l", 0, "peer index")
	flag.Parse()

	if *ind == 0 {
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
	pk := v["pkpeer"].(string)
	decodeString, e := hex.DecodeString(pk)
	privKey, err := p2pcrypto.UnmarshalPrivateKey(decodeString)
	if err != nil {
		log.Fatal(err)
	}

	var loader message.CommitteeLoader = &message.CommitteeLoaderImpl{}
	committee := loader.LoadFromFile("static/peers.json")
	// Next we'll create the node config
	cfg := &network.NodeConfig{
		PrivateKey: privKey,
		Port:       8000 + uint16(*ind),
		DataDir:    path.Join(os.TempDir(), strconv.Itoa(*ind)),
		Committee:  committee,
	}

	ctx := CreateContext(cfg)

	// Ok now we can bootstrap the node. This could take a little bit if we're
	// running on a live network.
	ctx.Bootstrap()

	// Publish to the topic
	//var text = "Hello"
	//any, e := ptypes.MarshalAny(&pb.HelloPayload{HelloMessage: text})
	//if e != nil {
	//	log.Fatal(err)
	//}
	//
	//var msg = &pb.Message{Type: pb.Message_HELLO, Payload: any}
	//marshalled, e := proto.Marshal(msg)
	//if e != nil {
	//	log.Fatal(err)
	//}

	//log.Infof("Publishing message [%s] to topic..", marshalled)
	//
	//tick := func() {
	//	err = node.PubSub.Publish(context.Background(), "shard", marshalled)
	//	if err != nil {
	//		log.Fatal(err)
	//	}
	//}
	//
	//t := time.NewTicker(11 * time.Second)
	//go func() {
	//	for {
	//		select {
	//		case <-t.C:
	//			log.Debugf("Sending...")
	//			tick()
	//		}
	//	}
	//}()
	//tick()

	select {}

}
