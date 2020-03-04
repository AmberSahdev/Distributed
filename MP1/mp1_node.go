package main

import (
	"bufio"
	"encoding/gob"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

var numNodes int // specified parameter, number of starting nodes
var numConns int // tracks number of other nodes connected to this node
var nodeNum int  // tracks local node's number
var nodeList []nodeComms

type nodeComms struct {
	number      int
	port        string
	address     string
	conn        net.Conn // TODO find out if pass by value or pointer is better here
	outbox      chan message
	isConnected bool
	isSelf      bool
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
	destNode.outbox <- m
}

// Pushes outgoing data to all channels so that our outgoing networking threads can push it out to other nodes
func bMulticast(m message) {
	for i := 0; i < numNodes; i++ {
		if nodeList[i].isConnected && !nodeList[i].isSelf {
			nodeList[i].unicast(m)
		}
	}
	// deliver to local
	nodeList[nodeNum].outbox <- m
}

// Pushes outgoing data to all channels so that our outgoing networking threads can push it out to other nodes
func rMulticast(m message) {
	m.isRMulticast = true
	bMulticast(m)
}

func (destNode *nodeComms) communicationTask() {
	var err error = nil
	var m message
	enc := gob.NewEncoder(destNode.conn)
	defer destNode.setConnected(false)
	for err != nil {
		m <- destNode.outbox
		err = enc.Encode(m)
	}
}

func (destNode *nodeComms) openOutgoingConn() {
	destNode.isConnected = true
	numConns++
	destNode.outbox = make(chan message)
	go destNode.communicationTask()
}

func (destNode *nodeComms) closeOutgoingConn() {
	numConns--
	destNode.conn.Close()
	close(destNode.outbox)
}

func parseHostTextfile(path string) []string {
	dat, err := ioutil.ReadFile(path)
	check(err)
	return strings.Split(dat, "\n")
}

func handleIncomingConn(conn net.Conn) {
	buf := make([]byte, 1048576) // Make 1MB buffer to hold incoming data
	incomingNodeNum := -1
	for {
		inputLen, err := conn.Read(buf)
		now := time.Now()
		nanoseconds := float64(now.UnixNano()) / 1e9
		if err != nil {
			fmt.Printf("%f - Node %d disconnected\n", nanoseconds, incomingNodeNum)
			return
		}
		eventList := strings.Split(string(buf[:inputLen]), "\n")
		for i := 0; i < len(eventList); i++ {
			if len(eventList[i]) == 0 {
				if i != len(eventList)-1 {
					panic("Why is this an empty transaction?")
				}
				continue
			}
			now = time.Now()
			nanoseconds = float64(now.UnixNano()) / 1e9
			if nodeName == "" {
				nodeName = eventList[i]
			}
			contents := strings.Split(eventList[i], " ")
			contentLen := len(contents)
			transmittedTime := 0.0
			fmt.Printf("%f ", nanoseconds)
			if contentLen == 2 {
				transmittedTime, err = strconv.ParseFloat(contents[0], 64)
				check(err)
				fmt.Println(nodeName + " " + contents[1])
			} else if contentLen == 1 {
				fmt.Println("- " + nodeName + " connected")
			} else {
				fmt.Println("\n%d", contentLen)
				panic("Why not be length 1 or 2")
			}

			// Writing to files delay.txt and bandwidth.txt
			// filePointers[0].WriteString(fmt.Sprintf("%f ", nanoseconds-transmittedTime))
			// filePointers[1].WriteString(fmt.Sprintf("%d ", inputLen))
		}
	}
}

func handleIncomingConns(port string) {
	listener, err := net.Listen("tcp", ":"+port)
	check(err)
	defer listener.Close()
	for {
		conn, err := listener.Accept()
		check(err)
		go handleIncomingConn(conn) // open up a goroutine
	}
}

func startHandlingIncomingConns(port string) {
	go handleIncomingConns(port)
	return
}

// handles stdin transaction messaging
func handleLocalEventGenerator() {
	var m message
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		scanner.Text()
	}
	rMulticast(m)
}

func setupConnections(port string, hostList []string) {
	var err error
	startHandlingIncomingConns(port)
	for curNodeNum := 0; curNodeNum < numNodes; curNodeNum++ {
		nodeList[curNodeNum].port = port
		nodeList[curNodeNum].address = hostList[curNodeNum]
		nodeList[curNodeNum].conn, err = net.Dial("tcp", nodeList[curNodeNum].address+":"+nodeList[curNodeNum].port)
		if err == nil {
			nodeList[curNodeNum].setConnected(true)
		}
	}
	waitForAllNodesSync()
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
	nodeNum, _ = strconv.Atoi(arguments[3])
	setupConnections(agreedPort, hostList)
	handleLocalEventGenerator()
}
