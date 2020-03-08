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

//var pq PriorityQueue // used to keep track of priority/sequence number of messages to attain total order rMulticast

func (destNode *nodeComms) communicationTask() {
	// fmt.Println("preparing To Receive m's")
	tcpEnc := gob.NewEncoder(destNode.conn)
	// fmt.Println("Ready To Receive m's")
	for m := range destNode.outbox {
		err := tcpEnc.Encode(m)
		// fmt.Println("ENCODE m IN communicationTask:", m)
		if err != nil {
			fmt.Println("Failed to send Message, receiver down?")
			_ = destNode.conn.Close()
			return
		}
	}
	_ = destNode.conn.Close()
}

func (destNode *nodeComms) openOutgoingConn() {
	var err error
	destNode.conn, err = net.Dial("tcp", destNode.address)
	if err == nil {
		destNode.isConnected = true
		numConns++
		destNode.outbox = make(chan BankMessage, 1024)
		m := BankMessage{
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

// ran when the incoming connection to this node throws and error
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
	var newM *BankMessage
	var incomingNodeNum uint8 = math.MaxUint8
	tcpDecode := gob.NewDecoder(conn)
	newM = new(BankMessage)
	err := tcpDecode.Decode(newM)
	if err == nil {
		incomingNodeNum = newM.OriginalSender
		if !nodeList[incomingNodeNum].isConnected {
			// set up a new connection
			localReceivingChannel <- ConnUpdateMessage{true, incomingNodeNum}
		}
		newM = new(BankMessage)
		err = tcpDecode.Decode(newM)
		for err == nil {
			// fmt.Println("DECODE m IN receiveIncomingData:", new_m)
			localReceivingChannel <- *newM
			newM = new(BankMessage)
			err = tcpDecode.Decode(newM)
		}
	}
	localReceivingChannel <- ConnUpdateMessage{false, incomingNodeNum}
	now := time.Now()
	nanoseconds := float64(now.UnixNano()) / 1e9
	fmt.Printf("\n%f - Node %d disconnected, %v\n", nanoseconds, incomingNodeNum, err)
}

func handleAllIncomingConns(listener net.Listener) {
	var conn net.Conn
	var err error = nil
	for err == nil {
		conn, err = listener.Accept()
		go receiveIncomingData(conn) // open up a go routine
	}
	panic("ERROR receiving incoming connections")
	_ = listener.Close()
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
		m := BankMessage{
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

func allNodesAreConnected() bool {
	return numConns == numNodes
}

func setupConnections(hostList []string) {
	var curNodeNum uint8
	nodeList = make([]nodeComms, numNodes)
	// Populate list of node connection information with string in format of "address:port"
	for curNodeNum = 0; curNodeNum < numNodes; curNodeNum++ {
		nodeList[curNodeNum].address = hostList[curNodeNum]
		nodeList[curNodeNum].senderMessageNum = 0
	}
	listener := openListener()
	go handleAllIncomingConns(listener)
	for curNodeNum = 0; curNodeNum < numNodes; curNodeNum++ {
		if localNodeNum == curNodeNum {
			nodeList[curNodeNum].conn = nil
			nodeList[curNodeNum].outbox = nil
			nodeList[curNodeNum].isConnected = true
		} else {
			nodeList[curNodeNum].openOutgoingConn()
		}
	}
}

func (m *BankMessage) isAlreadyReceived() bool {
	return m.IsRMulticast && nodeList[m.OriginalSender].senderMessageNum >= m.SenderMessageNumber
}

func (m *BankMessage) isLocalProposal() bool {
	return m.SequenceNumber >= 0 && !m.IsFinal && uint8(m.TransactionId>>(64-8)) == localNodeNum
}

func (m *BankMessage) needsProposal() bool {
	return !m.IsFinal && m.SequenceNumber == -1
}

func (m *BankMessage) setTransactionId() {
	m.TransactionId = (uint64(localNodeNum) << (64 - 8)) | (uint64(m.SenderMessageNumber) & 0x00FFFFFFFFFFFFFF) // {OriginalSender, SenderMessageNumber[55:0]}
}

func removeDeadHead(pqPtr *PriorityQueue) {
	pq := *pqPtr
	if len(pq) == 0 {
		return
	}
	headNodeNum := pq[0].value.TransactionId >> 56
	for !nodeList[headNodeNum].isConnected {
		fmt.Println("Sequencer for message at top of the queue died")
		_ = heap.Pop(&pq).(*Item)
		heap.Fix(&pq, 0)
		headNodeNum = pq[0].value.TransactionId >> 56
	}
}

func handleMessageChannel() {
	// Decentralized Causal + Total Ordering Protocol
	pq := make(PriorityQueue, 0)
	var maxFinalSeqNum int64 = 0
	var maxProposedSeqNum int64 = 0
	if allNodesAreConnected() {
		go handleLocalEventGenerator()
	}
	for incoming := range localReceivingChannel {
		switch incomingMessage := incoming.(type) {
		case BankMessage:
			mPtr := new(BankMessage)
			*mPtr = incomingMessage
			if mPtr.isAlreadyReceived() {
				// fmt.Println("Message Already Delivered:", mPtr)
				continue
			}
			// fmt.Println("MESSAGE RECEIVED", mPtr)
			if mPtr.SenderMessageNumber < 0 { // Handling of a local event
				if mPtr.OriginalSender != localNodeNum {
					panic("PANIC mPtr.OriginalSender != localNodeNum 1")
				}
				nodeList[localNodeNum].senderMessageNum += 1
				mPtr.SenderMessageNumber = nodeList[localNodeNum].senderMessageNum
				mPtr.setTransactionId()
				maxProposedSeqNum = findProposalNumber(maxProposedSeqNum, maxFinalSeqNum)
				heap.Push(&pq, NewItem(*mPtr, maxProposedSeqNum))
				// fmt.Println("Step 1: Local event:", mPtr)
				rMulticast(*mPtr)
				continue
			} else { // Handling event received from a different node
				/*
					for i := 0; i < len(pq); i++ {
						fmt.Println("Queue[",i,"]:", pq[i])
					}
				*/
				if mPtr.OriginalSender == localNodeNum {
					//fmt.Println("Message Num:", nodeList[localNodeNum].senderMessageNum)
					//fmt.Println(mPtr)
					panic("PANIC  mPtr.OriginalSender == localNodeNum, we should've filtered this out")
				}
				nodeList[mPtr.OriginalSender].senderMessageNum = mPtr.SenderMessageNumber
				if mPtr.IsRMulticast {
					rMulticast(*mPtr)
				}
			}
			// fmt.Println("Message Received:", mPtr)
			// delivery of Message to ISIS handler occurs here
			if mPtr.isLocalProposal() { // Receiving Message 2 and sending Message 3 handled here
				// fmt.Println("Message Priority Received:", mPtr)
				idx := pq.find(mPtr.TransactionId)
				if idx == math.MaxInt32 {
					panic("FIND RETURNED MAX INDEX 1")
				}
				// update priority in pq = max(proposed priority, local priority)
				pq[idx].priority = max(mPtr.SequenceNumber, pq[idx].priority)
				pq[idx].value.SequenceNumber = pq[idx].priority
				pq[idx].responsesReceived[mPtr.OriginalSender] = true
				if allResponsesReceived(pq[idx].responsesReceived) {
					pq[idx].value.IsFinal = true
					mPtr.IsFinal = true
					mPtr.OriginalSender = localNodeNum
					nodeList[localNodeNum].senderMessageNum += 1
					mPtr.SenderMessageNumber = nodeList[localNodeNum].senderMessageNum
					mPtr.Transaction = pq[idx].value.Transaction
					mPtr.SequenceNumber = pq[idx].priority
					rMulticast(*mPtr)
					// fmt.Println("Step 3: rMulticast Final Sequence :", mPtr)
					maxFinalSeqNum = max(mPtr.SequenceNumber, maxFinalSeqNum)
				}
				heap.Fix(&pq, idx)
				deliverAgreedTransactions(&pq)
			} else if mPtr.needsProposal() { // Receiving Message 1 and sending Message 2 handled here
				maxProposedSeqNum = findProposalNumber(maxProposedSeqNum, maxFinalSeqNum)
				heap.Push(&pq, NewItem(*mPtr, maxProposedSeqNum))
				mPtr.OriginalSender = localNodeNum
				nodeList[localNodeNum].senderMessageNum += 1
				mPtr.SenderMessageNumber = nodeList[localNodeNum].senderMessageNum
				mPtr.SequenceNumber = maxProposedSeqNum
				rMulticast(*mPtr)
				// fmt.Println("Sent Proposal Message 2:", mPtr)
			} else if mPtr.IsFinal { // Receiving Message 3 here
				// reorder based on final priority
				// fmt.Println("Receiving Message 3, Agreed on Priority:", mPtr)
				idx := pq.find(mPtr.TransactionId)
				if idx == math.MaxInt32 {
					panic("FIND RETURNED MAX INDEX 2")
				}
				pq[idx].priority = mPtr.SequenceNumber // update priority in pq = final priority
				pq[idx].value = *mPtr                  // copy the Message with the contents
				heap.Fix(&pq, idx)
				deliverAgreedTransactions(&pq)
				maxFinalSeqNum = max(maxFinalSeqNum, mPtr.SequenceNumber)
			} else {
				// fmt.Println("Message Skipped 2:", mPtr)
			}
		case ConnUpdateMessage:
			if incomingMessage.isConnected {
				nodeList[incomingMessage.nodeNumber].openOutgoingConn()
				if allNodesAreConnected() {
					//fmt.Println("All Nodes Connected, starting!")
					go handleLocalEventGenerator()
				}
			} else {
				nodeList[incomingMessage.nodeNumber].closeOutgoingConn()
			}
		default:
			_, _ = fmt.Fprintf(os.Stderr, "I don't know about type %T!\n", incomingMessage)
		}
		removeDeadHead(&pq)
	}
}

// finds the next proposal number to generate given the final sequence number already seen and max proposal number given
func findProposalNumber(maxProposedSeqNum int64, maxFinalSeqNum int64) int64 {
	if maxFinalSeqNum > maxProposedSeqNum {
		maxProposedSeqNum = maxFinalSeqNum
	}
	return maxProposedSeqNum + 1
}

func deliverAgreedTransactions(pqPtr *PriorityQueue) {
	// commit agreed transactions to account
	pq := *pqPtr
	if len(pq) == 0 {
		return
	}
	m := pq[0].value // highest priority // pq[0] is element with max priority
	for m.IsFinal {
		/*
			result := heap.Pop(pqPtr).(*Item)
			commitNum++
			fmt.Println("Delivering Transaction, commitNum:", commitNum, "Message:", result.value)
		*/
		// Deliver to our application code.
		fmt.Println("\n Popping off PQ")
		update_balances(heap.Pop(pqPtr).(*Item).value)

		pq := *pqPtr
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
	go print_balances()
	handleMessageChannel()
}
