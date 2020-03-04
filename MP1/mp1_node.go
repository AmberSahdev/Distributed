package main

import (
	"bufio"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

var numNodes int     // specified parameter, number of starting nodes
var numConns int     // tracks number of other nodes connected to this node
var localNodeNum int // tracks local node's number
var nodeList []nodeComms
var localReceivingChannels chan message

type nodeComms struct {
	sequenceNumber      int      //
	port        string   // outgoing node's port
	address     string   // outgoing node's address
	conn        net.Conn // TODO find out if pass by value or pointer is better here
	outbox      chan message
	isConnected bool
}

// Todo define an actual encode & decode method for this and settle on a/many concrete
// types for the decoded result
type message bank_message

type bank_message struct {
	originalSender      int    // local node number of sender of original transaction
	senderMessageNumber int    // local message number sent by sender
	transaction         string // sender's transaction generator string
	sequenceNumber      int    // -1 if uninitialized, used for proposal and final
	isFinal             bool   // distinguishes finalized vs proposed sequence Numbers
	isRMulticast        bool   // instructs receiver to multicast message
}

// Performs our current error handling
func check(e error) {
	if e != nil {
		panic(e)
	}
}

func (destNode nodeComms) unicast(m message) {
	if m.originalSender == localNodeNum {
		nodeList[localNodeNum].sequenceNumber++
		m.senderMessageNumber = nodeList[localNodeNum].sequenceNumber
	}
	destNode.outbox <- m
}

// Pushes outgoing data to all channels so that our outgoing networking threads can push it out to other nodes
func bMulticast(m message) {
	if m.originalSender == localNodeNum {
		nodeList[localNodeNum].sequenceNumber++
		m.senderMessageNumber = nodeList[localNodeNum].sequenceNumber
	}
	for i := 0; i < numNodes; i++ {
		if nodeList[i].isConnected && i != localNodeNum {
			nodeList[i].outbox <- m
		}
	}
}

// Pushes outgoing data to all channels so that our outgoing networking threads can push it out to other nodes
func rMulticast(m message) {
	m.isRMulticast = true
	bMulticast(m)
}

func (destNode *nodeComms) communicationTask() {
	tcpEnc := gob.NewEncoder(destNode.conn)
	defer destNode.conn.Close()
	for m := range destNode.outbox {
		err := tcpEnc.Encode(m)
		if err != nil {
			fmt.Println(os.Stderr, "Failed to send message, receiver down?")
			return
		}
	}
}

func (destNode *nodeComms) openOutgoingConn() {
	destNode.isConnected = true
	numConns++
	destNode.outbox = make(chan message)
	go destNode.communicationTask()
}

// ran when the incoming connection to this node throws and errror
func (destNode *nodeComms) closeOutgoingConn() {
	destNode.isConnected = false
	numConns--
	close(destNode.outbox)
}

func parseHostTextfile(path string) []string {
	dat, err := ioutil.ReadFile(path)
	check(err)
	return strings.Split(string(dat), "\n")
}

func receiveIncomingData(conn net.Conn) {
	incomingNodeNum := -1 // no known node number yet
	var m message
	tcpDecode := gob.NewDecoder(conn)
	for {
		err := tcpDecode.Decode(&m)
		if err != nil {
			now := time.Now()
			nanoseconds := float64(now.UnixNano()) / 1e9
			fmt.Printf("%f - Node %d disconnected\n", nanoseconds, incomingNodeNum)
			return
		}
		localReceivingChannels <- m
	}
}

func handleAllIncomingConns(listener net.Listener) {
	defer listener.Close()
	for {
		conn, err := listener.Accept()
		check(err)
		go receiveIncomingData(conn) // open up a go routine
	}
}

func openListener(port string) net.Listener {
	listener, err := net.Listen("tcp", ":"+port) // open port
	check(err)
	return listener
}

// TODO handles stdin transaction messaging
func handleLocalEventGenerator() {
	// read stuff from stdin infinitely
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		text = scanner.Text()
		m := message{
			originalSender = localNodeNum,
			senderMessageNumber = ,
			transaction = text,
			sequenceNumber = ,
			isFinal = false,
			isRMulticast = true
		}

		rMulticast(m)
		// add to local channel TODO
	}

}

// TODO figure out how to block until everyone is connected
func waitForAllNodesSync() {
	time.Sleep(10)
}

func setupConnections(port string, hostList []string) {
	var err error
	listener := openListener(port)
	go handleAllIncomingConns(listener)

	for curNodeNum := 0; curNodeNum < numNodes; curNodeNum++ {
		nodeList[curNodeNum].port = port
		nodeList[curNodeNum].address = hostList[curNodeNum]
		if localNodeNum == curNodeNum {
			nodeList[curNodeNum].conn = nil
			nodeList[curNodeNum].outbox = localReceivingChannels
		} else {
			nodeList[curNodeNum].conn, err = net.Dial("tcp", nodeList[curNodeNum].address+":"+nodeList[curNodeNum].port)
			if err == nil {
				nodeList[curNodeNum].openOutgoingConn()
			}
		}
	}
	waitForAllNodesSync()
}

func isAlreadyReceived(m message) bool {
	return nodeList[m.originalSender].sequenceNumber >= m.sequenceNumber
}

// TODO Biggest Fuck, drains the message Channel
func handleMessageChannel() {
	for m := range localReceivingChannels {
		if isAlreadyReceived(m) {
			continue
		}
		nodeList[m.originalSender].sequenceNumber = m.sequenceNumber
		if m.isRMulticast {
			rMulticast(m)
		}
		// delivery of message to ISIS handler occurs here

	}
}

func main() {
	arguments := os.Args
	if len(arguments) != 4 {
		fmt.Println(os.Stderr, "Expected Format: ./node [number of nodes] [port of centralized logging server]")
		return
	}
	numNodes, _ = strconv.Atoi(arguments[1])
	hostList := parseHostTextfile("../hosts.txt")
	agreedPort := arguments[2]
	localNodeNum, _ = strconv.Atoi(arguments[3])
	setupConnections(agreedPort, hostList)
	go handleLocalEventGenerator()
	handleMessageChannel()
}
