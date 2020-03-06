package main

import (
	"container/heap"
	"fmt"
	"math"
)

// Derived from  https://golang.org/pkg/container/heap/
// An Item is something we manage in a priority queue.
type Item struct {
	value             Message // The value of the item; arbitrary.
	priority          int64   // The priority of the item in the queue. Corresponds to largest sequence number proposed so far
	index             int     // The index of the item in the heap. The index is needed by update and is maintained by the heap.Interface methods.
	responsesReceived []bool  // 1 means reply received from node, 0 means Message not received
}

func NewItem(m Message, priorityNum int64) *Item {
	return &Item{
		value:             m,
		priority:          priorityNum,
		index:             -1,
		responsesReceived: make([]bool, numNodes),
	}
}

// A PriorityQueue implements heap.Interface and holds Items.
type PriorityQueue []*Item

func (pq PriorityQueue) find(transactionId uint64) int {
	for i := 0; i < len(pq); i++ {
		if pq[i].value.TransactionId == transactionId {
			return i
		}
	}
	return math.MaxInt32 // if nothing matches
}

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	// We want Pop to give us the highest, not lowest, priority so we use greater than here.
	return pq[i].priority < pq[j].priority
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

func test_heap() {
	// Some items and their priorities.
	numNodes = 10
	var test_m Message
	pq := make(PriorityQueue, 0)

	item1 := NewItem(test_m, 5)
	heap.Push(&pq, &item1)

	item2 := NewItem(test_m, 7)
	heap.Push(&pq, &item2)

	item3 := NewItem(test_m, 2)
	heap.Push(&pq, &item3)

	item4 := NewItem(test_m, 9)
	heap.Push(&pq, &item4)

	fmt.Println(heap.Pop(&pq).(*Item).priority)
	fmt.Println(heap.Pop(&pq).(*Item).priority)
	fmt.Println(heap.Pop(&pq).(*Item).priority)
	fmt.Println(heap.Pop(&pq).(*Item).priority)
}
