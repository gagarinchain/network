package tx

import (
	"context"
	cmn "github.com/gagarinchain/common"
	"github.com/gagarinchain/common/api"
	"github.com/gagarinchain/common/eth/common"
	"github.com/gagarinchain/common/tx"
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

func TestTransactionPoolImpl_DrainDifferentNonceAndFee(t *testing.T) {
	pool := NewTransactionPool()
	addresses := cmn.GenerateAddresses(4)

	txs := []api.Transaction{
		tx.CreateTransaction(api.Payment, addresses[0], addresses[1], 0, big.NewInt(10), big.NewInt(1), nil),
		tx.CreateTransaction(api.Payment, addresses[0], addresses[2], 1, big.NewInt(20), big.NewInt(2), nil),
		tx.CreateTransaction(api.Payment, addresses[0], addresses[2], 2, big.NewInt(30), big.NewInt(2), nil),
		tx.CreateTransaction(api.Payment, addresses[1], addresses[2], 1, big.NewInt(30), big.NewInt(3), nil),
		tx.CreateTransaction(api.Payment, addresses[2], addresses[2], 1, big.NewInt(20), big.NewInt(5), nil),
		tx.CreateTransaction(api.Payment, addresses[2], addresses[2], 2, big.NewInt(30), big.NewInt(6), nil),
		tx.CreateTransaction(api.Payment, addresses[1], addresses[2], 1, big.NewInt(30), big.NewInt(7), nil),
	}
	go func() {
		pool.Add(txs[0])
		pool.Add(txs[1])
		pool.Add(txs[2])
		pool.Add(txs[3])

		time.Sleep(250 * time.Millisecond)

		pool.Add(txs[4])
		pool.Add(txs[5])
		pool.Add(txs[6])
	}()

	background := context.Background()
	timeout, _ := context.WithTimeout(background, time.Second)

	time.Sleep(20 * time.Millisecond)
	chunks := pool.Drain(timeout)

	var txChunked [][]api.Transaction
	for chunk := range chunks {
		txChunked = append(txChunked, chunk)
	}

	assert.Equal(t, 4, len(txChunked[0]))
	assert.Equal(t, 0, len(txChunked[1]))
	assert.Equal(t, 0, len(txChunked[2]))
	assert.Equal(t, 3, len(txChunked[3]))

	chunked := append(txChunked[0], txChunked[3]...)

	assert.Equal(t, chunked[0], txs[3])
	assert.Equal(t, chunked[1], txs[2])
	assert.Equal(t, chunked[2], txs[1])
	assert.Equal(t, chunked[3], txs[0])
	assert.Equal(t, chunked[4], txs[6])
	assert.Equal(t, chunked[5], txs[5])
	assert.Equal(t, chunked[6], txs[4])
}
