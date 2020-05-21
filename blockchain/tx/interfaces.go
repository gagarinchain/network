package tx

import (
	"context"
	"github.com/gagarinchain/network/common/api"
)

type TransactionPool interface {
	Add(tx api.Transaction)
	Iterator() api.Iterator
	Remove(transaction api.Transaction)
	RemoveAll(transactions ...api.Transaction)
	Drain(ctx context.Context) (chunks chan []api.Transaction)
}
