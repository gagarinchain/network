package blockchain

import (
	"github.com/gagarinchain/common/api"
	"github.com/gagarinchain/common/eth/common"
	"github.com/gagarinchain/common/eth/crypto"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
	"time"
)

func TestValidation(t *testing.T) {
	qchash := crypto.Keccak256([]byte("hello! how high?"))
	header := createHeader(1, common.Hash{}, common.BytesToHash(qchash),
		common.Hash{}, common.Hash{}, common.Hash{}, common.Hash{}, time.Now())
	header.SetHash()

	pubkeys := make([]*crypto.PublicKey, 0, 100)
	sigs := make([]*crypto.Signature, 0, 100)
	for i := 0; i < 100; i++ {
		priv, _ := crypto.GenerateKey()
		pub := priv.PublicKey()
		sig := crypto.Sign(header.hash.Bytes(), priv)
		pubkeys = append(pubkeys, pub)
		sigs = append(sigs, sig)
	}
	lsh := big.NewInt(0).Lsh(big.NewInt(1), 100)
	bitmap := lsh.Sub(lsh, big.NewInt(1))
	aggSig := crypto.AggregateSignatures(bitmap, sigs)

	certificate := CreateQuorumCertificate(aggSig, header, api.QRef)
	b, e := certificate.IsValid(certificate.GetHash(), pubkeys)

	assert.True(t, b)
	assert.NoError(t, e)
}
func TestEmptyIsUsedForNonEmptyBlockValidation(t *testing.T) {
	qchash := crypto.Keccak256([]byte("hello! how high?"))
	header := createHeader(1, common.Hash{}, common.BytesToHash(qchash),
		common.Hash{}, common.Hash{}, common.Hash{}, common.Hash{}, time.Now())
	header.SetHash()

	pubkeys := make([]*crypto.PublicKey, 0, 100)
	sigs := make([]*crypto.Signature, 0, 100)
	for i := 0; i < 100; i++ {
		priv, _ := crypto.GenerateKey()
		pub := priv.PublicKey()
		sig := crypto.Sign(CalculateSyncHash(header.height, true).Bytes(), priv)
		pubkeys = append(pubkeys, pub)
		sigs = append(sigs, sig)
	}
	lsh := big.NewInt(0).Lsh(big.NewInt(1), 100)
	bitmap := lsh.Sub(lsh, big.NewInt(1))
	aggSig := crypto.AggregateSignatures(bitmap, sigs)

	certificate := CreateQuorumCertificate(aggSig, header, api.Empty)
	b, e := certificate.IsValid(certificate.GetHash(), pubkeys)

	assert.False(t, b)
	assert.Errorf(t, e, "empty qc is used for non empty block")
}

func TestEmptyIsUsedForEmptyBlockValidation(t *testing.T) {
	qchash := crypto.Keccak256([]byte("hello! how high?"))
	header := createHeader(1, common.Hash{}, common.BytesToHash(qchash),
		common.HexToHash(api.EmptyTxHashHex), common.Hash{}, common.Hash{}, common.Hash{}, time.Now())
	header.SetHash()

	pubkeys := make([]*crypto.PublicKey, 0, 100)
	sigs := make([]*crypto.Signature, 0, 100)
	for i := 0; i < 100; i++ {
		priv, _ := crypto.GenerateKey()
		pub := priv.PublicKey()
		sig := crypto.Sign(CalculateSyncHash(header.height, true).Bytes(), priv)
		pubkeys = append(pubkeys, pub)
		sigs = append(sigs, sig)
	}
	lsh := big.NewInt(0).Lsh(big.NewInt(1), 100)
	bitmap := lsh.Sub(lsh, big.NewInt(1))
	aggSig := crypto.AggregateSignatures(bitmap, sigs)

	certificate := CreateQuorumCertificate(aggSig, header, api.Empty)
	b, e := certificate.IsValid(certificate.GetHash(), pubkeys)

	assert.True(t, b)
	assert.NoError(t, e)
}
