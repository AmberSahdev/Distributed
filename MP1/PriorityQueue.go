package main

import (
	"container/heap"
	"math"
)

// Derived from  https://golang.org/pkg/container/heap/
// An Item is something we manage in a priority queue.
type Item struct {
	value             message // The value of the item; arbitrary.
	priority          int     // The priority of the item in the queue. Corresponds to largest sequence number proposed so far
	index             int     // The index of the item in the heap. The index is needed by update and is maintained by the heap.Interface methods.
	responsesReceived []bool // 1 means reply received from node, 0 means message not received
}

func NewItem(m message) Item {
	var item = Item{
		value:             m,
		priority:          -1,
		index:             -1,
		responsesReceived: make([]bool, numNodes)
	}
	return item
}

// A PriorityQueue implements heap.Interface and holds Items.
type PriorityQueue []*Item

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	// We want Pop to give us the highest, not lowest, priority so we use greater than here.
	return pq[i].priority > pq[j].priority
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*Item)
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

// update modifies the priority and value of an Item in the queue.
func (pq *PriorityQueue) update(item *Item, value string, priority int) {
	item.value = value
	item.priority = priority
	heap.Fix(pq, item.index)
}

func (pq PriorityQueue) find(transactionID uint64) int {
	for i := 0; i < pq.Len(); i++ {
		if pq[i].value.transactionId == transactionID {
			return i
		}
	}
	return math.MaxInt32 // if nothing matches
}