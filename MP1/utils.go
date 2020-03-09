package main

import (
	"os"
	"strconv"
	"time"
	"unsafe"
)

// Performs our current error handling
func check(e error) {
	if e != nil {
		panic(e)
	}
}

func (destNode *nodeComms) unicast(m BankMessage) {
	destNode.outbox <- m
}

// Pushes outgoing data to all channels so that our outgoing networking threads can push it out to other nodes
func bMulticast(m BankMessage) {
	var i uint8
	for i = 0; i < numNodes; i++ {
		if nodeList[i].isConnected && i != localNodeNum {
			nodeList[i].outbox <- m
		}
	}
}

// Pushes outgoing data to all channels so that our outgoing networking threads can push it out to other nodes
func rMulticast(m BankMessage) {
	m.IsRMulticast = true
	bMulticast(m)
}

func max(x, y int64) int64 {
	if x > y {
		return x
	}
	return y
}

// Graphing Utils
func update_files(filePointers [2]*os.File) {
	// Adds a new line to our two files after every 5 seconds
	if allNodesAreConnected() {
		oneSecMark1 := time.Now()
		for {
			oneSecMark2 := time.Now()
			if oneSecMark2.Sub(oneSecMark1).Seconds() >= 5 {
				oneSecMark1 = time.Now()
				filePointers[0].WriteString("\n")
				filePointers[1].WriteString("\n")
			}
		}
	}
}

func create_files() (*os.File, *os.File) {
	fDelay, err := os.Create("delay.txt")
	check(err)
	fBandwidth, err := os.Create("bandwidth_node" + strconv.Itoa(int(localNodeNum)) + ".txt")
	check(err)
	return fDelay, fBandwidth
}

// finds size of a BankMessage
func (ptr *BankMessage) size() int {
	size := int(unsafe.Sizeof(*ptr))
	size += len(ptr.Transaction)
	return size
}
