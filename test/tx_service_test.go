package test

import (
	"context"
	"github.com/davecgh/go-spew/spew"
	"github.com/gagarinchain/network/common/eth/common"
	"github.com/gagarinchain/network/common/message"
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	txMessage = []byte{
		8, 7, 26, 220, 1, 10, 31, 116, 121, 112, 101, 46, 103, 111, 111, 103, 108, 101, 97, 112, 105, 115, 46, 99, 111, 109, 47, 84, 114, 97, 110, 115, 97, 99, 116, 105, 111, 110, 18, 184, 1, 18, 20, 166, 12, 133, 190, 156, 43, 137, 128, 219, 211, 248, 44, 248, 206, 25, 222, 18, 243, 232, 41, 24, 1, 32, 1, 40, 1, 50, 148, 1, 10, 48, 141, 0, 215, 206, 168, 241, 246, 237, 201, 97, 254, 40, 103, 255, 185, 74, 30, 120, 239, 139, 142, 180, 22, 214, 245, 183, 179, 246, 221, 152, 17, 207, 194, 74, 184, 213, 96, 54, 168, 236, 169, 12, 123, 140, 117, 227, 89, 80, 18, 96, 136, 23, 15, 230, 117, 208, 101, 155, 12, 60, 104, 69, 249, 198, 201, 177, 108, 66, 181, 113, 238, 109, 156, 105, 233, 218, 32, 173, 206, 239, 55, 123, 75, 71, 42, 68, 68, 41, 63, 203, 45, 134, 129, 168, 103, 44, 53, 118, 2, 148, 53, 118, 119, 212, 113, 102, 102, 50, 6, 156, 189, 139, 66, 3, 210, 112, 174, 247, 160, 166, 99, 119, 15, 86, 55, 133, 222, 104, 139, 74, 41, 151, 216, 104, 49, 79, 111, 53, 71, 55, 178, 69, 13, 194, 182, 239, 58, 3, 113, 119, 101,
	}
)

func TestTransactionReceive(t *testing.T) {

	ctx := initContext(t)
	m := message.CreateFromSerialized(txMessage, ctx.me)
	ch := make(chan *message.Message)
	background := context.Background()
	ctx2, cancel := context.WithCancel(background)

	go func() {
		ctx.txService.Run(ctx2, ch)
	}()
	go func() {
		ch <- m
	}()

	spew.Dump(m)

	chunks := ctx.pool.Drain(ctx2)
	for tx := range chunks {
		if tx != nil {
			assert.Equal(t, common.HexToAddress("0xDd9811Cfc24aB8d56036A8ecA90C7B8C75e35950"), tx[0].From())
			break
		}
	}
	cancel()
}
