package common

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	p2pcrypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/libp2p/go-libp2p-peer"
	"github.com/op/go-logging"
	"github.com/poslibp2p/common/eth/crypto"
	"io/ioutil"
	"strconv"
)

var log = logging.MustGetLogger("hotstuff")

func GenerateIdentities() {
	committee := CommitteeData{}

	for i := 0; i < 10; i++ {
		pk, _ := crypto.GenerateKey()
		pkbytes := crypto.FromECDSA(pk)
		pkstring := hex.EncodeToString(pkbytes)
		address := crypto.PubkeyToAddress(*pk.Public().(*ecdsa.PublicKey)).Hex()

		privKey, _, _ := p2pcrypto.GenerateECDSAKeyPair(rand.Reader)
		id, _ := peer.IDFromPrivateKey(privKey)
		b, _ := p2pcrypto.MarshalPrivateKey(privKey)

		v := map[string]interface{}{"addr": address, "pk": pkstring, "id": id.Pretty(), "pkpeer": hex.EncodeToString(b)}

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
			Address:      address,
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