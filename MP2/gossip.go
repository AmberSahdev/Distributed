package main

import (
	"encoding/gob"
	"math"
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
		neighborMutex.Lock()
		addNeighbor(node)
		neighborMutex.Unlock()
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

// TODO: Remove this as it is unnecessary
func pollNeighbors() {
	// when called, ask neighbors about their neghbors
	m := new(DiscoveryMessage)
	*m = DiscoveryMessage{true}
	// TODO: Make this Mutex RW lock
	neighborMutex.RLock()
	for nodeName, node := range neighborMap {
		if nodeName != localNodeName && node.isConnected {
			node.outbox <- *m
		}
	}
	neighborMutex.RUnlock()
}

func (node *nodeComm) handleNodeComm(tcpDec *gob.Decoder) {
	Info.Println("Start handleNodeComm for ", node.nodeName)

	// handles all logic for communication between nodes
	go node.receiveIncomingData(tcpDec) // put messages of this conn into node.inbox
	go node.handleOutgoingMessages()

	// TODO: handle outbox goroutine responsible for all outgoing TCP comms

	// TODO: make lastSentTransactionIndex part of the node struct so you can easily reset it
	lastSentTransactionIndex := -1 // to send only new transactionIDs, need to keep track of last sent index
	lastSentBlockIndex := -1       // to send only new blocks, need to keep track of last sent index
	lastSentNodeIndex := -1        // to send only new nodes, need to keep track of last sent index

	for val := range node.inbox {
		switch m := val.(type) {
		case ConnectionMessage:
			neighborMutex.Lock()
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
				addNeighbor(newNode)
				neighborMutex.Unlock()
				go newNode.handleNodeComm(nil)
			} else {
				// 2-way communication now established
				incomingNode.isConnected = true
				neighborMutex.Unlock()
			}

		case TransactionMessage:
			Info.Println("Received transaction", m, "from", node.nodeName)
			addTransaction(m)

		case DiscoveryMessage:
			Info.Println("Processing Discovery Message:", m, "from", node.nodeName)
			if m.Request {
				nodeMutex.RLock()
				nodesPendingTransmission := make([]string, 0)
				for ; lastSentNodeIndex < len(nodeList)-1; lastSentNodeIndex++ {
					nodesPendingTransmission = append(nodesPendingTransmission, nodeList[lastSentNodeIndex+1].NodeName)
				}
				nodeMutex.RUnlock()
				blockMutex.RLock()
				blocksPendingTransmission := make([]BlockID, 0)
				for ; lastSentBlockIndex < len(blockList)-1; lastSentBlockIndex++ {
					blocksPendingTransmission = append(blocksPendingTransmission, blockList[lastSentBlockIndex+1].blockID)
				}
				blockMutex.RUnlock()
				transactionMutex.RLock()
				transactionsPendingTransmission := make([]TransID, 0)
				for ; lastSentTransactionIndex < len(transactionList)-1; lastSentTransactionIndex++ {
					transactionsPendingTransmission = append(transactionsPendingTransmission, transactionList[lastSentTransactionIndex+1].TransactionID)
				}
				transactionMutex.RUnlock()
				response := new(DiscoveryReplyMessage)
				*response = DiscoveryReplyMessage{nodesPendingTransmission, blocksPendingTransmission, transactionsPendingTransmission}
				node.outbox <- *response
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
				node.isConnected = false
				neighborMutex.Lock()
				removeNeighbor(node)
				neighborMutex.Unlock()
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
	for {
		rand := time.Duration(rand.Intn(500))                 // to reduce the stress on the network at the same time because of how I'm testing on the same system with the same clocks
		pollANeighbor()                                       // TODO: Polls a Neighbor (iterate over connectedNodes in order)
		time.Sleep((POLLINGPERIOD + rand) * time.Millisecond) //TODO: Tune Polling Period
		correctNumNeighbors()                                 //TODO: checks if number of Neighbors is too low, if so, randomly connect to a node from nodeList not in neighborMap
	}
}

func correctNumNeighbors() {
	nodeMutex.RLock()
	numNodes := len(nodeList) - 1 // index 0 is US!
	nodeMutex.RUnlock()
	desiredNumConnections := min(numNodes, int(math.Ceil(math.Log2(float64(numNodes))))+2)
	if desiredNumConnections < numConns {
		Warning.Println("Targeting having", desiredNumConnections, "connections, have", numConns)
		// TODO: successfully connect to 1 of remaining nodes randomly from nodeList
	}
}

func pollANeighbor() {
	if len(neighborList) > 0 {
		m := new(DiscoveryMessage)
		*m = DiscoveryMessage{true}

		for {
			neighborMutex.RLock()
			curNeighborToPoll %= len(neighborList)
			if neighborList[curNeighborToPoll] != nil {
				neighborList[curNeighborToPoll].outbox <- *m
				neighborMutex.RUnlock()
				curNeighborToPoll++
				break
			}
			neighborMutex.RUnlock()
			curNeighborToPoll++
		}
	}
}
