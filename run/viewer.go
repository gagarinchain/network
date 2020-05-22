package main

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/gagarinchain/common"
	"github.com/gagarinchain/network/blockchain"
	"os"
	"path"
	"strconv"
)

func main3() {
	storage, _ := common.NewStorage(path.Join(os.TempDir(), strconv.Itoa(3)), nil)
	persister := blockchain.BlockchainPersister{Storage: storage}
	for i := 0; i < 60; i++ {
		hashes, _ := persister.GetHeightIndexRecord(int32(i))

		spew.Dump(i, hashes)

	}
}
