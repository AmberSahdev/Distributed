package main

import (
	"encoding/gob"
	"fmt"
	"net"
	"os"
	"strings"
)

const MAXNEIGHBORS = 50
const POLLINGPERIOD = 10 // poll neighbors for their neighors every 10 seconds

var localNodeName string             // tracks local node's name
var neighborMap map[string]*nodeComm // undirected graph // var neighborList []nodeComm // undirected graph
var numConns uint8                   // tracks number of other nodes connected to this node for bookkeeping
var localReceivingChannel chan TransactionMessage
var localIPaddr string
var localPort string

var mp2_service_addr string

/* TODO
1. pull gossip
  1.1 setup a incoming channel
  1.2 send messages over TCP to service
  1.3 poll for data (receive incoming data)
  1.4 receive polls (send data)
2.
*/

func main() {
	arguments := os.Args
	if len(arguments) != 3 {
		fmt.Println("Expected Format: ./gossip [Local Node Number] [ip_address:port]")
		return
	}

	localNodeName = arguments[1]
	localIPaddr = strings.Split(arguments[2], ":")[0]
	localPort = strings.Split(arguments[2], ":")[1]

	localReceivingChannel = make(chan TransactionMessage, 65536)
	mp2_service_addr = "" // TODO: fill it
	neighborMap = make(map[string]*nodeComm)

	listener := setup_incoming_tcp()
	connect_to_service()
	go handle_service_comms()
	/*
		go listen_for_conns(listener)
		go handle_incoming_messages()
	*/

	// open outgoing message thread
	// go handle_outgoing_messages()

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

func listen_for_conns(listener net.Listener) {
	var conn net.Conn
	var err error = nil
	for err == nil {
		conn, err = listener.Accept()
		go receiveIncomingData(conn) // open up a go routine
	}
	_ = listener.Close()
	panic("ERROR receiving incoming connections")
}

func connect_to_service() {
	// Connect to mp2_service (over TCP)
	var err error = nil
	mp2_service := new(nodeComm)

	mp2_service.nodeName = "mp2_service"
	mp2_service.address = mp2_service_addr
	mp2_service.outbox = nil // we dont need an outbox as we're only gonna be sending one message

	mp2_service.conn, err = net.Dial("tcp", mp2_service.address)

	neighborMap["mp2_service"] = mp2_service
}

func handle_service_comms() {
	mp2_service := neighborMap["mp2_service"]
	tcpEnc := gob.NewEncoder(mp2_service.conn)
	m := "CONNECT " + localNodeName + " " + localIPaddr + " " + localPort + "\n" // Send a message like "CONNECT node1 172.22.156.2 4444"
	err := tcpEnc.Encode(m)                                                      // sends m over TCP

	// handle messages
	//    1.1 Upto 3 INTRODUCE
	//    1.2 TRANSACTION messages
	//    1.3 QUIT/DIE
}

func connect_to_initial_neighbors(initialIntroductionMessages []string) {
	/*
	   1. set up outgoing channels
	   2. add to neighborList
	*/
	for m := range initialIntroductionMessages {
		// parse m for ip_addr, port
		node := connect_to_node(ip_addr, port)
		neighborList.append(node)
		numConns++
	}
}

/**************************** Go Routines ****************************/
func handle_incoming_messages() {
	// setup incoming listener

	// recieve incoming data
	//    - receiving introductions from mp2_service
	//    - receiving transactions from mp2_service
	//    - QUIT/DIE from mp2_service
	//    - list of neighbor's neighbors
	//    - receiving pull request for transactions from neighbors
	//    - receiving transactions from neighbors

}

func (destNode *nodeComm) handle_outgoing_messages() {
	//   one per NodeComm
	//   open an outgoing connection
	//   handle messages of the following type:
	//     - send pull request
	//     - send transactions upon a pull request
	//     - ask neighbor for its neighbors
}
