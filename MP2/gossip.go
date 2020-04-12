package main

import (
	"encoding/gob"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

const MAXNEIGHBORS = 50
const POLLINGPERIOD = 10 // poll neighbors for their neighors every 10 seconds

var localNodeName string             // tracks local node's name
var neighborMap map[string]*nodeComm // undirected graph // var neighborList []nodeComm // undirected graph
var numConns uint8                   // tracks number of other nodes connected to this node for bookkeeping
var localReceivingChannel chan Message
var localIPaddr string
var localPort string

var mp2ServiceAddr string

var transactionList []*TransactionMessage // List of TransactionMessage

func main() {
	arguments := os.Args
	if len(arguments) != 4 {
		fmt.Println("Expected Format: ./gossip [Local Node Name] [ipAddress] [port]")
		return
	}

	localNodeName = arguments[1]
	localIPaddr = arguments[2]
	localPort = arguments[3]

	localReceivingChannel = make(chan Message, 65536)
	mp2ServiceAddr = "localhost:2000" // TODO: fix this to be more dynamic
	neighborMap = make(map[string]*nodeComm)
	//transactionMap = make(map[string]*TransactionMessage)
	// transactionList = make([]*TransactionMessage, 2000)

	listener := setup_incoming_tcp()
	connect_to_service()
	//time.Sleep(5 * time.Second) // give service time to communicate
	go handle_service_comms()

	go listen_for_conns(listener)

	// handle the ever updating list of transactions (do any kind of reordering, maintenance work here)
	go blockchain()
	for {
	}
}

/**************************** Setup Functions ****************************/
func setup_incoming_tcp() net.Listener {
	listener, err := net.Listen("tcp", ":"+localPort) // open port
	check(err)
	return listener
}

func connect_to_service() {
	// Connect to mp2_service.py (over TCP)
	var err error = nil
	mp2Service := new(nodeComm)

	mp2Service.nodeName = "mp2Service"
	mp2Service.address = mp2ServiceAddr
	mp2Service.inbox = nil

	mp2Service.conn, err = net.Dial("tcp", mp2Service.address)
	check(err)

	neighborMap["mp2Service"] = mp2Service
}

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
	return node
}

/**************************** Go Routines ****************************/
func handle_service_comms() {
	mp2Service := neighborMap["mp2Service"]
	m := "CONNECT " + localNodeName + " " + localIPaddr + " " + localPort + "\n" // Send a message like "CONNECT node1 172.22.156.2 4444"
	fmt.Printf("handle_service_comms \t type: %T\n", m)
	_, err := mp2Service.conn.Write([]byte(m)) // sends m over TCP
	check(err)

	// handle messages
	//    1.1 Upto 3 INTRODUCE
	//    1.2 TRANSACTION messages
	//    1.3 QUIT/DIE
	buf := make([]byte, 1024) // Make a buffer to hold incoming data
	for {
		len, err := mp2Service.conn.Read(buf)
		check(err)
		mp2ServiceMsg := strings.Split(string(buf[:len]), "\n")[0]

		msgType := strings.Split(mp2ServiceMsg, " ")[0]
		if msgType == "INTRODUCE" {
			// Example: INTRODUCE node2 172.22.156.3 4567
			print(mp2ServiceMsg, "\n")
			node := new(nodeComm)
			neighborMap[node.nodeName] = node
			node.nodeName = strings.Split(mp2ServiceMsg, " ")[1]
			node.address = strings.Split(mp2ServiceMsg, " ")[2] + ":" + strings.Split(mp2ServiceMsg, " ")[3]
			connect_to_node(node)
			go node.handle_node_comm()
		} else if msgType == "TRANSACTION" {
			// Example: TRANSACTION 1551208414.204385 f78480653bf33e3fd700ee8fae89d53064c8dfa6 183 99 10
			transactiontime, _ := strconv.ParseFloat(strings.Split(mp2ServiceMsg, " ")[1], 64)
			transactionID := strings.Split(mp2ServiceMsg, " ")[2]
			transactionSrc, _ := strconv.Atoi(strings.Split(mp2ServiceMsg, " ")[3])
			transactionDest, _ := strconv.Atoi(strings.Split(mp2ServiceMsg, " ")[4])
			transactionAmt, _ := strconv.Atoi(strings.Split(mp2ServiceMsg, " ")[5])
			transaction := new(TransactionMessage)
			*transaction = TransactionMessage{transactiontime, transactionID, uint32(transactionSrc), uint32(transactionDest), uint64(transactionAmt)}

			transactionList = append(transactionList, transaction) // TODO: make this more efficient
		} else if (msgType == "QUIT") || (msgType == "DIE") {

		}
	}
}

