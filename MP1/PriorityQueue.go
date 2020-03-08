package main

import (
	"math"
)

// Derived from  https://golang.org/pkg/container/heap/
// An Item is something we manage in a priority queue.
type Item struct {
	value             BankMessage // The value of the item; arbitrary.
	priority          int64       // The priority of the item in the queue. Corresponds to largest sequence number proposed so far
	index             int         // The index of the item in the heap. The index is needed by update and is maintained by the heap.Interface methods.
	responsesReceived []bool      // 1 means reply received from node, 0 means Message not received
}

func NewItem(m BankMessage, priorityNum int64) *Item {
	var item = new(Item)
	item.value = m
	item.priority = priorityNum
	item.index = -1
	item.responsesReceived = make([]bool, numNodes)
	return item
}

// A PriorityQueue implements heap.Interface and holds Items.
type PriorityQueue []*Item

func (pq PriorityQueue) find(transactionId uint64) int {
	result := math.MaxInt32
	for i := 0; i < len(pq); i++ {
		if pq[i].value.TransactionId == transactionId {
			if result != math.MaxInt32 {
				panic("Duplicates in PQ!")
			}
			result = i
		}
	}
	return result
}

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	// We want Pop to give us the smallest number, not highest, priority so we use less than here.
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

/*
func test_heap() {
	// Some items and their priorities.
	var test_m Message
	pq := make(PriorityQueue, 0)
	var i int64
	for i=0;i<10;i--{
		heap.Push(&pq, NewItem(test_m, 10-i))
	}
	for i=0;i<10;i-- {
		fmt.Println(heap.Pop(&pq).(*Item).priority)
	}
}
*/
