package state

import (
	"github.com/gagarinchain/common/api"
	"github.com/gagarinchain/common/eth/common"
	pb "github.com/gagarinchain/common/protobuff"
	"github.com/golang/protobuf/proto"
	"math/big"
)

type AccountImpl struct {
	nonce   uint64
	balance *big.Int
	origin  common.Address
	voters  []common.Address
}

func (a *AccountImpl) Serialize() []byte {
	b, e := proto.Marshal(a.ToStorageProto())
	if e != nil {
		log.Error("can't marshall account", e)
		return nil
	}
	return b
}

func (a *AccountImpl) ToStorageProto() *pb.Account {
	var addrBytes [][]byte
	for _, v := range a.Voters() {
		addrBytes = append(addrBytes, v.Bytes())
	}
	return &pb.Account{Nonce: a.Nonce(), Value: a.Balance().Bytes(), Origin: a.Origin().Bytes(), Voters: addrBytes}
}

func (a *AccountImpl) IncrementNonce() {
	a.nonce += 1
}

func (a *AccountImpl) SetOrigin(origin common.Address) {
	a.origin = origin
}

func (a *AccountImpl) Origin() common.Address {
	return a.origin
}

func (a *AccountImpl) Voters() []common.Address {
	return a.voters
}

func (a *AccountImpl) AddVoters(from common.Address) {
	a.voters = append(a.voters, from)
}

func (a *AccountImpl) Balance() *big.Int {
	return a.balance
}

func (a *AccountImpl) Nonce() uint64 {
	return a.nonce
}

func NewAccount(nonce uint64, balance *big.Int) *AccountImpl {
	return &AccountImpl{nonce: nonce, balance: balance}
}

func (a *AccountImpl) Copy() api.Account {
	return NewAccount(a.nonce, new(big.Int).Set(a.balance))
}

func DeserializeAccount(serialized []byte) *AccountImpl {
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
	return &AccountImpl{nonce: pbAcc.Nonce, balance: balance.SetBytes(pbAcc.Value), origin: common.BytesToAddress(pbAcc.Origin), voters: voters}
}
