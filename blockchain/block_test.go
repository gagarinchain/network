package blockchain

import (
	"github.com/gagarinchain/common/eth/common"
	"github.com/gagarinchain/common/eth/crypto"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestHeaderHash(t *testing.T) {
	hash := common.BytesToHash(crypto.Keccak256([]byte("Quid")))
	header := createHeader(1,
		hash,
		common.BytesToHash(crypto.Keccak256([]byte("latine"))),
		common.BytesToHash(crypto.Keccak256([]byte("dictum"))),
		common.BytesToHash(crypto.Keccak256([]byte("sit"))),
		common.BytesToHash(crypto.Keccak256([]byte("altum"))),
		common.BytesToHash(crypto.Keccak256([]byte("viditur"))),
		time.Now())

	calculated := HashHeader(header)

	assert.Equal(t, hash, header.Hash())
	assert.NotEqual(t, calculated, header.Hash())
}
