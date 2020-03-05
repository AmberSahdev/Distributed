package main

import (
	"bufio"
	"container/heap"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"math"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

var commitNum int
var numNodes uint8     // specified parameter, number of starting nodes
var numConns uint8     // tracks number of other nodes connected to this node
var localNodeNum uint8 // tracks local node's number
var nodeList []nodeComms
var localReceivingChannel chan Message

func (destNode *nodeComms) communicationTask() {
	fmt.Println("preparing To Receive m's")
	tcpEnc := gob.NewEncoder(destNode.conn)
	defer destNode.conn.Close()
	fmt.Println("Ready To Receive m's")
	for m := range destNode.outbox {
		fmt.Println("about to send m")
		err := tcpEnc.Encode(m)
		fmt.Println("sent m")
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to send Message, receiver down?")
			return
		}
	}
}

func (destNode *nodeComms) openOutgoingConn() {
	destNode.isConnected = true
	numConns++
	destNode.outbox = make(chan Message)
	m := Message{
		OriginalSender:      localNodeNum,
		SenderMessageNumber: 0,
		transaction:         "",
		SequenceNumber:      -1,
		TransactionId:       math.MaxUint64,
		IsFinal:             false,
		IsRMulticast:        false}
	destNode.outbox <- m
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
	var m Message
	tcpDecode := gob.NewDecoder(conn)
	fmt.Println("Ready To Receive Data")
	err := tcpDecode.Decode(&m)
	incomingNodeNum := m.OriginalSender
	fmt.Println("Fuck Me")
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
	var conn net.Conn
	var err error = nil
	for err == nil {
		conn, err = listener.Accept()
		go receiveIncomingData(conn) // open up a go routine
	}
	fmt.Println("ERROR receiving incoming connections")
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
		m := Message{
			OriginalSender:      localNodeNum,
			SenderMessageNumber: -1,
			transaction:         text,
			SequenceNumber:      -1,
			TransactionId:       math.MaxUint64,
			IsFinal:             false,
			IsRMulticast:        false}

		localReceivingChannel <- m
	}

}

// TODO figure out how to block until everyone is connected
func waitForAllNodesSync() {
	time.Sleep(10 * time.Second)
	if numConns != numNodes {
		fmt.Fprintf(os.Stderr, "numConns: %d, numNodes: %d", numConns, numNodes)
	}
}

func setupConnections(port string, hostList []string) {
	var err error
	var curNodeNum uint8
	nodeList = make([]nodeComms, numNodes)
	listener := openListener(port)
	go handleAllIncomingConns(listener)
	for curNodeNum = 0; curNodeNum < numNodes; curNodeNum++ {
		nodeList[curNodeNum].port = port
		nodeList[curNodeNum].address = hostList[curNodeNum]
		if localNodeNum == curNodeNum {
			nodeList[curNodeNum].conn = nil
			nodeList[curNodeNum].outbox = localReceivingChannel
			nodeList[curNodeNum].isConnected = true
		} else {
			nodeList[curNodeNum].conn, err = net.Dial("tcp", (nodeList[curNodeNum].address)+":"+(nodeList[curNodeNum].port))
			if err == nil {
				fmt.Println("Dialed in!")
				nodeList[curNodeNum].openOutgoingConn()
			} else {
				fmt.Println(err)
			}
		}
	}
	waitForAllNodesSync()
}

func (m *Message) isAlreadyReceived() bool {
	return m.IsRMulticast && nodeList[m.OriginalSender].senderMessageNum >= m.SenderMessageNumber
}

func (m *Message) isProposal() bool {
	return m.SequenceNumber >= 0 && !m.IsFinal
}

func (m *Message) needsProposal() bool {
	return !m.IsFinal && m.SequenceNumber == -1
}

func (m *Message) setTransactionId() {
	m.TransactionId = (uint64(localNodeNum) << (64 - 8)) | (uint64(m.SenderMessageNumber) & 0x00FFFFFFFFFFFFFF) // {OriginalSender, SenderMessageNumber[55:0]}
}

