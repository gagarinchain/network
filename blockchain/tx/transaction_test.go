package tx

import (
	"bytes"
	"github.com/gagarinchain/common/eth/common"
	pb "github.com/gagarinchain/common/protobuff"
	protoio "github.com/gogo/protobuf/io"
	"github.com/golang/protobuf/ptypes"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

func TestTxHashing(t *testing.T) {
	tx := CreateTransaction(2, common.HexToAddress("0xA01781411F964AC236bD05f96d45A02713d178d4"),
		common.HexToAddress("0x992AE4f167dD3Bf65c3c52b4149431cAA6b6c142"), 10,
		big.NewInt(20), big.NewInt(30), []byte("sdasdsdasdwftvxcasCQEWFQREF"))
	hash := Hash(tx)

	assert.Equal(t, hash.Hex(), "0xa6c19fdcbb6ea74e44be3d4b8405f6f5a62992f1a2f6327ce145832503267c17")
}

func TestCreateTransaction(t *testing.T) {
	hex := "f80108071af3010a34747970652e676f6f676c65617069732e636f6d2f6761676172696e2e6e6574776f726b2e636f72652e5472616e73616374696f6e12ba011214a60c85be9c2b8980dbd3f82cf8ce19de12f3e8291801200128013294010a308d00d7cea8f1f6edc961fe2867ffb94a1e78ef8b8eb416d6f5b7b3f6dd9811cfc24ab8d56036a8eca90c7b8c75e359501260b60ffb9dde0e00abb63f49a7805a6820220db0e4dd2e3394708aece7424ef65ba9ae77f33fd381a0947e6a8fb287c66d03f091ce62300fc2b0285a782c69ba6d246c84d99aceb62ff79a1cad17aa04934408f2ac0cb022d29bc330a43e47d10b3a037177654200"
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
