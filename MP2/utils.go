package main

import (
	"fmt"
	"net"
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

func connectToNode(node *nodeComm) error {
	// called when this node is trying to connect to a neighbor after INTRODUCE message
	var err error
	node.inbox = make(chan Message, 65536)
	node.outbox = make(chan Message, 65536)
	node.conn, err = net.Dial("tcp", node.address)
	if err != nil {
		return err
	}
	m := new(ConnectionMessage)
	*m = ConnectionMessage{
		NodeName: localNodeName,
		IPaddr:   localIPaddr,
		Port:     localPort,
	}
	node.outbox <- *m
	return nil
}

func addTransaction(m TransactionMessage, isInternal bool) {
	newM := new(TransactionMessage)
	*newM = m
	if _, exists := transactionMap[m.TransactionID]; !exists {
		transactionMap[m.TransactionID] = len(transactionList)
		transactionList = append(transactionList, newM)
		if !isInternal {
			logTransaction(m)
		}
	} else {
		Warning.Println("Got Transaction", m.TransactionID, "but already added to local set")
	}
}

func addBlock(m Block, isLocal bool) {
	newM := new(Block)
	*newM = m
	blockMutex.Lock()
	if _, exists := blockMap[m.BlockID]; !exists {
		myBlock := new(BlockInfo)
		*myBlock = BlockInfo{
			Index:           len(blockList),
			Verified:        false,
			ChildDependents: make([]BlockID, 0),
		}
		blockMap[m.BlockID] = myBlock
		blockList = append(blockList, newM)
		blockMutex.Unlock()
		if isLocal {
			localVerifiedBlocks <- newM
		} else {
			go verifyBlock(newM)
		}

		logBandwidthBlock(newM)
	} else {
		Warning.Println("Got Block", m.BlockID, "but already added to local set")
		blockMutex.Unlock()
	}
}

func addNode(m ConnectionMessage) {
	newM := new(ConnectionMessage)
	*newM = m
	if _, exists := nodeMap[m.NodeName]; !exists {
		nodeMap[m.NodeName] = len(nodeList)
		nodeList = append(nodeList, newM)
	} else {
		Warning.Println("Got Node", m.NodeName, "but already added to local set")
	}
}

func debugPrintTransactions() {
	for {
		time.Sleep(GOSSIPPOLLINGPERIOD * 5 * time.Millisecond)
		// print Transactions for debugging and verification purposes

		Debug.Println("\nCurrent Transactions:")
		for _, val := range transactionList {
			t := val.TransactionID
			transID := fmt.Sprintf("%x", t)
			Debug.Println(transID)
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

/*
DiscoveryReplyMessage
DiscoveryMessage
GossipRequestMessage
BatchGossipMessage
ConnectionMessage
*/
