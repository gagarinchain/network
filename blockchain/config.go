package blockchain

import (
	"encoding/json"
	"github.com/gagarinchain/network/blockchain/state"
	"io/ioutil"
	"math/big"
	"os"
)
import "github.com/gagarinchain/network/common/eth/common"

type BlockchainConfig struct {
	Seed           map[common.Address]*state.Account
	BlockPerister  *BlockPersister
	ChainPersister *BlockchainPersister
	BlockService   BlockService
	Pool           TransactionPool
	Db             state.DB
}

type SeedData struct {
	Accounts []*AccountData `json:"accounts"`
}

type AccountData struct {
	Address string `json:"address"`
	Balance int64  `json:"balance"`
	Nonce   uint64 `json:"nonce"`
}

func SeedFromFile(filePath string) map[common.Address]*state.Account {
	res := make(map[common.Address]*state.Account)
	file, e := os.Open(filePath)
	if e != nil {
		log.Fatal("Can't load seed", e)
		return nil
	}
	defer file.Close()

	byteValue, _ := ioutil.ReadAll(file)

	var data SeedData
	if err := json.Unmarshal(byteValue, &data); err != nil {
		log.Fatal("Can't unmarshal seed file", err)
		return nil
	}

	for _, a := range data.Accounts {
		address := common.HexToAddress(a.Address)
		balance := big.NewInt(a.Balance)
		acc := state.NewAccount(a.Nonce, balance)
		res[address] = acc
	}

	return res

}
