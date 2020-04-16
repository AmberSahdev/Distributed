package main

import (
	"encoding/gob"
	"math/rand"
	"net"
	"reflect"
	"time"
)

/**************************** Setup Functions ****************************/
func setupNeighbor(conn net.Conn) (*nodeComm, *gob.Decoder) {
	// Called when a new Node is trying to connect to this node
	tcpDec := gob.NewDecoder(conn)
	var incoming Message
	err := tcpDec.Decode(&incoming)
	check(err)
	myMsg := new(Message)
	*myMsg = incoming
	switch m := (*myMsg).(type) {
	case ConnectionMessage:
		node := new(nodeComm)
		node.isConnected = true
		node.nodeName = m.NodeName
		node.address = m.IPaddr + ":" + m.Port
		node.conn = conn
		node.inbox = make(chan Message, 65536)
		node.outbox = make(chan Message, 65536)
		Info.Println("setup_neighbor", m.NodeName)
		neighborMapMutex.Lock()
		neighborMap[m.NodeName] = node
		neighborMapMutex.Unlock()
		return node, tcpDec
	default:
		Error.Println("Got unexpected type as first messge:", incoming)
	}
	return nil, nil
}

/**************************** Go Routines ****************************/
// TODO: create a dedicated TCP thread for doing all the outgoing communications and push to it via an outbox channel
func (node *nodeComm) handleOutgoingMessages() {
	tcpEnc := gob.NewEncoder(node.conn)
	var m Message
	for m = range node.outbox {
		sendMsg := new(Message)
		*sendMsg = m
		err := tcpEnc.Encode(sendMsg)
		Info.Println("Sending", *sendMsg, "to", node.nodeName)
		if err != nil {
			Error.Println("Failed to send Message, receiver down?")
			_ = node.conn.Close()
			return
		}
	}
}

func pollNeighbors() { // TODO: push to all outboxes
	// when called, ask neighbors about their neghbors
	m := new(DiscoveryMessage)
	*m = DiscoveryMessage{true}
	// TODO: Make this Mutex RW lock
	neighborMapMutex.Lock()
	for nodeName, node := range neighborMap {
		if nodeName != localNodeName && node.isConnected {
			node.outbox <- *m
		}
	}
	neighborMapMutex.Unlock()
}

