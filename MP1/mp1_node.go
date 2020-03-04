package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

var numNodes int



var localTime vectorTime // tracks local vector timestamp
var numConns int // tracks number of other nodes connected to this node
var nodeNum int // keeps track of this node's number

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

type bank_message struct{
	originalSender int
	senderMessageNumber int
	event string
	sequenceNumber int
	isFinal bool
}

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

func (destNode *nodeComms) setConnected(status bool, conn net.Conn){
	destNode.isConnected = status
	// zero local vector timestamp index
	if status == false {
		numConns--
		conn.Close()
		close(destNode.outbox)
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
	for curNodeNum := 0; curNodeNum < numNodes; curNodeNum++ {
		nodeList[curNodeNum].port = port
		nodeList[curNodeNum].address = hostList[curNodeNum]
		conn, err := net.Dial("tcp", nodeList[curNodeNum].address+":"+nodeList[curNodeNum].port)
		if err == nil{
			nodeList[curNodeNum].setConnected(true, conn)
		}
	}
}

func handleIncomingConn(conn net.Conn){
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
					panic("Why is this an empty event?")
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

func handleIncomingConns(port string){
	listener, err := net.Listen("tcp", ":"+port)
	check(err)
	defer listener.Close()
	for {
		conn, err := listener.Accept()
		check(err)
		go handleIncomingConn(conn) // open up a goroutine
	}
}

func startHandlingIncomingConns(port string){
	go handleIncomingConns(port)
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
	if len(arguments) != 4 {
		fmt.Println(os.Stderr, "Expected Format: ./node [number of nodes] [port of centralized logging server]")
		return
	}
	numNodes, _ = strconv.Atoi(arguments[1])
	hostList := parseHostTextfile("../hosts.txt")
	agreedPort := arguments[2]
	nodeNum, _ = strconv.Atoi(arguments[3])
	startHandlingIncomingConns(agreedPort)
	setupConns(agreedPort, hostList)
	waitForAllNodes()
	go handleLocalEventGenerator()

}
