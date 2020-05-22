package blockchain

import (
	"encoding/json"
	"github.com/gagarinchain/common/api"
	"github.com/gagarinchain/network"
	"github.com/gagarinchain/network/blockchain/state"
	"github.com/gagarinchain/network/blockchain/tx"
	"io/ioutil"
	"math/big"
	"os"
	"time"
)
import "github.com/gagarinchain/common/eth/common"
import cmn "github.com/gagarinchain/common"

type BlockchainConfig struct {
	Seed           map[common.Address]api.Account
	BlockPerister  *BlockPersister
	ProposerGetter api.ProposerForHeight
	ChainPersister *BlockchainPersister
	Pool           tx.TransactionPool
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

func SeedFromFile(filePath string) map[common.Address]api.Account {
	res := make(map[common.Address]api.Account)
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