func (node *nodeComm) handleNodeComm(tcpDec *gob.Decoder) {
	Info.Println("Start handleNodeComm for ", node.nodeName)

	// handles all logic for communication between nodes
	go node.receiveIncomingData(tcpDec) // put messages of this conn into node.inbox
	go node.handleOutgoingMessages()

	// TODO: handle outbox goroutine responsible for all outgoing TCP comms

	// TODO: make lastSentTransactionIndex part of the node struct so you can easily reset it
	lastSentTransactionIndex := 0 // to send only new transactionIDs, need to keep track of last sent index

	for val := range node.inbox {
		switch m := val.(type) {
		case ConnectionMessage:
			neighborMapMutex.Lock()
			Info.Println("Processing Connection Message:", m, "from", node.nodeName)
			if incomingNode, exists := neighborMap[m.NodeName]; !exists {

				// TODO: you never opened the conn
				Error.Println("How Did we get here?")
				newNode := new(nodeComm)
				newNode.nodeName = m.NodeName
				newNode.address = m.IPaddr + ":" + m.Port
				newNode.inbox = make(chan Message, 65536)
				newNode.outbox = make(chan Message, 65536)
				newNode.isConnected = true
				neighborMap[newNode.nodeName] = newNode
				neighborMapMutex.Unlock()
				go newNode.handleNodeComm(nil)
			} else {
				// 2-way communication now established
				incomingNode.isConnected = true
				neighborMapMutex.Unlock()
			}

		case TransactionMessage:
			Info.Println("exchanged transaction, from ", node.nodeName)
			addTransaction(m)

		case DiscoveryMessage:
			Info.Println("Processing Discovery Message:", m, "from", node.nodeName)
			if m.Request {
				// send 5 random neighbors (first 5 neighbors)
				numNeighborsSend := min(5, len(neighborMap))
				i := 0
				for k, v := range neighborMap {
					if k == "mp2Service" || k == localNodeName || k == node.nodeName {
						continue
					} else if i == numNeighborsSend {
						break
					}
					node.outbox <- *nodecommToConnectionmessage(v)
					i++
				}
			} else {
				panic("ERROR received DiscoveryMessage with request false")
			}

		case TransactionRequest:
			Info.Println("Processing Transaction Request:", m, "from", node.nodeName)

			// TODO: revist after blockchain discussion
			if m.Request == true && len(m.TransactionIDs) == 0 {
				// send all your TransactionIDs
				l := max(0, len(transactionList)-lastSentTransactionIndex)
				TransactionIDs := make([]TransID, l)

				j := 0
				for i := lastSentTransactionIndex; i < len(transactionList); i++ {
					TransactionIDs[j] = transactionList[i].TransactionID
					j++
				}
				msg := new(Message)
				*msg = TransactionRequest{false, TransactionIDs}
				node.outbox <- *msg

			} else if m.Request == true && len(m.TransactionIDs) != 0 {
				// send requested TransactionIDs's corresponding TransactionMessage
				for _, transactionID := range m.TransactionIDs {
					exists, transactionPtr := findTransaction(transactionID)
					if exists {
						m := new(Message)
						*m = *transactionPtr
						node.outbox <- *m
					} else {
						panic("ERROR You should not receive request for a transactionID that you do not have")
					}
				}
			} else if m.Request == false {
				// you have received list of transactionIDs other node has
				// check if you have the received TransactionIDs
				if len(m.TransactionIDs) != 0 {
					var newTransactionIDs []TransID
					for _, transactionID := range m.TransactionIDs {
						exists, _ := findTransaction(transactionID)
						if !exists {
							newTransactionIDs = append(newTransactionIDs, transactionID)
						}
					}
					msg := new(Message)
					*msg = TransactionRequest{true, newTransactionIDs}
					node.outbox <- *msg
				}
			}
		default:
			if m == "DISCONNECTED" || m == nil {
				node.conn.Close()
				neighborMapMutex.Lock()
				node.isConnected = false
				delete(neighborMap, node.nodeName) // TODO: track disconnected node using variable not map
				neighborMapMutex.Unlock()
				Info.Println("\nreturning from handle_node_comm for ", node.nodeName)
				return
			}
			Warning.Println("Unknown Type in handle_node_comm ", m, "type:", reflect.TypeOf(m))
		}
	}
	panic("ERROR: outside for loop in handle_node_comm")
}

func (node *nodeComm) receiveIncomingData(tcpDec *gob.Decoder) {
	if tcpDec == nil {
		tcpDec = gob.NewDecoder(node.conn)
	}
	var err error = nil
	for err == nil {
		newM := new(Message)
		err = tcpDec.Decode(newM)
		Info.Println("Received", *newM, "from", node.nodeName)
		node.inbox <- *newM
	}
	node.inbox <- "DISCONNECTED"
	close(node.inbox)
	Warning.Println("Closing inbox for node", node.nodeName, "Cause:", err)
}

func configureGossipProtocol() {
	//   one per NodeComm
	//   Algorithm: Every POLLINGPERIOD seconds, ask for transactionIDs, transactions, neigbors
	//   handle messages of the following type:
	//		 - poll neighbor for transactions (POLL:TRANSACTION_IDs)
	//     - send pull request
	//     - send transactions upon a pull request
	//     - periodically ask neighbor for its neighbors (send a string in the format: POLL:NEIGHBORS)

	rand := time.Duration(rand.Intn(3)) // to reduce the stress on the network at the same time because of how I'm testing on the same system with the same clocks
	//rand := time.Duration(0) // for stress test debugging purposes
	time.Sleep((POLLINGPERIOD + rand) * time.Second)
	pollNeighbors()

}
