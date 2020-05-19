package tx

import (
	"github.com/gagarinchain/network/common/eth/common"
	"math/big"
	"testing"
)

func TestTxHashing(t *testing.T) {
	tx := CreateTransaction(Payment, common.RandomAddress(), common.RandomAddress(), 1,
		big.NewInt(10), big.NewInt(1), []byte("ooops"))

	Hash(tx)
}
