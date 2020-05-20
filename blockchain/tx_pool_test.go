package blockchain

import (
	"context"
	"github.com/gagarinchain/network/blockchain/tx"
	"github.com/gagarinchain/network/common/api"
	"github.com/gagarinchain/network/common/eth/common"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
	"time"
)

func TestTransactionPoolImpl_Drain(t *testing.T) {
	pool := NewTransactionPool()

	go func() {
		pool.Add(tx.CreateTransaction(api.Payment, common.Address{}, common.Address{}, 0, big.NewInt(0), big.NewInt(90), nil))
		pool.Add(tx.CreateTransaction(api.Payment, common.Address{}, common.Address{}, 0, big.NewInt(0), big.NewInt(20), nil))
		pool.Add(tx.CreateTransaction(api.Payment, common.Address{}, common.Address{}, 0, big.NewInt(0), big.NewInt(30), nil))
		pool.Add(tx.CreateTransaction(api.Payment, common.Address{}, common.Address{}, 0, big.NewInt(0), big.NewInt(10), nil))
		pool.Add(tx.CreateTransaction(api.Payment, common.Address{}, common.Address{}, 0, big.NewInt(0), big.NewInt(0), nil))
		pool.Add(tx.CreateTransaction(api.Payment, common.Address{}, common.Address{}, 0, big.NewInt(0), big.NewInt(50), nil))
		pool.Add(tx.CreateTransaction(api.Payment, common.Address{}, common.Address{}, 0, big.NewInt(0), big.NewInt(20), nil))

		time.Sleep(250 * time.Millisecond)

		pool.Add(tx.CreateTransaction(api.Payment, common.Address{}, common.Address{}, 0, big.NewInt(0), big.NewInt(20), nil))
		pool.Add(tx.CreateTransaction(api.Payment, common.Address{}, common.Address{}, 0, big.NewInt(0), big.NewInt(30), nil))
		pool.Add(tx.CreateTransaction(api.Payment, common.Address{}, common.Address{}, 0, big.NewInt(0), big.NewInt(10), nil))

		time.Sleep(300 * time.Millisecond)

		pool.Add(tx.CreateTransaction(api.Payment, common.Address{}, common.Address{}, 0, big.NewInt(0), big.NewInt(40), nil))
		pool.Add(tx.CreateTransaction(api.Payment, common.Address{}, common.Address{}, 0, big.NewInt(0), big.NewInt(40), nil))
		pool.Add(tx.CreateTransaction(api.Payment, common.Address{}, common.Address{}, 0, big.NewInt(0), big.NewInt(10), nil))
		pool.Add(tx.CreateTransaction(api.Payment, common.Address{}, common.Address{}, 0, big.NewInt(0), big.NewInt(20), nil))
		pool.Add(tx.CreateTransaction(api.Payment, common.Address{}, common.Address{}, 0, big.NewInt(0), big.NewInt(30), nil))
	}()

	background := context.Background()
	timeout, _ := context.WithTimeout(background, time.Second)

	time.Sleep(20 * time.Millisecond)
	chunks := pool.Drain(timeout)

	var txChunked [][]api.Transaction
	for chunk := range chunks {
		txChunked = append(txChunked, chunk)
	}

	assert.Equal(t, 7, len(txChunked[0]))
	assert.Equal(t, 0, len(txChunked[1]))
	assert.Equal(t, 0, len(txChunked[2]))
	assert.Equal(t, 3, len(txChunked[3]))
	assert.Equal(t, 0, len(txChunked[4]))
	assert.Equal(t, 0, len(txChunked[5]))
	assert.Equal(t, 5, len(txChunked[6]))
	assert.Equal(t, 0, len(txChunked[7]))
	assert.Equal(t, 0, len(txChunked[8]))

	assert.Equal(t, big.NewInt(90), txChunked[0][0].Fee())
	assert.Equal(t, big.NewInt(50), txChunked[0][1].Fee())
	assert.Equal(t, big.NewInt(30), txChunked[3][0].Fee())
	assert.Equal(t, big.NewInt(20), txChunked[3][1].Fee())
	assert.Equal(t, big.NewInt(40), txChunked[6][0].Fee())

}
