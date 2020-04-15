package main

import (
	"encoding/gob"
	"encoding/json"
	"math/rand"
	"net"
	"time"
)

/**************************** Setup Functions ****************************/
func setup_neighbor(conn net.Conn) *nodeComm {
	// Called when a neighbor is trying to connect to this node
	tcpDec := gob.NewDecoder(conn)
	m := new(ConnectionMessage)
	err := tcpDec.Decode(m)
	check(err)
	node := new(nodeComm)
	node.nodeName = m.NodeName
	node.address = m.IPaddr + ":" + m.Port
	node.conn = conn
	node.inbox = make(chan Message, 65536)
	// node.isConnected = true
	Info.Println("setup_neighbor ", m.NodeName, "\n")
	neighborMapMutex.Lock()
	neighborMap[m.NodeName] = node
	neighborMapMutex.Unlock()
	return node
}

/**************************** Go Routines ****************************/
// TODO: create a dedicated TCP thread for doing all the outgoing communications and push to it via an outbox channel
func (node *nodeComm) handle_outgoing_messages() {
	//   one per NodeComm
	//   Algorithm: Every POLLINGPERIOD seconds, ask for transactionIDs, transactions, neigbors
	//   handle messages of the following type:
	//		 - poll neighbor for transactions (POLL:TRANSACTION_IDs)
	//     - send pull request
	//     - send transactions upon a pull request
	//     - periodically ask neighbor for its neighbors (send a string in the format: POLL:NEIGHBORS)

	rand := time.Duration(rand.Intn(3)) // to reduce the stress on the network at the same time because of how I'm testing on the same system with the same clocks
	//rand := time.Duration(0) // for stress test debugging purposes
	var alive bool
	for { // TODO: use isConnected node status here
		alive = node.check_node_status()
		if alive {
			node.poll_for_transaction()
		}
		time.Sleep((POLLINGPERIOD + rand) * time.Second)
		alive = node.check_node_status()
		if alive {
			node.poll_for_neighbors()
		}
		time.Sleep((POLLINGPERIOD + rand) * time.Second)
	}
}

func (node *nodeComm) poll_for_transaction() { // TODO: push to outbox
	// when called, asks node.conn neighbor about the transaction IDs it has
	TransactionIDs := make([]string, 0)
	m := TransactionRequest{true, TransactionIDs}
	err := node.tcp_enc_struct(m)
	check(err)
}

func (node *nodeComm) poll_for_neighbors() { // TODO: push to outbox
	// when called, ask neighbors about their neghbors
	m := DiscoveryMessage{true}
	err := node.tcp_enc_struct(m)
	check(err)
}

func (node *nodeComm) handle_node_comm() {
	Info.Println("Start handle_node_comm for ", node.nodeName)
	// handles all logic for communication between nodes
	go node.handle_outgoing_messages()
	go node.receive_incoming_data() // put messages of this conn into node.inbox
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
					err := node.tcp_enc_struct(newMsg)
					check(err)
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

				err := node.tcp_enc_struct(msg)

				check(err)

			} else if m.Request == true && len(m.TransactionIDs) != 0 {
				// send requested TransactionIDs's corresponding TransactionMessage
				for _, transactionID := range m.TransactionIDs {
					exists, transactionPtr := find_transaction(transactionID)
					if exists {
						err := node.tcp_enc_struct(*transactionPtr) // TODO: Should push to outbox
						check(err)
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
					err := node.tcp_enc_struct(msg)
					check(err)
				}
			}
		default:
			if m == "DISCONNECTED" {
				node.conn.Close()
				close(node.inbox)
				// TODO: set isConnected for the node to false
				neighborMapMutex.Lock()
				delete(neighborMap, node.nodeName) // TODO: track disconnected node using variable not map
				neighborMapMutex.Unlock()
				Info.Println("\nreturning from handle_node_comm for ", node.nodeName)
				return
			}
			Error.Println("\n ERROR Unknown Type in handle_node_comm \t ", m)
			panic("")
		}
	}
	panic("ERROR: outside for loop in handle_node_comm")
}

func (node *nodeComm) receive_incoming_data() {
	// handles incoming data from other nodes (not mp2_service)
	overflowData := ""
	var structDataList []string
	var structTypeList []string
	for {
		structTypeList, structDataList, overflowData = node.tcp_dec_struct(overflowData)

		if structTypeList == nil && structDataList == nil && overflowData == "DISCONNECTED" {
			node.inbox <- "DISCONNECTED"
			return
		}

		for i, structType := range structTypeList {
			structData := structDataList[i]

			// NOTE: Couldn't put the following code in tcp_dec_struct() function because functions needed concrete return types and interfaces weren't working
			// TODO convert to case statement
			if structType == "main.ConnectionMessage" {
				m := new(ConnectionMessage)
				err := json.Unmarshal([]byte(structData), m)
				check(err)
				node.inbox <- *m
			} else if structType == "main.TransactionMessage" {
				m := new(TransactionMessage)
				err := json.Unmarshal([]byte(structData), m)
				check(err)
				node.inbox <- *m
			} else if structType == "main.DiscoveryMessage" {
				m := new(DiscoveryMessage)
				err := json.Unmarshal([]byte(structData), m)
				check(err)
				node.inbox <- *m
			} else if structType == "main.TransactionRequest" {
				m := new(TransactionRequest)
				err := json.Unmarshal([]byte(structData), m)
				check(err)
				node.inbox <- *m
			} else {
				panic("\n ERROR receive_incoming_data type: " + structType)
			}
		}
	}
}