// TODO Biggest Fuck, drains the Message Channel
func handleMessageChannel() {
	// Decentralized Causal + Total Ordering Protocol
	pq := make(PriorityQueue, 0)
	var maxFinalSeqNum int64 = 0
	var maxProposedSeqNum int64 = 0

	for m := range localReceivingChannel {
		if m.isAlreadyReceived() {
			continue
		}
		if m.SenderMessageNumber < 0 { // Handling of a local event
			nodeList[localNodeNum].senderMessageNum += 1
			m.SenderMessageNumber = nodeList[localNodeNum].senderMessageNum

			maxProposedSeqNum = findProposalNumber(maxProposedSeqNum, maxFinalSeqNum)

			item := NewItem(m, maxProposedSeqNum)
			heap.Push(&pq, &item)
			m.setTransactionId()
			bMulticast(m)
		} else { // Handling event received from a different node
			nodeList[m.OriginalSender].senderMessageNum = m.SequenceNumber
			if m.IsRMulticast {
				rMulticast(m)
			}
		}
		// delivery of Message to ISIS handler occurs here
		if m.isProposal() { // Receiving Message 2 and sending Message 3 handled here
			idx := pq.find(m.TransactionId)

			// update priority in pq = max(proposed priority, local priority)
			pq[idx].priority = max(m.SequenceNumber, pq[idx].priority)

			pq[idx].responsesReceived[m.OriginalSender] = true

			if allResponsesReceived(pq[idx].responsesReceived) {
				pq[idx].value.IsFinal = true
				m.IsFinal = true
				m.OriginalSender = localNodeNum
				nodeList[localNodeNum].senderMessageNum += 1
				m.SenderMessageNumber = nodeList[localNodeNum].senderMessageNum
				m.transaction = pq[idx].value.transaction
				m.SequenceNumber = int64(pq[idx].priority)
				rMulticast(m)
				maxFinalSeqNum = max(m.SequenceNumber, maxFinalSeqNum)
			}
			heap.Fix(&pq, idx)
			deliverAgreedTransactions(pq)

		} else if m.needsProposal() { // Receiving Message 1 and sending Message 2 handled here
			maxProposedSeqNum = findProposalNumber(maxProposedSeqNum, maxFinalSeqNum)

			item := NewItem(m, maxProposedSeqNum)
			heap.Push(&pq, &item)

			m.OriginalSender = localNodeNum
			nodeList[m.OriginalSender].unicast(m)

		} else if m.IsFinal { // Receiving Message 3 here
			// reorder based on final priority
			idx := pq.find(m.TransactionId)
			pq[idx].priority = m.SequenceNumber // update priority in pq = final priority
			pq[idx].value = m                   // copy the Message with the contents
			heap.Fix(&pq, idx)

			deliverAgreedTransactions(pq)
			maxFinalSeqNum = max(maxFinalSeqNum, m.SequenceNumber)
		}
	}
}

// finds the next proposal number to generate given the final sequence number already seen and max proposal number given
func findProposalNumber(maxProposedSeqNum int64, maxFinalSeqNum int64) int64 {
	if maxFinalSeqNum > maxProposedSeqNum {
		maxProposedSeqNum = maxFinalSeqNum
	}
	return maxProposedSeqNum + 1
}

func deliverAgreedTransactions(pq PriorityQueue) {
	// commit agreed transactions to account
	m := pq[0].value // highest priority // pq[0] is element with max priority
	for m.IsFinal {
		result := heap.Pop(&pq).(*Item) // TODO: put it into our account balances
		commitNum++
		fmt.Printf("%d %d "+result.value.transaction, result.value.SequenceNumber, commitNum)
		m = pq[0].value
	}
}

// check if Message ready (all nodes that are active and have response Received = true in responsesReceived)
func allResponsesReceived(responsesReceived []bool) bool {
	var i uint8
	for i = 0; i < numNodes; i++ {
		if i != localNodeNum && nodeList[i].isConnected && responsesReceived[i] == false {
			return false
		}
	}
	return true
}

func main() {
	arguments := os.Args
	if len(arguments) != 4 {
		fmt.Fprintln(os.Stderr, "Expected Format: ./node [number of nodes] [port of centralized logging server]")
		return
	}
	commitNum = 0
	numConns = 1
	newNumNodes, err := strconv.Atoi(arguments[1])
	check(err)
	hostList := parseHostTextfile("../hosts.txt")
	agreedPort := arguments[2]
	newNodeNum, err := strconv.Atoi(arguments[3])
	check(err)
	numNodes = uint8(newNumNodes)
	localNodeNum = uint8(newNodeNum)
	setupConnections(agreedPort, hostList)
	go handleLocalEventGenerator()
	handleMessageChannel()
}
