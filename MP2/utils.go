package main

import (
	"encoding/gob"
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
	node.isConnected = false
	node.conn, err = net.Dial("tcp", node.address)
	check(err) // TODO: maybe dont crash here
	tcpEnc := gob.NewEncoder(node.conn)
	m := new(Message)
	*m = ConnectionMessage{
		NodeName: localNodeName,
		IPaddr:   localIPaddr,
		Port:     localPort,
	}
	// fmt.Println("connect_to_node \t ", m)
	err = tcpEnc.Encode(m)
	check(err)
}

func addTransaction(m TransactionMessage) {
	newM := new(TransactionMessage)
	*newM = m
	transactionListMutex.Lock()
	transactionList = append(transactionList, newM) // TODO: make it more efficient
	transactionListMutex.Unlock()

	transactionMapMutex.Lock()
	transactionMap[m.TransactionID] = newM // can't assume what's in the list is in the map.
	transactionMapMutex.Unlock()
}

// Find takes a slice and looks for an element in it. If found it will
// return it's key, otherwise it will return -1 and a bool of false.
func findTransaction(key TransID) (bool, *TransactionMessage) {
	transactionMapMutex.Lock()
	val, exists := transactionMap[key]
	transactionMapMutex.Unlock()
	if exists {
		return true, val
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

func (node *nodeComm) checkNodeStatus() bool {
	neighborMapMutex.Lock()
	defer neighborMapMutex.Unlock()
	if _, exists := neighborMap[node.nodeName]; !exists {
		Warning.Println("\nDisconnected ", node.nodeName)
		return false
	}
	return true
}
