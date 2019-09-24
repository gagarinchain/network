package crypto

import (
	"github.com/gagarinchain/network/common/eth/crypto/bls12_381"
	pb "github.com/gagarinchain/network/common/protobuff"
	"github.com/op/go-logging"
	"github.com/phoreproject/bls/g1pubs"
	"math/big"
)

var log = logging.MustGetLogger("crypto")

type PublicKey struct {
	v *g1pubs.PublicKey
}

func (key *PublicKey) V() *g1pubs.PublicKey {
	return key.v
}

func NewPublicKey(v *g1pubs.PublicKey) *PublicKey {
	return &PublicKey{v: v}
}

func (key *PublicKey) Bytes() []byte {
	b := key.v.Serialize()
	return b[:]
}

type PrivateKey struct {
	v *g1pubs.SecretKey
}

func (pk *PrivateKey) V() *g1pubs.SecretKey {
	return pk.v
}

func NewPrivateKey(v *g1pubs.SecretKey) *PrivateKey {
	return &PrivateKey{v: v}
}

func (pk *PrivateKey) PublicKey() *PublicKey {
	return &PublicKey{g1pubs.PrivToPub(pk.v)}
}

type Signature struct {
	pub  *g1pubs.PublicKey
	sign *g1pubs.Signature
}

func (s *Signature) IsEmpty() bool {
	return g1pubs.NewAggregatePubkey().Equals(*s.Pub())
}

func (s *Signature) Sign() *g1pubs.Signature {
	return s.sign
}

func (s *Signature) Pub() *g1pubs.PublicKey {
	return s.pub
}

func (s *Signature) ToProto() *pb.Signature {
	pkBytes := s.Pub().Serialize()
	signBytes := s.Sign().Serialize()
	return &pb.Signature{
		From:      pkBytes[:],
		Signature: signBytes[:],
	}
}

func (s *Signature) ToStorageProto() *pb.Sign {
	pkBytes := s.Pub().Serialize()
	signBytes := s.Sign().Serialize()
	return &pb.Sign{
		From:      pkBytes[:],
		Signature: signBytes[:],
	}
}

func SignatureFromProto(mes *pb.Signature) *Signature {
	return NewSignatureFromBytes(mes.From, mes.Signature)
}
func SignatureFromStorage(mes *pb.Sign) *Signature {
	return NewSignatureFromBytes(mes.From, mes.Signature)
}

func NewSignature(pk *g1pubs.PublicKey, sign *g1pubs.Signature) *Signature {
	return &Signature{pub: pk, sign: sign}
}

func NewSignatureFromBytes(pk []byte, sign []byte) *Signature {
	pkBytes := bls12_381.ToBytes48(pk)
	key, e := g1pubs.DeserializePublicKey(pkBytes)
	if e != nil {
		log.Error(e)
		return nil
	}
	signBytes := bls12_381.ToBytes96(sign)
	signature, e := g1pubs.DeserializeSignature(signBytes)
	if e != nil {
		log.Error(e)
		return nil
	}

	return NewSignature(key, signature)
}

type SignatureAggregate struct {
	bitmap    *big.Int
	n         int
	aggregate *g1pubs.Signature
}

func (sa *SignatureAggregate) Aggregate() *g1pubs.Signature {
	return sa.aggregate
}

func (sa *SignatureAggregate) N() int {
	return sa.n
}

func (sa *SignatureAggregate) Bitmap() *big.Int {
	return sa.bitmap
}

func AggregateFromProto(mes *pb.SignatureAggregate) *SignatureAggregate {
	return NewAggregateFromBytes(mes.Bitmap, mes.N, mes.Signature)
}

func EmptySignature() *Signature {
	return &Signature{
		pub:  g1pubs.NewAggregatePubkey(),
		sign: g1pubs.NewAggregateSignature(),
	}
}

func EmptyAggregateSignatures() *SignatureAggregate {
	return &SignatureAggregate{
		bitmap:    big.NewInt(0),
		n:         0,
		aggregate: g1pubs.NewAggregateSignature(),
	}
}

func NewAggregateFromBytes(bitmap []byte, n int32, sign []byte) *SignatureAggregate {
	b := big.NewInt(0).SetBytes(bitmap)
	nn := int(n)

	signBytes := bls12_381.ToBytes96(sign)
	s, e := g1pubs.DeserializeSignature(signBytes)
	if e != nil {
		log.Error(e)
		return nil
	}

	return &SignatureAggregate{
		bitmap:    b,
		n:         nn,
		aggregate: s,
	}
}

func (sa *SignatureAggregate) ToProto() *pb.SignatureAggregate {
	sign := sa.aggregate.Serialize()
	return &pb.SignatureAggregate{
		Bitmap:    sa.bitmap.Bytes(),
		N:         int32(sa.n),
		Signature: sign[:],
	}

}
