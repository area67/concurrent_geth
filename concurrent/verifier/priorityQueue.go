package correctness_tool

import (
	"container/heap"
	"math/big"
)


// An QueueItem is something we manage in a priority queue.
type QueueItem struct {
	value    int // The value of the item; arbitrary.
	priority *big.Int    // The priority of the item in the queue.
	// The index is needed by update and is maintained by the heap.Interface methods.
	index int // The index of the item in the heap.
}

func NewQueueItem(_value int, _priority *big.Int) *QueueItem {
	return &QueueItem{
		value: _value,
		priority: _priority,
	}
}

// A PriorityQueue implements heap.Interface and holds Items.
type PriorityQueue []*QueueItem

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	// We want Pop to give us the highest, not lowest, priority so we use greater than here.
	return pq[i].priority.Cmp(pq[j].priority) == MORE
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*QueueItem)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

// update modifies the priority and value of an QueueItem in the queue.
func (pq *PriorityQueue) update(item *QueueItem, value int, priority *big.Int) {
	item.value = value
	item.priority = priority
	heap.Fix(pq, item.index)
}