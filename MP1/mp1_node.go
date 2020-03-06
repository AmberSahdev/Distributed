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
	// fmt.Println("preparing To Receive m's")
	tcpEnc := gob.NewEncoder(destNode.conn)

	defer destNode.conn.Close()
	// fmt.Println("Ready To Receive m's")
	for m := range destNode.outbox {
		err := tcpEnc.Encode(m)
		// fmt.Println("ENCODE m IN communicationTask:", m)
		if err != nil {
			fmt.Println("Failed to send Message, receiver down?")
			return
		}
	}
}

func (destNode *nodeComms) openOutgoingConn() {
	var err error
	destNode.conn, err = net.Dial("tcp", destNode.address)
	if err == nil {
		destNode.isConnected = true
		numConns++
		destNode.outbox = make(chan Message, 1024)
		m := Message{
			OriginalSender:      localNodeNum,
			SenderMessageNumber: 0,
			Transaction:         "",
			SequenceNumber:      -1,
			TransactionId:       math.MaxUint64,
			IsFinal:             false,
			IsRMulticast:        false}
		go destNode.communicationTask()
		destNode.outbox <- m
	}
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
	var new_m *Message
	var incomingNodeNum uint8 = math.MaxUint8
	tcpDecode := gob.NewDecoder(conn)
	new_m = new(Message)
	err := tcpDecode.Decode(new_m)
	if err == nil {
		incomingNodeNum = new_m.OriginalSender
		if !nodeList[incomingNodeNum].isConnected {
			// set up a new connection
			nodeList[incomingNodeNum].openOutgoingConn()
		}
		defer nodeList[incomingNodeNum].closeOutgoingConn()
		err = tcpDecode.Decode(new_m)
		for err == nil {
			// fmt.Println("DECODE m IN receiveIncomingData:", new_m)
			localReceivingChannel <- *new_m
			new_m = new(Message)
			err = tcpDecode.Decode(new_m)
		}
	}
	now := time.Now()
	nanoseconds := float64(now.UnixNano()) / 1e9
	fmt.Printf("%f - Node %d disconnected, %v\n", nanoseconds, incomingNodeNum, err)
}

func handleAllIncomingConns(listener net.Listener) {
	defer listener.Close()
	var conn net.Conn
	var err error = nil
	for err == nil {
		conn, err = listener.Accept()
		go receiveIncomingData(conn) // open up a go routine
	}
	// TODO consider replacing this with panic
	fmt.Println("ERROR receiving incoming connections")
}

func openListener() net.Listener {
	localPort := strings.Split(nodeList[localNodeNum].address, ":")[1]
	listener, err := net.Listen("tcp", ":"+localPort) // open port
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
			Transaction:         text,
			SequenceNumber:      -1,
			TransactionId:       math.MaxUint64,
			IsFinal:             false,
			IsRMulticast:        false,
		}

		localReceivingChannel <- m
	}

}

// TODO figure out how to block until everyone is connected
func waitForAllNodesSync() {
	time.Sleep(5 * time.Second)
	if numConns != numNodes {
		// TODO potentially insert a panic here
		fmt.Println("Failed to establish all expected connections")
		fmt.Println("numConns: %d, numNodes: %d", numConns, numNodes)
	}
}

