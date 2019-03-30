package message

import (
	"encoding/json"
	"github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/poslibp2p/eth/common"
	"io/ioutil"
	"os"
)

type CommitteeLoader interface {
	LoadFromFile(filePath string) []*Peer
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

func (c *CommitteeLoaderImpl) LoadFromFile(filePath string) (res []*Peer) {
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

		info, e := peerstore.InfoFromP2pAddr(addr)
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
