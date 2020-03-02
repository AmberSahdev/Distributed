package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"strings"
)

var numNodes int

type vectorTimestamp struct {
	time []int
	lock mux
}



var localTime vectorTime // tracks local vector timestamp
var numConns int // tracks number of other nodes connected to this node

type nodeComms struct {
	number int
	port string
	address string
	isConnected bool
	isLeader bool
	isSelf bool
	outbox chan message
}

var nodeList []nodeComms

// Todo define an actual encode & decode method for this and settle on a/many concrete
// types for the decoded result
type message interface{}

// Performs our current error handling
func check(e error) {
	if e != nil {
		panic(e)
	}
}

// Pushes outgoing data to all channels so that our outgoing networking threads can push it out to other nodes
func rMulticast(m message){
	// Add vector clock increment here

	for i := 0; i < numNodes; i++{
		if nodeList[i].isConnected {
			nodeList[i].outbox <- m
		}
	}
}

func (destNode *nodeComms) sendData(m message) error{
	// Todo
	// serialize m and send it via conn (a TCP connection)
	//  use conn.Write() and get error handling from conn
	conn.Write(m)
	return err
}

func (destNode *nodeComms) communicationTask(conn net.Conn){
	var err error = nil
	for err != nil {
		m :<- destNode.outbox
		err = destNode.sendData(conn, m)
	}
	destNode.setConnected(false, conn)
	return
}

func electNewLeader(){
	// Todo
	// Drain buffer of sequenced messages
	// elect leader with most sequenced messages
	return
}

func (destNode *nodeComms) setConnected(status bool, conn net.Conn){
	destNode.isConnected = status
	// zero local vector timestamp index
	if status == false {
		numConns--
		conn.Close()
		if destNode.isLeader {
			updateLeader()
		}
	} else {
		numConns++
		destNode.outbox = make(chan message)
		go nodeList[curNodeNum].communicationTask(conn)
	}
	return
}

func parseHostTextfile(path string) []string{
	dat, err := ioutil.ReadFile(path)
	check(err)
	return strings.Split(dat, "\n")
}
dat, err := ioutil.ReadFile(hostTextFile)
check(err)

func setupConns(port string, hostList []string) {
	var numConns = 0
	for curNodeNum := 0; curNodeNum < numNodes; curNodeNum++ {
		nodeList[curNodeNum].port = port
		nodeList[curNodeNum].address = hostList[curNodeNum]
		conn, err := net.Dial("tcp", nodeList[curNodeNum].address+":"+nodeList[curNodeNum].port)
		if err == nil{
			nodeList[curNodeNum].setConnected(true, conn)
		}
	}
}

func handleIncomingConns(){
	// TODO fix this
	return
}

// handles stdin event messaging
func handleLocalEventGenerator(){
	var m message
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		scanner.Text()
	}

	rMulticast(m)
}

func main() {
	arguments := os.Args
	if len(arguments) != 3 {
		fmt.Println(os.Stderr, "Expected Format: ./node [number of nodes] [port of centralized logging server]")
		return
	}
	numNodes, _ = strconv.Atoi(arguments[1])
	hostList := parseHostTextfile("../hosts.txt")
	agreedPort := arguments[2]
	fmt.Println(numNodes)
	go handleIncomingConns()
	setupConns(agreedPort, hostList)
	go handleLocalEventGenerator()
	// do operations if we are the leader
}
