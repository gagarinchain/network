package bls12_381

import (
	"encoding/binary"
	"github.com/phoreproject/bls/g1pubs"
	"github.com/pkg/errors"
	"io"
)

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
func ToBytes48(bytes []byte) (res [48]byte) {
	copy(res[:], bytes[:])
	return
}
func ToBytes96(bytes []byte) (res [96]byte) {
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
