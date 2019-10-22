package test

import (
	"bytes"
	"context"
	"github.com/davecgh/go-spew/spew"
	"github.com/gagarinchain/network/common/eth/common"
	"github.com/gagarinchain/network/common/message"
	pb "github.com/gagarinchain/network/common/protobuff"
	protoio "github.com/gogo/protobuf/io"

	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	txMessage = []byte{
		8, 7, 26, 220, 1, 10, 31, 116, 121, 112, 101, 46, 103, 111, 111, 103, 108, 101, 97, 112, 105, 115, 46, 99, 111, 109, 47, 84, 114, 97, 110, 115, 97, 99, 116, 105, 111, 110, 18, 184, 1, 18, 20, 166, 12, 133, 190, 156, 43, 137, 128, 219, 211, 248, 44, 248, 206, 25, 222, 18, 243, 232, 41, 24, 1, 32, 1, 40, 1, 50, 148, 1, 10, 48, 141, 0, 215, 206, 168, 241, 246, 237, 201, 97, 254, 40, 103, 255, 185, 74, 30, 120, 239, 139, 142, 180, 22, 214, 245, 183, 179, 246, 221, 152, 17, 207, 194, 74, 184, 213, 96, 54, 168, 236, 169, 12, 123, 140, 117, 227, 89, 80, 18, 96, 136, 23, 15, 230, 117, 208, 101, 155, 12, 60, 104, 69, 249, 198, 201, 177, 108, 66, 181, 113, 238, 109, 156, 105, 233, 218, 32, 173, 206, 239, 55, 123, 75, 71, 42, 68, 68, 41, 63, 203, 45, 134, 129, 168, 103, 44, 53, 118, 2, 148, 53, 118, 119, 212, 113, 102, 102, 50, 6, 156, 189, 139, 66, 3, 210, 112, 174, 247, 160, 166, 99, 119, 15, 86, 55, 133, 222, 104, 139, 74, 41, 151, 216, 104, 49, 79, 111, 53, 71, 55, 178, 69, 13, 194, 182, 239, 58, 3, 113, 119, 101,
	}

	someMessage = []byte{
		105, 8, 7, 26, 101, 10, 41, 116, 121, 112, 101, 46, 103, 111, 111, 103, 108, 101, 97, 112, 105, 115, 46, 99, 111, 109, 47, 65, 99, 99, 111, 117, 110, 116, 82, 101, 113, 117, 101, 115, 116, 80, 97, 121, 108, 111, 97, 100, 18, 56, 10, 32, 223, 154, 107, 145, 195, 164, 138, 184, 185, 9, 23, 5, 155, 20, 152, 110, 32, 99, 62, 255, 7, 5, 53, 7, 142, 48, 12, 158, 252, 177, 182, 147, 18, 20, 221, 152, 17, 207, 194, 74, 184, 213, 96, 54, 168, 236, 169, 12, 123, 140, 117, 227, 89, 80,
	}

	someHex = "0000008c41bbf320390ab8ae194eec442d0fce63668d0f7017542a1381b363375d8bb9a7e6b3955dce0b9954526b4eba1e1cb3eab5023b4e7a0404b9856bae12eb112b2a99d09a1045d6412f61da34ff962811e479973a537c4ef88459404ea88babd8da9d77c902c9fc0a187ae39564e97ae148baf0c249eba91cab3b40a29476d33877baacb28138d8078fcc4cd806"
)

func TestSomeMessageParse(t *testing.T) {
	m := &pb.Message{}

	bb := common.Hex2Bytes(someHex)
	reader := bytes.NewReader(bb)
	spew.Dump(bb)
	spew.Dump(someMessage)
	dr := protoio.NewDelimitedReader(reader, 1024)

	if err := dr.ReadMsg(m); err != nil {
		t.Error(err)
	}
	spew.Dump(m)
}
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
		if len(tx) > 0 {
			assert.Equal(t, common.HexToAddress("0xDd9811Cfc24aB8d56036A8ecA90C7B8C75e35950"), tx[0].From())
			break
		}
	}
	cancel()
}
