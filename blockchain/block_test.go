package blockchain

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/gagarinchain/common/eth/common"
	"github.com/gagarinchain/common/eth/crypto"
	"github.com/prysmaticlabs/go-ssz"
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

func TestCreateBlockWithData(t *testing.T) {
	rollup := common.Hex2Bytes("080000005800000041e46d84c206007982f93c0874152b8cf6127985dd9811cfc24ab8d56036a8eca90c7b8c75e35950f3aa514423ae2c6f66497d69f2fc899f0ad25b0095ac9774cf32adb66f76726502fc0c223bce13e2010000000200000005000000000000000100000003000000040000000000000000000000010000000d000000000000000200000000000000080000000000000000000000020000001000000000000000")

	hash := common.BytesToHash(crypto.Keccak256([]byte("Quid")))
	header := createHeader(3,
		hash,
		crypto.Keccak256Hash([]byte("QcHash")),
		crypto.Keccak256Hash([]byte("TxHash")),
		*new([32]byte),
		crypto.Keccak256Hash(rollup),
		crypto.Keccak256Hash([]byte("Parent")),
		time.Date(2020, time.October, 3, 12, 0, 0, 0, time.Local))

	hash = HashHeader(header)

	type SendableHeader struct {
		Height    uint32
		Hash      [32]byte
		TxHash    [32]byte
		StateHash [32]byte
		DataHash  [32]byte
		QcHash    [32]byte
		Parent    [32]byte
		Timestamp uint64
	}

	h := &SendableHeader{
		Height:    uint32(header.Height()),
		Hash:      hash,
		TxHash:    header.TxHash(),
		StateHash: header.StateHash(),
		DataHash:  header.DataHash(),
		QcHash:    header.QCHash(),
		Parent:    header.Parent(),
		Timestamp: uint64(header.Timestamp().Unix()),
	}

	mh, _ := ssz.Marshal(h)

	s := fmt.Sprintf("%x", mh)
	println(s)

	//0x010000009e3d0e49056d998b5007b06157f36de98c4ad14cc68dbb7ba553b8a45ea9ae310000000000000000000000000000000000000000000000000000000000000000a82e3d7e96d07730f747b7bd4c13b7bba33b9f91a1db6dc99ec142941e6f9a3fc6e9bc3e163dd8e600563396ed32279d729b1fb4a96345972e5fcb80d782be5ba55a81100e8391212d2f22fc6b84b36b5d02588c2a96c4db616268d8abe389b8903d785f000000000000000000000000000000000000000000000000000000000000000000000000 bytes
	//0x010000009E3D0E49056D998B5007B06157F36DE98C4AD14CC68DBB7BA553B8A45EA9AE310000000000000000000000000000000000000000000000000000000000000000A82E3D7E96D07730F747B7BD4C13B7BBA33B9F91A1DB6DC99EC142941E6F9A3FC6E9BC3E163DD8E600563396ED32279D729B1FB4A96345972E5FCB80D782BE5BA55A81100E8391212D2F22FC6B84B36B5D02588C2A96C4DB616268D8ABE389B8903D785F00000000
	spew.Dump(h)
}
