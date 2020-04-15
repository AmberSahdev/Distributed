package main

import (
	"encoding/gob"
	"math/rand"
	"net"
	"time"
)

/**************************** Setup Functions ****************************/
func setup_neighbor(conn net.Conn) *nodeComm {
	// Called when a neighbor is trying to connect to this node
	tcpDec := gob.NewDecoder(conn)
	var incoming interface{}
	err := tcpDec.Decode(&incoming)
	check(err)
	switch m := incoming.(type) {
	case *ConnectionMessage:
		node := new(nodeComm)
		node.isConnected = true
		node.nodeName = m.NodeName
		node.address = m.IPaddr + ":" + m.Port
		node.conn = conn
		node.inbox = make(chan Message, 65536)
		node.outbox = make(chan Message, 65536)
		// node.isConnected = true
		Info.Println("setup_neighbor", m.NodeName)
		neighborMapMutex.Lock()
		neighborMap[m.NodeName] = node
		neighborMapMutex.Unlock()
		return node
	default:
		Error.Println("Got unexpected type as first messge:", incoming)
	}
	return nil
}

/**************************** Go Routines ****************************/
// TODO: create a dedicated TCP thread for doing all the outgoing communications and push to it via an outbox channel
func (node *nodeComm) handle_outgoing_messages() {
	tcpEnc := gob.NewEncoder(node.conn)
	var m Message
	for m = range node.outbox {
		err := tcpEnc.Encode(m)
		if err != nil {
			Error.Println("Failed to send Message, receiver down?")
			_ = node.conn.Close()
			return
		}
	}
}

func (node *nodeComm) poll_for_transaction() { // TODO: push to outbox
	// when called, asks node.conn neighbor about the transaction IDs it has
	TransactionIDs := make([]string, 0)
	m := new(TransactionRequest)
	*m = TransactionRequest{true, TransactionIDs}
	if node.isConnected {
		node.outbox <- m
	} else {
		Warning.Println("node", node.nodeName, "not connected, cancelling Poll for Transactions")
	}
}

func (node *nodeComm) poll_for_neighbors() { // TODO: push to outbox
	// when called, ask neighbors about their neghbors
	m := new(DiscoveryMessage)
	*m = DiscoveryMessage{true}
	if node.isConnected {
		node.outbox <- m
	} else {
		Warning.Println("node", node.nodeName, "not connected, cancelling Poll for Neighbors")
	}

}

func (node *nodeComm) handle_node_comm() {
	Info.Println("Start handle_node_comm for ", node.nodeName)
	// handles all logic for communication between nodes
	go node.receive_incoming_data() // put messages of this conn into node.inbox
	go node.handle_outgoing_messages()

	// TODO: handle outbox goroutine responsible for all outgoing TCP comms

	lastSentTransactionIndex := 0 // to send only new transactionIDs, need to keep track of last sent index

	for val := range node.inbox {
		switch m := val.(type) {
		case ConnectionMessage:
			neighborMapMutex.Lock()
			if _, exists := neighborMap[m.NodeName]; !exists {
				newNode := new(nodeComm)
				newNode.nodeName = m.NodeName
				newNode.address = m.IPaddr + ":" + m.Port
				newNode.inbox = make(chan Message, 65536)
				newNode.outbox = make(chan Message, 65536)
				newNode.isConnected = true
				neighborMap[newNode.nodeName] = newNode
				neighborMapMutex.Unlock()
				connect_to_node(newNode)
				go newNode.handle_node_comm() // TODO: do this inside connect to node
			} else {
				neighborMapMutex.Unlock()
			}

		case TransactionMessage:
			Info.Println("exchanged transaction, from ", node.nodeName)
			add_transaction(m)

		case DiscoveryMessage:
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
					newMsg := *nodeComm_to_ConnectionMessage(v)
					node.outbox <- newMsg
					i++
				}
			} else {
				panic("ERROR received DiscoveryMessage with request false")
			}

		case TransactionRequest:
			// TODO: revist after blockchain discussion
			if m.Request == true && len(m.TransactionIDs) == 0 {
				// send all your TransactionIDs
				l := max(0, len(transactionList)-lastSentTransactionIndex)
				TransactionIDs := make([]string, l)

				j := 0
				for i := lastSentTransactionIndex; i < len(transactionList); i++ {
					TransactionIDs[j] = transactionList[i].TransactionID
					j++
				}

				msg := TransactionRequest{false, TransactionIDs}

				node.outbox <- msg

			} else if m.Request == true && len(m.TransactionIDs) != 0 {
				// send requested TransactionIDs's corresponding TransactionMessage
				for _, transactionID := range m.TransactionIDs {
					exists, transactionPtr := find_transaction(transactionID)
					if exists {
						node.outbox <- *transactionPtr
					} else {
						panic("ERROR You should not receive request for a transactionID that you do not have")
					}
				}
			} else if m.Request == false {
				// you have received list of transactionIDs other node has
				// check if you have the received TransactionIDs
				if len(m.TransactionIDs) != 0 {
					var newtransactionIDs []string
					for _, transactionID := range m.TransactionIDs {
						exists, _ := find_transaction(transactionID)
						if !exists {
							newtransactionIDs = append(newtransactionIDs, transactionID)
						}
					}

					msg := TransactionRequest{true, newtransactionIDs}
					node.outbox <- msg
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
			Warning.Println("Unknown Type in handle_node_comm ", m)
		}
	}
	panic("ERROR: outside for loop in handle_node_comm")
}

func (node *nodeComm) receive_incoming_data() {
	tcpDecode := gob.NewDecoder(node.conn)
	var err error = nil
	for err == nil {
		newM := new(Message)
		err = tcpDecode.Decode(newM)
		node.inbox <- *newM
	}
	node.inbox <- "DISCONNECTED"
	close(node.inbox)
	Warning.Println("Closing inbox for node", node.nodeName)
}

func (node *nodeComm) configureGossipProtocol() {
	//   one per NodeComm
	//   Algorithm: Every POLLINGPERIOD seconds, ask for transactionIDs, transactions, neigbors
	//   handle messages of the following type:
	//		 - poll neighbor for transactions (POLL:TRANSACTION_IDs)
	//     - send pull request
	//     - send transactions upon a pull request
	//     - periodically ask neighbor for its neighbors (send a string in the format: POLL:NEIGHBORS)

	rand := time.Duration(rand.Intn(3)) // to reduce the stress on the network at the same time because of how I'm testing on the same system with the same clocks
	//rand := time.Duration(0) // for stress test debugging purposes
	for node.isConnected { // TODO: use isConnected node status here
		node.poll_for_transaction()
		node.poll_for_neighbors()
		time.Sleep((POLLINGPERIOD + rand) * time.Second)
	}
}
