package blockchain

import (
	"github.com/gagarinchain/network/common/eth/common"
	"github.com/gagarinchain/network/common/eth/crypto"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
	"time"
)

func TestValidation(t *testing.T) {
	msg := crypto.Keccak256([]byte("hello! how low?"))
	qchash := crypto.Keccak256([]byte("hello! how high?"))

	pubkeys := make([]*crypto.PublicKey, 0, 100)
	sigs := make([]*crypto.Signature, 0, 100)
	for i := 0; i < 100; i++ {
		priv, _ := crypto.GenerateKey()
		pub := priv.PublicKey()
		sig := crypto.Sign(msg, priv)
		pubkeys = append(pubkeys, pub)
		sigs = append(sigs, sig)
	}
	lsh := big.NewInt(0).Lsh(big.NewInt(1), 100)
	bitmap := lsh.Sub(lsh, big.NewInt(1))
	aggSig := crypto.AggregateSignatures(bitmap, sigs)

	certificate := CreateQuorumCertificate(aggSig, createHeader(1, common.BytesToHash(msg), common.BytesToHash(qchash),
		common.Hash{}, common.Hash{}, common.Hash{}, common.Hash{}, time.Now()))
	b, e := certificate.IsValid(certificate.GetHash(), pubkeys)

	assert.True(t, b)
	assert.NoError(t, e)

}
