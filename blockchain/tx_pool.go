package blockchain

import (
	"bytes"
	"context"
	"github.com/gagarinchain/network/common/tx"
	"sort"
	"sync"
	"time"
)

const Interval = 100 * time.Millisecond

type TransactionPoolImpl struct {
	pending []*tx.Transaction
	lock    sync.RWMutex
}

func NewTransactionPool() TransactionPool {
	return &TransactionPoolImpl{lock: sync.RWMutex{}}
}

func (tp *TransactionPoolImpl) Add(tx *tx.Transaction) {
	tp.lock.Lock()
	defer tp.lock.Unlock()

	tp.pending = append(tp.pending, tx)
}

func (tp *TransactionPoolImpl) getTopByFee() []*tx.Transaction {
	pendingCopy := append(tp.pending[:0:0], tp.pending...)
	sort.Sort(sort.Reverse(tx.ByFeeAndNonce(pendingCopy)))
	return pendingCopy
}

func (tp *TransactionPoolImpl) Iterator() tx.Iterator {
	tp.lock.RLock()
	defer tp.lock.RUnlock()
	return newIterator(tp.getTopByFee())
}

func (tp *TransactionPoolImpl) Remove(transaction *tx.Transaction) {
	tp.lock.Lock()
	defer tp.lock.Unlock()
	tp.remove(transaction)
}

func (tp *TransactionPoolImpl) remove(transaction *tx.Transaction) {
	for i, k := range tp.pending {
		if bytes.Equal(k.HashKey().Bytes(), transaction.HashKey().Bytes()) {
			tp.pending = append(tp.pending[:i], tp.pending[i+1:]...)
		}
	}
}

func (tp *TransactionPoolImpl) RemoveAll(transactions ...*tx.Transaction) {
	tp.lock.Lock()
	defer tp.lock.Unlock()

	for _, t := range transactions {
		tp.remove(t)
	}
}

type orderedIterator struct {
	txs   []*tx.Transaction
	state int
}

func newIterator(txs []*tx.Transaction) tx.Iterator {
	return &orderedIterator{txs: txs, state: 0}
}

//TODO make channel here, so we can collect incoming transactions until threshold is achieved
func (i *orderedIterator) Next() *tx.Transaction {
	if i.state < len(i.txs) {
		cur := i.txs[i.state]
		i.state++
		return cur
	} else {
		return nil
	}
}

func (i *orderedIterator) HasNext() bool {
	return i.state < len(i.txs)
}

func (tp *TransactionPoolImpl) Drain(ctx context.Context) (chunks chan []*tx.Transaction) {
	chunks = make(chan []*tx.Transaction)
	ticker := time.NewTicker(Interval)
	index := 0

	tick := func(pending []*tx.Transaction) {
		last := len(pending)
		part := tp.pending[index:last]
		sort.Sort(sort.Reverse(tx.ByFeeAndNonce(part)))
		index = last
		go func(txs chan []*tx.Transaction) {
			select {
			case txs <- part:
			case <-ctx.Done():
				log.Warning("Cancelled writing part")
				return
			}
		}(chunks)
	}

	tick(tp.pending)
	go func() {
		for {
			select {
			case <-ticker.C:
				tick(tp.pending)
			case <-ctx.Done():
				log.Warning("Cancelled writing chunks")
				close(chunks)
				return
			}
		}
	}()

	return
}
