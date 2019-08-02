package blockchain

import "github.com/gagarinchain/network/common/tx"

type TransactionPool interface {
	Add(tx *tx.Transaction)
	Iterator() tx.Iterator
	Remove(transaction *tx.Transaction)
	RemoveAll(transactions ...*tx.Transaction)
}