func listen_for_conns(listener net.Listener) {
	var conn net.Conn
	var err error = nil
	for err == nil {
		conn, err = listener.Accept()
		node := setup_neighbor(conn)
		go node.handle_node_comm() // open up a go routine
	}
	_ = listener.Close()
	panic("ERROR receiving incoming connections")
}

func (node *nodeComm) handle_incoming_messages() {
	print("\nSuccesfully connected to ", node.nodeName)
	for {

	}
	// recieve incoming data
	//		- receiving poll request for transactionIDs from neighbors (POLL:TRANSACTION_IDs)
	//    - receiving pull request for specific transactions from neighbors (PULL:transactionID1,transactionID2,...)
	//    - receiving transactions from neighbors
	//				series of two messages:
	//					- TRANSACTION:num_transactions
	//					- a list of size num_transactions of data type TransactionMessage
	//
	//    - receiving poll request for this node's neighbors (POLL:NEIGHBORS)
	//    - list of neighbor's neighbors (as a string of type NEIGHBORS:num_neighbors)
	//				series of two messages:
	//					-	string in format NEIGHBORS:num_neighbors
	//					-	list of size num_neighbors of data type ConnectionMessage

}

func (node *nodeComm) handle_outgoing_messages() {
	//   one per NodeComm
	//   Algorithm: Every POLLINGPERIOD seconds, ask for transactionIDs, transactions, neigbors
	//   handle messages of the following type:
	//		 - poll neighbor for transactions (POLL:TRANSACTION_IDs)
	//     - send pull request
	//     - send transactions upon a pull request
	//     - periodically ask neighbor for its neighbors (send a string in the format: POLL:NEIGHBORS)
	//rand := time.Duration(rand.Intn(5)) // to reduce the stress on the network at the same time because of how I'm testing on the same system with the same clocks
	rand := time.Duration(0) // for stress test debugging purposes
	for {
		poll_for_transaction(node.conn)
		time.Sleep((POLLINGPERIOD + rand) * time.Second)
		poll_for_neighbors(node.conn)
		time.Sleep((POLLINGPERIOD + rand) * time.Second)
	}
}

func poll_for_transaction(conn net.Conn) {
	var TransactionIDs []string // empty TransactionIDs list, len() == 0
	m := TransactionRequest{true, TransactionIDs}
	tcpEnc := gob.NewEncoder(conn)
	fmt.Printf("poll_for_transaction \t type: %T\n", m)
	err := tcpEnc.Encode(m)
	check(err)
}

func poll_for_neighbors(conn net.Conn) {

}

func (node *nodeComm) handle_node_comm() {
	go node.handle_outgoing_messages()
	go node.receive_incoming_data() // put messages of this conn into node.inbox

	tcpEnc := gob.NewEncoder(node.conn)
	//tcpDec := gob.NewDecoder(node.conn)

	for val := range node.inbox {
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
				tcpEnc.Encode(m)

			} else if m.Request == true && len(m.TransactionIDs) != 0 {
				// send requested TransactionIDs's corresponding TransactionMessage
				for _, transactionID := range m.TransactionIDs {
					i, _ := find_transaction(transactionList, transactionID)
					fmt.Printf("case TransactionRequest 2 \t type: %T\n", m)
					tcpEnc.Encode(*transactionList[i])
				}

			} else if m.Request == false {
				// you have received list of transactionIDs other node has
				// check if you have the sent TransactionIDs TODO: make it faster by making it dict
				var newtransactionIDs []string
				for _, transactionID := range m.TransactionIDs {
					_, exists := find_transaction(transactionList, transactionID)
					if !exists {
						newtransactionIDs = append(newtransactionIDs, transactionID)
					}
				}

				m = TransactionRequest{true, newtransactionIDs}
				fmt.Printf("case TransactionRequest 3 \t type: %T\n", m)
				tcpEnc.Encode(m)
			}

		default:
			print("Unknown Type")
		}
	}
}

func (node *nodeComm) receive_incoming_data() {
	tcpDec := gob.NewDecoder(node.conn)
	// ideal way to do this
	for {
		var m *Message
		m = new(Message)
		err := tcpDec.Decode(m)
		check(err)
		node.inbox <- *m
	}

	/*
		for {
			var m Message
			m = new(ConnectionMessage)
			err := tcpDec.Decode(m)
			if err == nil {
				node.inbox <- m
				continue
			}

		}
	*/

}
