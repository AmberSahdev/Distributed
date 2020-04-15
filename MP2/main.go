package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const MAXNEIGHBORS = 50
const POLLINGPERIOD = 10 // poll neighbors for their neighors, transactions every POLLINGPERIOD*2 seconds

var localNodeName string             // tracks local node's name
var neighborMap map[string]*nodeComm // undirected graph // var neighborList []nodeComm // undirected graph
var numConns uint8                   // tracks number of other nodes connected to this node for bookkeeping
var localIPaddr string
var localPort string

var mp2ServiceAddr string

var transactionList []*TransactionMessage // List of TransactionMessage // TODO: have a locking mechanism for this bc both handle_service_comms and node.handle_node_comm accessing it
var transactionMap map[string]*TransactionMessage

var neighborMapMutex sync.Mutex
var transactionMapMutex sync.Mutex
var transactionListMutex sync.Mutex

func main() {
	arguments := os.Args
	if len(arguments) != 4 {
		fmt.Println("Expected Format: ./gossip [Local Node Name] [ipAddress] [port]")
		return
	}

	localNodeName = arguments[1]
	localIPaddr = arguments[2]
	localPort = arguments[3]

	mp2ServiceAddr = "localhost:2000" // TODO: fix this to be more dynamic
	transactionMap = make(map[string]*TransactionMessage)

	neighborMap = make(map[string]*nodeComm)
	neighborMap[localNodeName] = nil // To avoid future errors

	neighborMapMutex = sync.Mutex{}
	transactionMapMutex = sync.Mutex{}
	transactionListMutex = sync.Mutex{}

	listener := setup_incoming_tcp()
	connect_to_service()
	go handle_service_comms()
	time.Sleep(5 * time.Second) // give service time to communicate

	go listen_for_conns(listener)

	go debug_print_transactions() // TODO: remove later

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

	neighborMapMutex.Lock()
	neighborMap["mp2Service"] = mp2Service
	neighborMapMutex.Unlock()
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
			node.nodeName = strings.Split(mp2ServiceMsg, " ")[1]
			node.address = strings.Split(mp2ServiceMsg, " ")[2] + ":" + strings.Split(mp2ServiceMsg, " ")[3]
			node.inbox = make(chan Message, 65536)
			connect_to_node(node)
			neighborMapMutex.Lock()
			neighborMap[node.nodeName] = node
			neighborMapMutex.Unlock()
			go node.handle_node_comm()
		} else if msgType == "TRANSACTION" {
			// Example: TRANSACTION 1551208414.204385 f78480653bf33e3fd700ee8fae89d53064c8dfa6 183 99 10
			//fmt.Println("received mp2_service transaction")
			transactiontime, _ := strconv.ParseFloat(strings.Split(mp2ServiceMsg, " ")[1], 64)
			transactionID := strings.Split(mp2ServiceMsg, " ")[2]
			transactionSrc, _ := strconv.Atoi(strings.Split(mp2ServiceMsg, " ")[3])
			transactionDest, _ := strconv.Atoi(strings.Split(mp2ServiceMsg, " ")[4])
			transactionAmt, _ := strconv.Atoi(strings.Split(mp2ServiceMsg, " ")[5])
			transaction := new(TransactionMessage)
			*transaction = TransactionMessage{transactiontime, transactionID, uint32(transactionSrc), uint32(transactionDest), uint64(transactionAmt)}
			add_transaction(*transaction) // transactionList = append(transactionList, transaction) // TODO: make this more efficient
		} else if (msgType == "QUIT") || (msgType == "DIE") {
			// TODO:
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
