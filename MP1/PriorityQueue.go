package main

import (
	"math"
)

// Derived from  https://golang.org/pkg/container/heap/
// An Item is something we manage in a priority queue.
type Item struct {
	value             message // The value of the item; arbitrary.
	priority          int64   // The priority of the item in the queue. Corresponds to largest sequence number proposed so far
	index             int     // The index of the item in the heap. The index is needed by update and is maintained by the heap.Interface methods.
	responsesReceived []bool  // 1 means reply received from node, 0 means message not received
}

func NewItem(m message, priorityNum int64) Item {
	var item = Item{
		value:             m,
		priority:          priorityNum,
		index:             -1,
		responsesReceived: make([]bool, numNodes),
	}
	return item
}

// A PriorityQueue implements heap.Interface and holds Items.
type PriorityQueue []*Item

func (pq PriorityQueue) find(transactionId uint64) int {
	for i := 0; i < len(pq); i++ {
		if pq[i].value.transactionId == transactionId {
			return i
		}
	}
	return math.MaxInt32 // if nothing matches
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	// We want Pop to give us the highest, not lowest, priority so we use greater than here.
	panic("PriorityQueue Bad Func")
}

func (pq *PriorityQueue) Push(x interface{}) {
	panic("PriorityQueue Bad Func")
}

func (pq *PriorityQueue) Pop() interface{} {
	panic("PriorityQueue Bad Func")
}

// update modifies the priority and value of an Item in the queue.
func (pq *PriorityQueue) update(item *Item, value string, priority int) {
	panic("PriorityQueue Bad Func")
}
