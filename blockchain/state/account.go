package state

import (
	"github.com/gagarinchain/network/common/eth/common"
	pb "github.com/gagarinchain/network/common/protobuff"
	"github.com/gogo/protobuf/proto"
	"math/big"
)

type Account struct {
	nonce   uint64
	balance *big.Int
	origin  common.Address
	voters  []common.Address
}

func (a *Account) Voters() []common.Address {
	return a.voters
}

func (a *Account) Balance() *big.Int {
	return a.balance
}

func (a *Account) Nonce() uint64 {
	return a.nonce
}

func NewAccount(nonce uint64, balance *big.Int) *Account {
	return &Account{nonce: nonce, balance: balance}
}

func (a *Account) Copy() *Account {
	return NewAccount(a.nonce, new(big.Int).Set(a.balance))
}

func DeserializeAccount(serialized []byte) *Account {
	pbAcc := &pb.Account{}
	if err := proto.Unmarshal(serialized, pbAcc); err != nil {
		log.Error(err)
		return nil
	}
	balance := big.NewInt(0)
	var voters []common.Address
	for _, pbv := range pbAcc.Voters {
		voters = append(voters, common.BytesToAddress(pbv))
	}
	return &Account{nonce: pbAcc.Nonce, balance: balance.SetBytes(pbAcc.Value), origin: common.BytesToAddress(pbAcc.Origin), voters: voters}
}