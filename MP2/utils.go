package main

import (
	"math"
	"net"
	"strings"
	"time"
)

// Performs our current error handling
func check(e error) {
	if e != nil {
		Error.Print("Error Detected:\n")
		panic(e)
	}
}

func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func min(x, y int) int {
	if x > y {
		return y
	}
	return x
}

// TODO: Make this a goroutine
func connectToNode(node *nodeComm) {
	// called when this node is trying to connect to a neighbor after INTRODUCE message
	var err error
	node.inbox = make(chan Message, 65536)
	node.outbox = make(chan Message, 65536)
	node.conn, err = net.Dial("tcp", node.address)
	check(err) // TODO: maybe dont crash here IMPORTANT!!!
	m := new(Message)
	*m = ConnectionMessage{
		NodeName: localNodeName,
		IPaddr:   localIPaddr,
		Port:     localPort,
	}
	node.outbox <- *m
}

func addTransaction(m TransactionMessage) {
	newM := new(TransactionMessage)
	*newM = m
	transactionMutex.Lock()
	transactionList = append(transactionList, newM)
	transactionMap[m.TransactionID] = len(transactionList)
	transactionMutex.Unlock()
}

// Find takes a slice and looks for an element in it. If found it will
// return it's key, otherwise it will return -1 and a bool of false.
func findTransaction(key TransID) (bool, *TransactionMessage) {
	transactionMutex.RLock()
	defer transactionMutex.RUnlock()
	ind, exists := transactionMap[key]
	if exists && ind != math.MaxInt64 {
		return true, transactionList[ind]
	} else {
		return false, nil
	}
}

func nodecommToConnectionmessage(nodePtr *nodeComm) *ConnectionMessage {
	ret := new(ConnectionMessage)
	ret.NodeName = nodePtr.nodeName
	ret.IPaddr = strings.Split(nodePtr.address, ":")[0]
	ret.Port = strings.Split(nodePtr.address, ":")[1]
	return ret
}

func debugPrintTransactions() {
	for {
		time.Sleep(POLLINGPERIOD * time.Second)

		// print transactions for debugging and verification purposes
		Info.Println("\n")
		for _, val := range transactionList {
			Info.Println(*val)
		}
	}
}

func addNeighbor(newNode *nodeComm) {
	numConns++
	neighborMap[newNode.nodeName] = newNode
	neighborList = neighborList
}

func removeNeighbor(node *nodeComm) {
	numConns--
	delete(neighborMap, node.nodeName)
	for ind, curNode := range neighborList {
		if curNode != nil && curNode.nodeName == node.nodeName {
			neighborList[ind] = nil
			return
		}
	}
	Error.Println("Failed to delete", node.nodeName, "from neighborList!")
}
