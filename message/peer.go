package message

import (
	"crypto/ecdsa"
	"github.com/libp2p/go-libp2p-peerstore"
	"github.com/poslibp2p/eth/common"
	"github.com/poslibp2p/eth/crypto"
)

type Peer struct {
	address    common.Address
	publicKey  *ecdsa.PublicKey
	privateKey *ecdsa.PrivateKey
	peerInfo   *peerstore.PeerInfo
}

func (p *Peer) GetAddress() common.Address {
	return p.address
}

func (p *Peer) SetAddress(address common.Address) {
	p.address = address
}

func (p *Peer) GetPrivateKey() *ecdsa.PrivateKey {
	return p.privateKey
}

func CreatePeer(publicKey *ecdsa.PublicKey, privateKey *ecdsa.PrivateKey, peerInfo *peerstore.PeerInfo) *Peer {
	peer := &Peer{
		publicKey:  publicKey,
		privateKey: privateKey,
		peerInfo:   peerInfo,
	}

	peer.address = common.BytesToAddress(crypto.FromECDSAPub(publicKey))
	return peer
}

func (p *Peer) Equals(toCompare *Peer) bool {
	return p.address == toCompare.address
}
