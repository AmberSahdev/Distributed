package main

import (
	"encoding/gob"
	"fmt"
	"net"
)

/*
func (destNode *nodeComm) unicast(m TransactionMessage) {
	destNode.outbox <- m
}

// Sends TransactionMessage to all our neighbors in neighborList
func bMulticast(m TransactionMessage) {
	// TODO: change to neighborMap
	for _, node := range neighborList {
		node.outbox <- m
	}

}
*/

// Performs our current error handling
func check(e error) {
	if e != nil {
		fmt.Print("\n")
		panic(e)
	}
}

func max(x, y int64) int64 {
	if x > y {
		return x
	}
	return y
}

// Find takes a slice and looks for an element in it. If found it will
// return it's key, otherwise it will return -1 and a bool of false.
func find_transaction(slice []*TransactionMessage, val string) (int, bool) {
	for i, item := range slice {
		if item.TransactionID == val {
			return i, true
		}
	}
	return -1, false
}

func connect_to_node(node *nodeComm) {
	// called when this node is trying to connect to a neighbor after INTRODUCE message
	var err error
	node.conn, err = net.Dial("tcp", node.address)
	check(err) // TODO: maybe dont crash here

	// send ConnectionMessage
	/*
		m := "TRYNA CONNECT UP IN HERE"     // Send a message like "CONNECT node1 172.22.156.2 4444"
		_, err = node.conn.Write([]byte(m)) // sends m over TCP
		check(err)
	*/

	tcpEnc := gob.NewEncoder(node.conn)
	m := ConnectionMessage{
		NodeName: localNodeName,
		IPaddr:   localIPaddr,
		Port:     localPort,
	}
	err = tcpEnc.Encode(m)
	check(err)
}
