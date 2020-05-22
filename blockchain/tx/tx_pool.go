package tx

import (
	"bytes"
	"context"
	"github.com/gagarinchain/common/api"
	"sort"
	"sync"
	"time"
)

const Interval = 100 * time.Millisecond

type TransactionPoolImpl struct {
	pending []api.Transaction
	lock    sync.RWMutex
}

func NewTransactionPool() TransactionPool {
	return &TransactionPoolImpl{lock: sync.RWMutex{}}
}

func (tp *TransactionPoolImpl) Add(tx api.Transaction) {
	tp.lock.Lock()
	defer tp.lock.Unlock()

	tp.pending = append(tp.pending, tx)
}

func (tp *TransactionPoolImpl) getTopByFee() []api.Transaction {
	pendingCopy := append(tp.pending[:0:0], tp.pending...)
	//TODO we can sort only top n elements in array with optimized sorting algorithms
	sort.Sort(sort.Reverse(ByFeeAndNonce(pendingCopy)))
	return pendingCopy
}

func (tp *TransactionPoolImpl) Iterator() api.Iterator {
	tp.lock.RLock()
	defer tp.lock.RUnlock()
	return NewIterator(tp.getTopByFee())
}

func (tp *TransactionPoolImpl) Remove(transaction api.Transaction) {
	tp.lock.Lock()
	defer tp.lock.Unlock()
	tp.remove(transaction)
}

func (tp *TransactionPoolImpl) remove(transaction api.Transaction) {
	for i, k := range tp.pending {
		if bytes.Equal(k.Hash().Bytes(), transaction.Hash().Bytes()) {
			tp.pending = append(tp.pending[:i], tp.pending[i+1:]...)
			break
		}
	}
}

func (tp *TransactionPoolImpl) RemoveAll(transactions ...api.Transaction) {
	tp.lock.Lock()
	defer tp.lock.Unlock()

	for _, t := range transactions {
		tp.remove(t)
	}
}

type orderedIterator struct {
	txs   []api.Transaction
	state int
}

func NewIterator(txs []api.Transaction) api.Iterator {
	return &orderedIterator{txs: txs, state: 0}
}

func (i *orderedIterator) Next() api.Transaction {
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
func (tp *TransactionPoolImpl) Drain(ctx context.Context) (chunks chan []api.Transaction) {
	//this channel is used by child goroutines to write result if it's work
	chunks = make(chan []api.Transaction)
	//this channel which is used by child goroutines in tick to  notify main about job finishing.
	//when routine is done it is safe to close channel
	done := make(chan bool)
	//ticker to notify head routine it's time to start new child
	ticker := time.NewTicker(Interval)

	tick := func(pending []api.Transaction, index int) int {
		last := len(pending)
		part := make([]api.Transaction, len(tp.pending[index:last]))
		copy(part, tp.pending[index:last])
		sort.Sort(sort.Reverse(ByFeeAndNonce(part)))
		go func(txs chan []api.Transaction) {
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
