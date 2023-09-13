package txpool

import (
	"container/heap"
	"math/big"
	"sync"
	"sync/atomic"

	"github.com/0xPolygon/polygon-edge/types"
)

// A thread-safe wrapper of a minNonceQueue.
// All methods assume the (correct) lock is held.
type accountQueue struct {
	sync.RWMutex
	wLock uint32
	queue minNonceQueue
}

func newAccountQueue() *accountQueue {
	q := accountQueue{
		queue: make(minNonceQueue, 0),
	}

	heap.Init(&q.queue)

	return &q
}

func (q *accountQueue) lock(write bool) {
	switch write {
	case true:
		q.Lock()
		atomic.StoreUint32(&q.wLock, 1)
	case false:
		q.RLock()
		atomic.StoreUint32(&q.wLock, 0)
	}
}

func (q *accountQueue) unlock() {
	if atomic.SwapUint32(&q.wLock, 0) == 1 {
		q.Unlock()
	} else {
		q.RUnlock()
	}
}

// prune removes all transactions from the queue
// with nonce lower than given.
func (q *accountQueue) prune(nonce uint64) (
	pruned []*types.Transaction,
) {
	for {
		if tx := q.peek(); tx == nil || tx.Nonce >= nonce {
			break
		}

		pruned = append(pruned, q.pop())
	}

	return
}

// clear removes all transactions from the queue.
func (q *accountQueue) clear() (removed []*types.Transaction) {
	// store txs
	removed = q.queue

	// clear the underlying queue
	q.queue = q.queue[:0]

	return
}

// push pushes the given transactions onto the queue.
func (q *accountQueue) push(tx *types.Transaction) {
	heap.Push(&q.queue, tx)
}

// peek returns the first transaction from the queue without removing it.
func (q *accountQueue) peek() *types.Transaction {
	if q.length() == 0 {
		return nil
	}

	return q.queue.Peek()
}

// pop removes the first transactions from the queue and returns it.
func (q *accountQueue) pop() *types.Transaction {
	if q.length() == 0 {
		return nil
	}

	transaction, ok := heap.Pop(&q.queue).(*types.Transaction)
	if !ok {
		return nil
	}

	return transaction
}

// length returns the number of transactions in the queue.
func (q *accountQueue) length() uint64 {
	return uint64(q.queue.Len())
}

// transactions sorted by nonce (ascending)
type minNonceQueue []*types.Transaction

/* Queue methods required by the heap interface */

func (q *minNonceQueue) Peek() *types.Transaction {
	if q.Len() == 0 {
		return nil
	}

	return (*q)[0]
}

func (q *minNonceQueue) Len() int {
	return len(*q)
}

func (q *minNonceQueue) Swap(i, j int) {
	(*q)[i], (*q)[j] = (*q)[j], (*q)[i]
}

func (q *minNonceQueue) Less(i, j int) bool {
	// The higher gas price Tx comes first if the nonces are same
	if (*q)[i].Nonce == (*q)[j].Nonce {
		return (*q)[i].GasPrice.Cmp((*q)[j].GasPrice) > 0
	}

	return (*q)[i].Nonce < (*q)[j].Nonce
}

func (q *minNonceQueue) Push(x interface{}) {
	transaction, ok := x.(*types.Transaction)
	if !ok {
		return
	}

	*q = append(*q, transaction)
}

func (q *minNonceQueue) Pop() interface{} {
	old := q
	n := len(*old)
	x := (*old)[n-1]
	*q = (*old)[0 : n-1]

	return x
}

type pricedQueue struct {
	queue *maxPriceQueue
}

// newPricesQueue creates the priced queue with initial transactions and base fee
func newPricesQueue(baseFee uint64, initialTxs []*types.Transaction) *pricedQueue {
	q := &pricedQueue{
		queue: &maxPriceQueue{
			baseFee: new(big.Int).SetUint64(baseFee),
			txs:     initialTxs,
		},
	}

	heap.Init(q.queue)

	return q
}

// Pushes the given transactions onto the queue.
func (q *pricedQueue) push(tx *types.Transaction) {
	heap.Push(q.queue, tx)
}

// Pop removes the first transaction from the queue
// or nil if the queue is empty.
func (q *pricedQueue) pop() *types.Transaction {
	if q.length() == 0 {
		return nil
	}

	transaction, ok := heap.Pop(q.queue).(*types.Transaction)
	if !ok {
		return nil
	}

	return transaction
}

// length returns the number of transactions in the queue.
func (q *pricedQueue) length() int {
	return q.queue.Len()
}

// // transactions sorted by gas price (descending)
// type maxPriceQueue []*types.Transaction

// transactions sorted by gas price (descending)
type maxPriceQueue struct {
	baseFee *big.Int
	txs     []*types.Transaction
}

/* Queue methods required by the heap interface */

func (q *maxPriceQueue) Peek() *types.Transaction {
	if q.Len() == 0 {
		return nil
	}

	return q.txs[0]
}

func (q *maxPriceQueue) Len() int {
	return len(q.txs)
}

func (q *maxPriceQueue) Swap(i, j int) {
	q.txs[i], q.txs[j] = q.txs[j], q.txs[i]
}

func (q *maxPriceQueue) Push(x interface{}) {
	transaction, ok := x.(*types.Transaction)
	if !ok {
		return
	}

	q.txs = append(q.txs, transaction)
}

func (q *maxPriceQueue) Pop() interface{} {
	old := q.txs
	n := len(old)
	x := old[n-1]
	q.txs = old[0 : n-1]

	return x
}

// @see https://github.com/etclabscore/core-geth/blob/4e2b0e37f89515a4e7b6bafaa40910a296cb38c0/core/txpool/list.go#L458
// for details why is something implemented like it is
func (q *maxPriceQueue) Less(i, j int) bool {
	switch cmp(q.txs[i], q.txs[j], q.baseFee) {
	case -1:
		return false
	case 1:
		return true
	default:
		return q.txs[i].Nonce < q.txs[j].Nonce
	}
}

func cmp(a, b *types.Transaction, baseFee *big.Int) int {
	if baseFee.BitLen() > 0 {
		// Compare effective tips if baseFee is specified
		if c := a.EffectiveGasTip(baseFee).Cmp(b.EffectiveGasTip(baseFee)); c != 0 {
			return c
		}
	}

	// Compare fee caps if baseFee is not specified or effective tips are equal
	if c := a.GetGasFeeCap().Cmp(b.GetGasFeeCap()); c != 0 {
		return c
	}

	// Compare tips if effective tips and fee caps are equal
	return a.GetGasTipCap().Cmp(b.GetGasTipCap())
}
