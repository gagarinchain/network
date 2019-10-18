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
	//TODO we can sort only top n elements in array with optimized sorting algorithms
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

//For drain to work correctly we must guarantee that pending transactions are never reordered and are only appended
func (tp *TransactionPoolImpl) Drain(ctx context.Context) (chunks chan []*tx.Transaction) {
	//this channel is used by child goroutines to write result if it's work
	chunks = make(chan []*tx.Transaction)
	//this channel which is used by child goroutines in tick to  notify main about job finishing.
	//when routine is done it is safe to close channel
	done := make(chan bool)
	//ticker to notify head routine it's time to start new child
	ticker := time.NewTicker(Interval)

	tick := func(pending []*tx.Transaction, index int) int {
		last := len(pending)
		part := make([]*tx.Transaction, len(tp.pending[index:last]))
		copy(part, tp.pending[index:last])
		sort.Sort(sort.Reverse(tx.ByFeeAndNonce(part)))
		go func(txs chan []*tx.Transaction) {
			select {
			case txs <- part:
				done <- true //notify we ended work
			case <-ctx.Done():
				log.Warning("Cancelled writing part")
				done <- true //notify we ended work
				return
			}
		}(chunks)

		return last
	}

	index := 0
	tp.lock.RLock()
	index = tick(tp.pending, index)
	tp.lock.RUnlock()
	trigger := ticker.C
	go func(trigger <-chan time.Time, index int) {
		working := 1
		for {
			select {
			case <-done:
				working--
			case <-trigger:
				working++
				tp.lock.RLock()
				index = tick(tp.pending, index)
				tp.lock.RUnlock()
			case <-ctx.Done(): //should be careful we can spin-wait here
				log.Warning("Cancelled writing chunks")
				trigger = nil //prevent starting new tasks
				if working <= 0 {
					close(chunks)
					return
				}
			}
		}
	}(trigger, index)

	return
}
