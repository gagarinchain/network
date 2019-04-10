package common

import (
	"crypto/ecdsa"
	"github.com/libp2p/go-libp2p-peerstore"
	"github.com/poslibp2p/common/eth/common"
	"github.com/poslibp2p/common/eth/crypto"
)

type Peer struct {
	address    common.Address
	publicKey  *ecdsa.PublicKey
	privateKey *ecdsa.PrivateKey
	peerInfo   *peerstore.PeerInfo
}

func (p *Peer) SetPrivateKey(privateKey *ecdsa.PrivateKey) {
	p.privateKey = privateKey
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

func (p *Peer) GetPeerInfo() *peerstore.PeerInfo {
	return p.peerInfo
}