func setupConnections(hostList []string) {
	var curNodeNum uint8
	nodeList = make([]nodeComms, numNodes)
	for curNodeNum = 0; curNodeNum < numNodes; curNodeNum++ {
		nodeList[curNodeNum].address = hostList[curNodeNum]
	}
	listener := openListener()
	go handleAllIncomingConns(listener)
	for curNodeNum = 0; curNodeNum < numNodes; curNodeNum++ {
		if localNodeNum == curNodeNum {
			nodeList[curNodeNum].conn = nil
			nodeList[curNodeNum].outbox = localReceivingChannel
			nodeList[curNodeNum].isConnected = true
		} else {
			nodeList[curNodeNum].openOutgoingConn()
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

	for incomingMessage := range localReceivingChannel {
		m_ptr := new(Message)
		*m_ptr = incomingMessage
		// fmt.Println("MESSAGE RECEIVED", m_ptr)
		if m_ptr.isAlreadyReceived() {
			continue
		}
		if m_ptr.SenderMessageNumber < 0 { // Handling of a local event

			if m_ptr.OriginalSender != localNodeNum {
				panic("PANIC incomingMessage.OriginalSender != localNodeNum 1")
			}

			nodeList[localNodeNum].senderMessageNum += 1
			m_ptr.SenderMessageNumber = nodeList[localNodeNum].senderMessageNum
			m_ptr.setTransactionId()
			maxProposedSeqNum = findProposalNumber(maxProposedSeqNum, maxFinalSeqNum)

			heap.Push(&pq, NewItem(*m_ptr, maxProposedSeqNum))
			// fmt.Println("Step 1: Local event:", m_ptr)
			bMulticast(*m_ptr)
			continue

		} else { // Handling event received from a different node
			if m_ptr.OriginalSender == localNodeNum {
				fmt.Println(m_ptr)
				fmt.Println("Message Num:", nodeList[localNodeNum].senderMessageNum)
				panic("PANIC  m_ptr.OriginalSender == localNodeNum, we should've filtered this out")
			}

			nodeList[m_ptr.OriginalSender].senderMessageNum = m_ptr.SenderMessageNumber
			if m_ptr.IsRMulticast {
				rMulticast(*m_ptr)
			}
		}
		// delivery of Message to ISIS handler occurs here
		if m_ptr.isProposal() { // Receiving Message 2 and sending Message 3 handled here
			idx := pq.find(m_ptr.TransactionId)
			if idx == math.MaxInt32 {
				panic("FIND RETURNED MAX INDEX 1")
			}
			// update priority in pq = max(proposed priority, local priority)
			pq[idx].priority = max(m_ptr.SequenceNumber, pq[idx].priority)
			pq[idx].value.SequenceNumber = pq[idx].priority
			pq[idx].responsesReceived[m_ptr.OriginalSender] = true

			if allResponsesReceived(pq[idx].responsesReceived) {
				pq[idx].value.IsFinal = true
				m_ptr.IsFinal = true
				m_ptr.OriginalSender = localNodeNum
				nodeList[localNodeNum].senderMessageNum += 1
				m_ptr.SenderMessageNumber = nodeList[localNodeNum].senderMessageNum
				m_ptr.Transaction = pq[idx].value.Transaction
				m_ptr.SequenceNumber = pq[idx].priority
				rMulticast(*m_ptr)
				// fmt.Println("Step 3: rMulticasted Final Sequence : ", m_ptr)
				maxFinalSeqNum = max(m_ptr.SequenceNumber, maxFinalSeqNum)
			}
			heap.Fix(&pq, idx)
			deliverAgreedTransactions(&pq)
		} else if m_ptr.needsProposal() { // Receiving Message 1 and sending Message 2 handled here
			maxProposedSeqNum = findProposalNumber(maxProposedSeqNum, maxFinalSeqNum)
			heap.Push(&pq, NewItem(*m_ptr, maxProposedSeqNum))
			prevSender := m_ptr.OriginalSender
			m_ptr.OriginalSender = localNodeNum
			nodeList[localNodeNum].senderMessageNum += 1
			m_ptr.SenderMessageNumber = nodeList[localNodeNum].senderMessageNum
			m_ptr.SequenceNumber = maxProposedSeqNum
			nodeList[prevSender].unicast(*m_ptr)
			// fmt.Println("Sent Proposal Message 2:", m_ptr)
		} else if m_ptr.IsFinal { // Receiving Message 3 here
			// reorder based on final priority
			// fmt.Println("Receiving Message 3, Agreed on Priority:", m_ptr)
			idx := pq.find(m_ptr.TransactionId)
			if idx == math.MaxInt32 {
				panic("FIND RETURNED MAX INDEX 2")
			}
			pq[idx].priority = m_ptr.SequenceNumber // update priority in pq = final priority
			pq[idx].value = *m_ptr                  // copy the Message with the contents
			heap.Fix(&pq, idx)

			deliverAgreedTransactions(&pq)
			maxFinalSeqNum = max(maxFinalSeqNum, m_ptr.SequenceNumber)
		} else {
			fmt.Println(m_ptr)
			panic("NO CONDITION SATISFIED by above message")
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

func deliverAgreedTransactions(pq_ptr *PriorityQueue) {
	// commit agreed transactions to account
	pq := *pq_ptr
	if len(pq) == 0 {
		return
	}
	m := pq[0].value // highest priority // pq[0] is element with max priority
	for m.IsFinal {
		result := heap.Pop(pq_ptr).(*Item) // TODO: put it into our account balances
		commitNum++
		fmt.Println("Delivering Transaction, commitNum:", commitNum, "Message:", result.value)
		pq := *pq_ptr
		if len(pq) == 0 {
			return
		}
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
		fmt.Println("Expected Format: ./node [number of nodes] [path to file for hostList] [Local Node Number]")
		return
	}
	commitNum = 0
	numConns = 1
	newNumNodes, err := strconv.Atoi(arguments[1])
	check(err)
	hostList := parseHostTextfile(arguments[2])
	newNodeNum, err := strconv.Atoi(arguments[3])
	check(err)
	numNodes = uint8(newNumNodes)
	localNodeNum = uint8(newNodeNum)
	localReceivingChannel = make(chan Message, 65536)
	setupConnections(hostList)
	go handleLocalEventGenerator()
	handleMessageChannel()
}
