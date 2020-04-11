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

var mp2ServiceAddr string

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
	if len(arguments) != 4 {
		fmt.Println("Expected Format: ./gossip [Local Node Name] [ipAddress] [port]")
		return
	}

	localNodeName = arguments[1]
	localIPaddr = arguments[2]
	localPort = arguments[3]

	localReceivingChannel = make(chan TransactionMessage, 65536)
	mp2ServiceAddr = "localhost:2000" // TODO: fix this to be more dynamic
	neighborMap = make(map[string]*nodeComm)

	listener := setup_incoming_tcp()
	connect_to_service()
	go handle_service_comms()

	go listen_for_conns(listener)
	//go handle_incoming_messages(listener)

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

func connect_to_service() {
	// Connect to mp2_service.py (over TCP)
	var err error = nil
	mp2Service := new(nodeComm)

	mp2Service.nodeName = "mp2Service"
	mp2Service.address = mp2ServiceAddr
	mp2Service.outbox = nil // dont need an outbox because only sending 1 message

	mp2Service.conn, err = net.Dial("tcp", mp2Service.address)
	check(err)

	neighborMap["mp2Service"] = mp2Service
}

func setup_neighbor(conn net.Conn) *nodeComm {
	// Called when a neighbor is trying to connect to this node
	tcpDecode := gob.NewDecoder(conn)
	m := new(ConnectionMessage)
	err := tcpDecode.Decode(m)
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
	_, err := mp2Service.conn.Write([]byte(m))                                   // sends m over TCP
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
			print(mp2ServiceMsg)
			node := new(nodeComm)
			neighborMap[node.nodeName] = node
			node.nodeName = strings.Split(mp2ServiceMsg, " ")[1]
			node.address = strings.Split(mp2ServiceMsg, " ")[2] + ":" + strings.Split(mp2ServiceMsg, " ")[3]
			connect_to_node(node)

		} else if msgType == "TRANSACTION" {
			//println(mp2ServiceMsg)
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
		go node.handle_incoming_messages() // open up a go routine
	}
	_ = listener.Close()
	panic("ERROR receiving incoming connections")
}

func (node *nodeComm) handle_incoming_messages() {
	print("\nSuccesfully connected to node ", node.nodeName)
	for {

	}

	// recieve incoming data
	//    - list of neighbor's neighbors
	//    - receiving pull request for transactions from neighbors
	//    - receiving transactions from neighbors

}

func (node *nodeComm) handle_outgoing_messages() {
	//   one per NodeComm
	//   open an outgoing connection
	//   handle messages of the following type:
	//     - ask neighbor for its neighbors
	//     - send pull request
	//     - send transactions upon a pull request

}
