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
	m := new(ConnectionMessage)
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
	if _, exists := transactionMap[m.TransactionID]; !exists {
		transactionMap[m.TransactionID] = len(transactionList)
		transactionList = append(transactionList, newM)
	} else {
		Warning.Println("Got Transaction", m.TransactionID, "but already added to local set")
	}
}

func addBlock(m Block) {
	newM := new(Block)
	*newM = m
	// TODO: put block in a separate map for pending verification before commiting to block list and propagating
	if _, exists := blockMap[m.BlockID]; !exists {
		blockMap[m.BlockID] = len(blockList)
		blockList = append(blockList, newM)
	} else {
		Warning.Println("Got Block", m.BlockID, "but already added to local set")
	}
}

func addNode(m ConnectionMessage) {
	newM := new(ConnectionMessage)
	*newM = m
	// TODO: put block in a separate map for pending verification before commiting to block list and propagating
	if _, exists := nodeMap[m.NodeName]; !exists {
		nodeMap[m.NodeName] = len(nodeList)
		nodeList = append(nodeList, newM)
	} else {
		Warning.Println("Got Node", m.NodeName, "but already added to local set")
	}
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
		time.Sleep(POLLINGPERIOD * 5 * time.Millisecond)
		// print Transactions for debugging and verification purposes
		Debug.Println("\nCurrent Transactions:")
		for _, val := range transactionList {
			Debug.Println(*val)
		}
	}
}

func addNeighbor(newNode *nodeComm) {
	numConns++
	neighborMap[newNode.nodeName] = newNode
	neighborList = append(neighborList, newNode)
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
