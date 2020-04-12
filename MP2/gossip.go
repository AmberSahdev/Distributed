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
		node.poll_for_neighbors()
		time.Sleep((POLLINGPERIOD + rand) * time.Second)
	}
}

func (node *nodeComm) poll_for_transaction() {
	// when called, asks node.conn neighbor about the transaction IDs it has
	var TransactionIDs []string // empty TransactionIDs list, len() == 0
	m := TransactionRequest{true, TransactionIDs}
	fmt.Printf("poll_for_transaction \t type: %T\n", m)
	err := node.tcp_enc_struct(m)
	check(err)
}

func (node *nodeComm) poll_for_neighbors() {
	// when called, ask neighbors about their neghbors
	// TODO:
}

func (node *nodeComm) handle_node_comm() {
	// handles all logic for communication between nodes
	go node.handle_outgoing_messages()
	go node.receive_incoming_data() // put messages of this conn into node.inbox

	for val := range node.inbox {
		fmt.Println("popping from node.inbox")
		switch m := val.(type) {
		case ConnectionMessage:
			print("ConnectionMessage")
		case TransactionMessage:
			print("TransactionMessage")
		case DiscoveryMessage:
			print("DiscoveryMessage")
		case TransactionRequest:
			print("TransactionRequest")
			if m.Request == true && len(m.TransactionIDs) == 0 {
				// send all your TransactionIDs TODO: send only new transactionIDs (keep track of last sent index)
				TransactionIDs := make([]string, len(transactionList)) // //var TransactionIDs []string

				for i, transaction := range transactionList {
					TransactionIDs[i] = transaction.TransactionID
				}
				m = TransactionRequest{false, TransactionIDs}
				fmt.Printf("case TransactionRequest 1 \t type: %T\n", m)
				err := node.tcp_enc_struct(m)
				check(err)

			} else if m.Request == true && len(m.TransactionIDs) != 0 {
				// send requested TransactionIDs's corresponding TransactionMessage
				for _, transactionID := range m.TransactionIDs {
					i, _ := find_transaction(transactionList, transactionID)
					fmt.Printf("case TransactionRequest 2 \t type: %T\n", m)
					err := node.tcp_enc_struct(*transactionList[i])
					check(err)
				}

			} else if m.Request == false {
				// you have received list of transactionIDs other node has
				// check if you have the sent TransactionIDs (TODO: make it faster by making it dict)
				var newtransactionIDs []string
				for _, transactionID := range m.TransactionIDs {
					_, exists := find_transaction(transactionList, transactionID)
					if !exists {
						newtransactionIDs = append(newtransactionIDs, transactionID)
					}
				}

				m = TransactionRequest{true, newtransactionIDs}
				fmt.Printf("case TransactionRequest 3 \t type: %T\n", m)
				err := node.tcp_enc_struct(m)
				check(err)
			}

		default:
			panic("\n ERROR Unknown Type in handle_node_comm")
		}
	}
}

func (node *nodeComm) receive_incoming_data() {
	// handles incoming data from other nodes (not mp2_service)
	for {
		structType, structData := node.tcp_dec_struct()
		fmt.Println("decoded in receive_incoming_data")
		Databytes := []byte(structData)

		// NOTE: Couldn't put the following code in tcp_dec_struct() function because functions needed concrete return types and interfaces weren't working
		if structType == "main.ConnectionMessage" {
			m := new(ConnectionMessage)
			err := json.Unmarshal(Databytes, m)
			check(err)
			node.inbox <- *m
		} else if structType == "main.TransactionMessage" {
			m := new(TransactionMessage)
			err := json.Unmarshal(Databytes, m)
			check(err)
			node.inbox <- *m
		} else if structType == "main.DiscoveryMessage" {
			m := new(DiscoveryMessage)
			err := json.Unmarshal(Databytes, m)
			check(err)
			node.inbox <- *m
		} else if structType == "main.TransactionRequest" {
			m := new(TransactionRequest)
			err := json.Unmarshal(Databytes, m)
			check(err)
			node.inbox <- *m
		} else {
			panic("\n ERROR receive_incoming_data type: " + structType)
		}
	}
}
