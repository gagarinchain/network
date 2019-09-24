package sparse

import (
	"github.com/gagarinchain/network/common/eth/common"
	"math/big"
)

type VersionedSMT struct {
	trees map[common.Hash]*SMT
}

func (smt *VersionedSMT) Put(version common.Hash, key big.Int, value []byte) {

}
