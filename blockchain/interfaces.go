package blockchain

import "github.com/poslibp2p/common/tx"

type TransactionPool interface {
	Add(tx *tx.Transaction)
	Iterator() tx.Iterator
	Remove(transaction *tx.Transaction)
}
