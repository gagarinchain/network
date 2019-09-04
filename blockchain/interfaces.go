package blockchain

import (
	"context"
	"github.com/gagarinchain/network/common/tx"
)

type TransactionPool interface {
	Add(tx *tx.Transaction)
	Iterator() tx.Iterator
	Remove(transaction *tx.Transaction)
	RemoveAll(transactions ...*tx.Transaction)
	Drain(ctx context.Context) (chunks chan []*tx.Transaction)
}
