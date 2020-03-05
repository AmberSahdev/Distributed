package main

import (
	"math"
)

// Derived from  https://golang.org/pkg/container/heap/
// An Item is something we manage in a priority queue.
type Item struct {
	value             message // The value of the item; arbitrary.
	priority          int64     // The priority of the item in the queue. Corresponds to largest sequence number proposed so far
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

func (pq PriorityQueue) find(transactionID uint64) int {
	for i := 0; i < len(pq); i++ {
		if pq[i].value.transactionId == transactionID {
			return i
		}
	}
	return math.MaxInt32 // if nothing matches
}
