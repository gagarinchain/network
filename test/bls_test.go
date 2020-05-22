package test

import (
	"encoding/binary"
	"github.com/davecgh/go-spew/spew"
	"github.com/gagarinchain/common/eth/crypto"
	"github.com/phoreproject/bls/g1pubs"
	"math/big"
	"testing"
	"time"
)

func TestBlsSignatureVerify(t *testing.T) {
	priv, _ := crypto.GenerateKey()
	msg := []byte("hello")
	sig := crypto.Sign(msg, priv)
	if !crypto.Verify(msg, sig) {
		t.Error("Signature did not verify")
	}
}

func TestVerifyAggregate(t *testing.T) {
	pubkeys := make([]*crypto.PublicKey, 0, 100)
	g1pubkeys := make([]*g1pubs.PublicKey, 0, 100)
	sigs := make([]*crypto.Signature, 0, 100)
	msg := []byte("guldaaaaan")

	now := time.Now()
	for i := 0; i < 100; i++ {
		priv, _ := crypto.GenerateKey()
		pub := priv.PublicKey()
		sig := crypto.Sign(msg, priv)
		pubkeys = append(pubkeys, pub)
		g1pubkeys = append(g1pubkeys, pub.V())
		sigs = append(sigs, sig)
	}
	now2 := time.Now()
	spew.Dump(now2.Sub(now))

	lsh := big.NewInt(0).Lsh(big.NewInt(1), 100)
	bitmap := lsh.Sub(lsh, big.NewInt(1))
	aggSig := crypto.AggregateSignatures(bitmap, sigs)
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, 0)
	if !crypto.VerifyAggregate(msg, pubkeys, aggSig) {
		t.Error("Signature did not verify")
	}

	now3 := time.Now()
	spew.Dump(now3.Sub(now2))

}

func TestVerifyAggregateWithSpacesInCommittee(t *testing.T) {
	pubkeys := make([]*crypto.PublicKey, 0, 100)
	sigs := make([]*crypto.Signature, 0, 100)
	msg := []byte("guldaaaaan")

	bitmap := big.NewInt(0)

	for i := 0; i < 100; i++ {
		priv, _ := crypto.GenerateKey()
		pub := priv.PublicKey()
		pubkeys = append(pubkeys, pub)
		if i%3 == 0 {
			bitmap = bitmap.Lsh(bitmap, 1)
			continue
		}

		bitmap = bitmap.Lsh(bitmap, 1)
		bitmap.SetBit(bitmap, 0, 1)
		sig := crypto.Sign(msg, priv)
		sigs = append(sigs, sig)
	}
	spew.Dump(bitmap)
	aggSig := crypto.AggregateSignatures(bitmap, sigs)
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, 0)
	if !aggSig.IsValid(msg, pubkeys) {
		t.Error("Signature did not verify")
	}
}
