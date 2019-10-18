package blockchain

import (
	"encoding/json"
	"github.com/gagarinchain/network"
	"github.com/gagarinchain/network/blockchain/state"
	"io/ioutil"
	"math/big"
	"os"
	"time"
)
import "github.com/gagarinchain/network/common/eth/common"
import cmn "github.com/gagarinchain/network/common"

type BlockchainConfig struct {
	Seed           map[common.Address]*state.Account
	BlockPerister  *BlockPersister
	ProposerGetter cmn.ProposerForHeight
	ChainPersister *BlockchainPersister
	BlockService   BlockService
	Pool           TransactionPool
	Db             state.DB
	Storage        gagarinchain.Storage
	Delta          time.Duration
	EventBus       cmn.EventBus
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
