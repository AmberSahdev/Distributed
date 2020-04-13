package main

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
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
	neighborMap[m.NodeName] = node
	node.nodeName = m.NodeName
	node.address = m.IPaddr + ":" + m.Port
	node.conn = conn
	node.inbox = make(chan Message, 65536)
	fmt.Println("setup_neighbor ", m.NodeName, "\n")
	return node
}

/**************************** Go Routines ****************************/
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
	for {
		node.poll_for_transaction()
		time.Sleep((POLLINGPERIOD + rand) * time.Second)

		// print transactions for debugging and verification purposes
		fmt.Println("\n")
		for _, val := range transactionMap {
			fmt.Println(*val)
		}

		node.poll_for_neighbors()
		time.Sleep((POLLINGPERIOD + rand) * time.Second)
	}
}

func (node *nodeComm) poll_for_transaction() {
	// when called, asks node.conn neighbor about the transaction IDs it has
	TransactionIDs := make([]string, 0)
	m := TransactionRequest{true, TransactionIDs}
	err := node.tcp_enc_struct(m)
	check(err)
}

func (node *nodeComm) poll_for_neighbors() {
	// when called, ask neighbors about their neghbors
	m := DiscoveryMessage{true}
	err := node.tcp_enc_struct(m)
	check(err)
}

func (node *nodeComm) handle_node_comm() {
	fmt.Println("Start handle_node_comm for ", node.nodeName)
	// handles all logic for communication between nodes
	go node.handle_outgoing_messages()
	go node.receive_incoming_data() // put messages of this conn into node.inbox

	lastSentTransactionIndex := 0 // to send only new transactionIDs, need to keep track of last sent index

	for val := range node.inbox {
		switch m := val.(type) {
		case ConnectionMessage:
			if _, exists := neighborMap[m.NodeName]; !exists {
				newNode := new(nodeComm)
				newNode.nodeName = m.NodeName
				newNode.address = m.IPaddr + ":" + m.Port
				newNode.inbox = make(chan Message, 65536)
				neighborMap[newNode.nodeName] = newNode
				connect_to_node(newNode)
				go newNode.handle_node_comm()
			}

		case TransactionMessage:
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
						err := node.tcp_enc_struct(*transactionPtr)
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
			panic("\n ERROR Unknown Type in handle_node_comm")
		}
	}
}

func (node *nodeComm) receive_incoming_data() {
	// handles incoming data from other nodes (not mp2_service)
	overflowData := ""
	var structDataList []string
	var structTypeList []string
	for {
		structTypeList, structDataList, overflowData = node.tcp_dec_struct(overflowData)

		for i, structType := range structTypeList {
			structData := structDataList[i]

			// NOTE: Couldn't put the following code in tcp_dec_struct() function because functions needed concrete return types and interfaces weren't working
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
