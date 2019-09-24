package test

import (
	"crypto/rand"
	"encoding/binary"
	"github.com/davecgh/go-spew/spew"
	"github.com/phoreproject/bls/g1pubs"
	"github.com/pkg/errors"
	"io"
	"math/big"
	"testing"
)

type SignatureAggregate struct {
	bitmap    *big.Int
	n         int32
	aggregate *g1pubs.Signature
}

func TestBlsSignatureVerify(t *testing.T) {
	priv, _ := RandKey(rand.Reader)
	pub := g1pubs.PrivToPub(priv)
	msg := []byte("hello")
	sig := Sign(msg, priv, 0)
	if !Verify(msg, sig, pub, 0) {
		t.Error("Signature did not verify")
	}
}

func TestVerifyAggregate(t *testing.T) {
	pubkeys := make([]*g1pubs.PublicKey, 0, 100)
	sigs := make([]*g1pubs.Signature, 0, 100)
	msg := []byte("guldaaaaan")
	for i := 0; i < 100; i++ {
		priv, _ := g1pubs.RandKey(rand.Reader)
		pub := g1pubs.PrivToPub(priv)
		sig := Sign(msg, priv, 0)
		pubkeys = append(pubkeys, pub)
		sigs = append(sigs, sig)
	}
	aggSig := AggregateSignatures(sigs)
	spew.Dump(sigs[0].Serialize())
	spew.Dump(aggSig.Serialize())
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, 0)
	if !aggSig.VerifyAggregateCommonWithDomain(pubkeys, ToBytes32(msg), ToBytes8(b)) {
		t.Error("Signature did not verify")
	}

	spew.Dump(pubkeys[0].Serialize())
}

func Sign(msg []byte, s *g1pubs.SecretKey, domain uint64) *g1pubs.Signature {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, domain)
	sig := g1pubs.SignWithDomain(ToBytes32(msg), s, ToBytes8(b))
	return sig
}

func Verify(msg []byte, s *g1pubs.Signature, pub *g1pubs.PublicKey, domain uint64) bool {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, domain)
	return g1pubs.VerifyWithDomain(ToBytes32(msg), pub, s, ToBytes8(b))
}
func ToBytes32(bytes []byte) (res [32]byte) {
	copy(res[:], bytes[:])
	return
}

func ToBytes8(bytes []byte) (res [8]byte) {
	copy(res[:], bytes[:])
	return
}

func RandKey(r io.Reader) (*g1pubs.SecretKey, error) {
	k, err := g1pubs.RandKey(r)
	if err != nil {
		return nil, errors.Wrap(err, "could not initialize secret key")
	}
	return k, nil
}

func AggregateSignatures(sigs []*g1pubs.Signature) *g1pubs.Signature {
	var ss []*g1pubs.Signature
	for _, v := range sigs {
		if v == nil {
			continue
		}
		ss = append(ss, v)
	}
	return g1pubs.AggregateSignatures(ss)
}
