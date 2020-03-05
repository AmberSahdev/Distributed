package main

import (
	"bufio"
	"container/heap"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

var numNodes uint8     // specified parameter, number of starting nodes
var numConns uint8     // tracks number of other nodes connected to this node
var localNodeNum uint8 // tracks local node's number
var nodeList []nodeComms
var localReceivingChannel chan message

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
	var i uint8
	for i = 0; i < numNodes; i++ {
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
	var m message
	tcpDecode := gob.NewDecoder(conn)
	err := tcpDecode.Decode(&m)
	incomingNodeNum := m.originalSender
	if !nodeList[incomingNodeNum].isConnected {
		// set up a new connection
		nodeList[incomingNodeNum].openOutgoingConn()
	}
	defer nodeList[incomingNodeNum].closeOutgoingConn()
	for err != nil {
		localReceivingChannel <- m
		err = tcpDecode.Decode(&m)
	}
	now := time.Now()
	nanoseconds := float64(now.UnixNano()) / 1e9
	fmt.Printf("%f - Node %d disconnected\n", nanoseconds, incomingNodeNum)
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

func handleLocalEventGenerator() {
	// read stuff from stdin infinitely
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		text := scanner.Text()
		m := message{
			originalSender:      localNodeNum,
			senderMessageNumber: -1,
			transaction:         text,
			sequenceNumber:      -1,
			transactionId:       -1,
			isFinal:             false,
			isRMulticast:        false}

		localReceivingChannel <- m
	}

}

// TODO figure out how to block until everyone is connected
func waitForAllNodesSync() {
	time.Sleep(10)
}

func setupConnections(port string, hostList []string) {
	var err error
	var curNodeNum uint8
	listener := openListener(port)
	go handleAllIncomingConns(listener)

	for curNodeNum = 0; curNodeNum < numNodes; curNodeNum++ {
		nodeList[curNodeNum].port = port
		nodeList[curNodeNum].address = hostList[curNodeNum]
		if localNodeNum == curNodeNum {
			nodeList[curNodeNum].conn = nil
			nodeList[curNodeNum].outbox = localReceivingChannel
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
	return m.isRMulticast && nodeList[m.originalSender].senderMessageNum >= m.senderMessageNumber
}

func (m message) isProposal() bool {
	return m.sequenceNumber >= 0 && !m.isFinal
}

func (m *message) setTransactionID() {
	m.transactionId = (uint64(localNodeNum) << (64 - 8)) | (m.senderMessageNumber & 0x00FFFFFFFFFFFFFF) // {originalSender, senderMessageNumber[55:0]}
}

// TODO Biggest Fuck, drains the message Channel
func handleMessageChannel() {
	// Decentralized Causal - Total Ordering Protocol
	pq := make(PriorityQueue, 0)
	var maxFinalSeqNum int64 = 0
	var maxProposedSeqNum int64 = 0

	for m := range localReceivingChannel {
		if isAlreadyReceived(m) {
			continue
		}
		if m.senderMessageNumber < 0 { // Handling of a local event
			nodeList[localNodeNum].senderMessageNum += 1
			m.senderMessageNumber = nodeList[localNodeNum].senderMessageNum

			if maxFinalSeqNum > maxProposedSeqNum {
				maxProposedSeqNum = maxFinalSeqNum
			}
			maxProposedSeqNum += 1

			// todo make pq node
			item := NewItem(m, maxProposedSeqNum)

			heap.Push(&pq, item)
			m.setTransactionID()
			bMulticast(m)
		} else { // Handling event received from a different node
			nodeList[m.originalSender].senderMessageNum = m.sequenceNumber
			if m.isRMulticast {
				rMulticast(m)
			}
		}
		// delivery of message to ISIS handler occurs here
		if m.isProposal() {
			idx := pq.find(m.transactionId)

			// update priority in pq = max(proposed priority, local priority)
			pq[idx].priority = max(m.sequenceNumber, pq[idx].priority)

			pq[idx].responsesReceived[m.originalSender] = true

			if allResponsesReceived(pq[idx].responsesReceived) {
				pq[idx].value.isFinal = true
				m.isFinal = true
				m.originalSender = localNodeNum
				nodeList[localNodeNum].senderMessageNum += 1
				m.senderMessageNumber = nodeList[localNodeNum].senderMessageNum
				m.transaction = pq[idx].value.transaction
				m.sequenceNumber = int64(pq[idx].priority)
				rMulticast(m)
				maxFinalSeqNum = max(m.sequenceNumber, maxFinalSeqNum)
			}
			heap.Fix(pq, idx)

			// commmit agreed transacations to account
			m = pq[0].value // highest priority // pq[0] is element with max priority
			for m.isFinal {
				_ = heap.Pop(pq).(*Item) // TODO: put it into our account balances
				m = pq[0].value
			}

		} else if m.needsProposal() { // external message needing proposal
			m.proposeSequenceNum()
		}
	}
}

// check if message ready (all nodes that are active and have response Received = true in responsesReceived)
func allResponsesReceived(responsesReceived []bool) bool {
	for i := 0; i < numNodes; i++ {
		if nodeList[i].isConnected && responsesReceived[i] == false {
			return false
		}
	}
	return true
}

func main() {
	var err error
	arguments := os.Args
	if len(arguments) != 4 {
		fmt.Println(os.Stderr, "Expected Format: ./node [number of nodes] [port of centralized logging server]")
		return
	}
	numNodes, err = strconv.Atoi(arguments[1])
	check(err)
	hostList := parseHostTextfile("../hosts.txt")
	agreedPort := arguments[2]
	newNodeNum, err := strconv.Atoi(arguments[3])
	check(err)
	localNodeNum = uint8(newNodeNum)
	setupConnections(agreedPort, hostList)
	go handleLocalEventGenerator()
	handleMessageChannel()
}

func max(x, y int64) int64 {
	if x > y {
		return x
	}
	return y
}
