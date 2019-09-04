package common

import (
	"encoding/hex"
	"encoding/json"
	"github.com/gagarinchain/network/common/eth/common"
	"github.com/gagarinchain/network/common/eth/crypto"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"io/ioutil"
	"os"
)

type CommitteeLoader interface {
	LoadPeerListFromFile(filePath string) []*Peer
	LoadPeerFromFile(fileName string, peer *Peer) (peerKey p2pcrypto.PrivKey, err error)
}

type CommitteeData struct {
	Peers []PeerData `json:"peers"`
}

type PeerData struct {
	Address      string `json:"addr"`
	MultiAddress string `json: ma"`
}

type CommitteeLoaderImpl struct {
}

func (c *CommitteeLoaderImpl) LoadPeerListFromFile(filePath string) (res []*Peer) {
	file, e := os.Open(filePath)
	if e != nil {
		log.Fatal("Can't load committee list", e)
		return nil
	}
	defer file.Close()

	byteValue, _ := ioutil.ReadAll(file)

	var data CommitteeData
	if err := json.Unmarshal(byteValue, &data); err != nil {
		log.Fatal("Can't unmarshal committee file", err)
		return nil
	}

	for _, v := range data.Peers {
		address := common.HexToAddress(v.Address)
		addr, e := ma.NewMultiaddr(v.MultiAddress)
		if e != nil {
			log.Fatal("Can't create address", e)
			return nil
		}

		info, e := peer.AddrInfoFromP2pAddr(addr)
		if e != nil {
			log.Fatal("Can't create address", e)
			return nil
		}
		res = append(res, &Peer{
			address:  address,
			peerInfo: info,
		})

	}

	return res
}

func (c *CommitteeLoaderImpl) LoadPeerFromFile(fileName string, peer *Peer) (peerKey p2pcrypto.PrivKey, err error) {
	var v map[string]interface{}

	bytes, e := ioutil.ReadFile(fileName)
	if e != nil {
		return nil, e
	}
	e = json.Unmarshal(bytes, &v)
	if e != nil {
		return nil, e
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
		return nil, err
	}

	pkbytes, e := hex.DecodeString(pk)
	if e != nil {
		return nil, e

	}
	key, e := crypto.ToECDSA(pkbytes)
	if e != nil {
		return nil, e

	}
	peer.SetAddress(common.HexToAddress(addr))
	peer.SetPrivateKey(key)

	return privKey, nil
}
