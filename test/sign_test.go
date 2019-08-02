package test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"github.com/gagarinchain/network/common/eth/crypto"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPubKeyRecovery(t *testing.T) {
	text := "This is top secret text"
	hash := crypto.Keccak256([]byte(text))
	pub, priv := generateKeyPair()

	sig, _ := crypto.Sign(hash, priv)

	recovered, err := crypto.SigToPub(hash, sig)
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, pub, recovered)

}

func TestSignature(t *testing.T) {
	text := "This is top secret text"
	hash := crypto.Keccak256([]byte(text))
	pub, priv := generateKeyPair()

	sig, _ := crypto.Sign(hash, priv)

	res := crypto.VerifySignature(crypto.CompressPubkey(pub), hash, sig[0:cap(sig)-1])

	assert.True(t, res)
}

func generateKeyPair() (pubkey *ecdsa.PublicKey, key *ecdsa.PrivateKey) {
	key, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	if err != nil {
		panic(err)
	}
	b := elliptic.Marshal(crypto.S256(), key.X, key.Y)
	pubkey, _ = crypto.UnmarshalPubkey(b)

	return pubkey, key
}
