package tx

import (
	"bytes"
	"github.com/gagarinchain/common/api"
	"github.com/gagarinchain/common/eth/common"
	pb "github.com/gagarinchain/common/protobuff"
	protoio "github.com/gagarinchain/common/protobuff/io"
	"github.com/golang/protobuf/ptypes"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

func TestTxHashingForPayment(t *testing.T) {
	tx := CreateTransaction(api.Payment, common.HexToAddress("0xA01781411F964AC236bD05f96d45A02713d178d4"),
		common.HexToAddress("0x992AE4f167dD3Bf65c3c52b4149431cAA6b6c142"), 10,
		big.NewInt(20), big.NewInt(30), []byte("sdasdsdasdwftvxcasCQEWFQREF"))
	hash := Hash(tx)

	assert.Equal(t, "0x1bae6c57899e4016e58415ebe212fabd2b0211d1ad27e35574a84ce37dc1f07f", hash.Hex())
}
func TestTxHashingForSettlement(t *testing.T) {
	tx := CreateTransaction(api.Settlement, common.HexToAddress("0xA01781411F964AC236bD05f96d45A02713d178d4"),
		common.HexToAddress("0x992AE4f167dD3Bf65c3c52b4149431cAA6b6c142"), 10,
		big.NewInt(20), big.NewInt(30), []byte("sdasdsdasdwftvxcasCQEWFQREF"))
	hash := Hash(tx)

	assert.Equal(t, "0x176715c494706e2e7918f7a45b912b88cebbfcaf53ba586498480e548accb8fb", hash.Hex())
}

func TestCreateTransaction(t *testing.T) {
	hex := "fa0108071af5010a34747970652e676f6f676c65617069732e636f6d2f6761676172696e2e6e6574776f726b2e636f72652e5472616e73616374696f6e12bc01080212149cf72c59bb3624b7d2a9d82059b2f3832fd9973d1803200a28013294010a308d00d7cea8f1f6edc961fe2867ffb94a1e78ef8b8eb416d6f5b7b3f6dd9811cfc24ab8d56036a8eca90c7b8c75e359501260ad56628ff138a0e3b4d4bf7b86c109af30632aa2116bfce4645664c25b704a79f5ec9c447e2f7a43e1578493da76dea41709ef7e53714eb5a120080313b304b4632741beed53dce89793b29a73f5a97ae87c27d1445a95932f7c476ec7936b683a037177654200"
	b := common.Hex2Bytes(hex)

	pbm := &pb.Message{}

	reader := bytes.NewReader(b)
	dr := protoio.NewDelimitedReader(reader, 101024)

	if err := dr.ReadMsg(pbm); err != nil {
		t.Error(err)
	}

	pbt := &pb.Transaction{}
	if err := ptypes.UnmarshalAny(pbm.Payload, pbt); err != nil {
		t.Error(err)
		return
	}

	_, err := CreateTransactionFromMessage(pbt, false)

	assert.NoError(t, err)
}
