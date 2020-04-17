package main

import (
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
)

// from https://www.ardanlabs.com/blog/2013/11/using-log-package-in-go.html
var (
	Debug   *log.Logger
	Info    *log.Logger
	Warning *log.Logger
	Error   *log.Logger
)

const MaxTransactionsInBlock = 2000
const GOSSIPPOLLINGPERIOD = 1000 // Global Gossip Pull Request sent to a node every PollingPeriod ms
const CONNPOLLINGPERIOD = 1000   // polling period for connecting to new nodes.

const TranSize = 16      // transactionID Size in bytes (128 bit IDs)
var localNodeName string // tracks local node's name
var numConns int
var rootBlockID BlockID
var empty struct{} // to avoid ugly struct{}{}

var localIPaddr string
var localPort string

var mp2Service nodeComm

var transactionList []*TransactionMessage // List of TransactionMessage // TODO: have a locking mechanism for this bc both handle_service_comms and node.handle_node_comm accessing it
var transactionMap map[TransID]int
var processedTransactionSet map[TransID]struct{}
var processedTransactionMutex sync.RWMutex
var currentBlockBeingMined *Block
var currentBlockBeingMinedMutex sync.Mutex
var curLongestChainLeafMutex sync.Mutex

var curNeighborToPoll int
var neighborMap map[string]*nodeComm
var neighborList []*nodeComm

var nodeMap map[string]int
var nodeList []*ConnectionMessage

var blockMap map[BlockID]*BlockInfo
var blockList []*Block

var nodeMutex sync.RWMutex
var neighborMutex sync.RWMutex
var transactionMutex sync.RWMutex
var blockMutex sync.RWMutex
var serviceVerifiedBlockIDs chan BlockID
var localVerifiedBlocks chan *Block

func main() {
	initLogging()
	initGob()
	arguments := os.Args
	if len(arguments) != 3 {
		Error.Println("Expected Format: ./main [Local Node Name] [port]")
		return
	}
	localNodeName = arguments[1]
	localIPaddr = GetOutboundIP()
	Info.Println("Found local IP to be " + localIPaddr)
	localPort = arguments[2]
	numConns = 0
	curNeighborToPoll = 0
	mp2ServiceAddr := parseServiceTextfile("serviceAddr.txt")[0]
	Info.Println("Found MP2 Service Address to be:", mp2ServiceAddr)
	transactionMap = make(map[TransID]int)

	serviceVerifiedBlockIDs = make(chan BlockID, 65536)
	localVerifiedBlocks = make(chan *Block, 65536)
	nodeMap = make(map[string]int)
	neighborMap = make(map[string]*nodeComm)

	nodeMap[localNodeName] = 0 // to avoid future errors
	myConn := new(ConnectionMessage)
	*myConn = ConnectionMessage{
		NodeName: localNodeName,
		IPaddr:   localIPaddr,
		Port:     localPort,
	}
	nodeList = append(nodeList, myConn)
	nodeMutex = sync.RWMutex{}
	transactionMutex = sync.RWMutex{}
	blockMutex = sync.RWMutex{}
	neighborMutex = sync.RWMutex{}
	curLongestChainLeafMutex = sync.Mutex{}
	processedTransactionMutex = sync.RWMutex{}

	blockMap = make(map[BlockID]*BlockInfo)
	rootBlockID = [sha256.Size]byte{}
	rootBlockInfo := new(BlockInfo)
	*rootBlockInfo = BlockInfo{
		Index:           0,
		Verified:        true,
		ChildDependents: nil,
	}
	blockMap[rootBlockID] = rootBlockInfo
	// relying on zero value of byte array
	rootAccountBalances := make(map[AccountID]uint64)
	startBlock := new(Block)
	*startBlock = Block{
		ParentBlockID:   rootBlockID,
		Transactions:    nil,
		AccountBalances: rootAccountBalances,
		BlockHeight:     0,
		BlockID:         rootBlockID,
		BlockProof:      [sha256.Size]byte{},
	}
	blockList = append(blockList, startBlock)
	currentBlockBeingMinedMutex = sync.Mutex{}
	currentBlockBeingMined = nil
	localVerifiedBlocks <- startBlock
	go handleIncomingConns()

	go configureGossipProtocol()

	go debugPrintTransactions() // TODO: remove later
	go logging()

	go blockchain()

	handleServiceComms(mp2ServiceAddr)
}

/**************************** Setup Functions ****************************/
// from https://www.ardanlabs.com/blog/2013/11/using-log-package-in-go.html
func initLogging() {
	debugHandle, infoHandle, warningHandle, errorHandle := os.Stdout, os.Stdout, os.Stdout, os.Stderr
	Debug = log.New(debugHandle,
		"DEBUG: ",
		log.Ltime|log.Lshortfile)

	Info = log.New(infoHandle,
		"INFO: ",
		log.Ltime|log.Lshortfile)

	Warning = log.New(warningHandle,
		"WARNING: ",
		log.Ltime|log.Lshortfile)

	Error = log.New(errorHandle,
		"ERROR: ",
		log.Ltime|log.Lshortfile)

}

func initGob() {
	gob.Register(ConnectionMessage{})
	gob.Register(TransactionMessage{})
	gob.Register(DiscoveryMessage{})
	gob.Register(BatchGossipMessage{})
	gob.Register(DiscoveryReplyMessage{})
	gob.Register(GossipRequestMessage{})
}

// Get preferred outbound ip of this machine
func GetOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	check(err)
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	err = conn.Close()
	check(err)
	return localAddr.IP.String()
}

func parseServiceTextfile(path string) []string {
	dat, err := ioutil.ReadFile(path)
	check(err)
	return strings.Split(string(dat), "\n")
}

/**************************** Go Routines ****************************/
func handleServiceComms(mp2ServiceAddr string) {
	var err error = nil
	mp2Service.nodeName = "mp2Service"
	mp2Service.address = mp2ServiceAddr
	mp2Service.inbox = make(chan Message, 65536)
	mp2Service.outbox = make(chan Message, 65536)
	mp2Service.isConnected = true
	mp2Service.conn, err = net.Dial("tcp", mp2Service.address)
	check(err)
	mp2Service.outbox <- "CONNECT " + localNodeName + " " + localIPaddr + " " + localPort + "\n" // Send a message like "CONNECT node1 172.22.156.2 4444"
	go handleServiceSending()
	go handleServiceReceiving()
	for incomingMsg := range mp2Service.inbox {
		// handle messages
		//    1.1 Upto 3 INTRODUCE
		//    1.2 TRANSACTION messages
		//    1.3 QUIT/DIE
		switch mp2ServiceMsg := incomingMsg.(type) {
		case string:
			mp2ServiceMsgArr := strings.Split(mp2ServiceMsg, " ")
			msgType := mp2ServiceMsgArr[0]
			// TODO: replace this with a case statement
			if msgType == "TRANSACTION" {
				// Example: TRANSACTION 1551208414.204385 f78480653bf33e3fd700ee8fae89d53064c8dfa6 183 99 10
				//fmt.Println("received mp2_service transaction")
				var transactionID TransID
				transactionTime, err := strconv.ParseFloat(mp2ServiceMsgArr[1], 64)
				check(err)
				transactionIDSlice, err := hex.DecodeString(mp2ServiceMsgArr[2])
				check(err)
				transactionSrc, err := strconv.Atoi(mp2ServiceMsgArr[3])
				check(err)
				transactionDest, err := strconv.Atoi(mp2ServiceMsgArr[4])
				check(err)
				transactionAmt, err := strconv.Atoi(mp2ServiceMsgArr[5])
				check(err)
				transaction := new(TransactionMessage)
				copy(transactionID[:], transactionIDSlice[:TranSize])
				*transaction = TransactionMessage{transactionTime, transactionID, AccountID(transactionSrc), AccountID(transactionDest), uint64(transactionAmt)}
				transactionMutex.Lock()
				addTransaction(*transaction) // transactionList = append(transactionList, transaction) // TODO: make this more efficient
				transactionMutex.Unlock()
			} else if msgType == "VERIFY" {
				if mp2ServiceMsgArr[1] == "OK" {
					// Block mining verified, push to blockchain thread
					result := new(BlockID)
					blockIDSlice, err := hex.DecodeString(mp2ServiceMsgArr[2])
					check(err)
					copy(result[:], blockIDSlice[:sha256.Size])
					serviceVerifiedBlockIDs <- *result
				} else {
					Warning.Println("Got Rejection of verification for:", mp2ServiceMsgArr[2], mp2ServiceMsgArr[3])
				}
			} else if msgType == "SOLVED" {
				var solvedBlockID BlockID
				var solvedProof BlockPW
				tmp, err := hex.DecodeString(mp2ServiceMsgArr[1])
				check(err)
				copy(solvedBlockID[:], tmp[:sha256.Size])
				tmp, err = hex.DecodeString(mp2ServiceMsgArr[2])
				check(err)
				copy(solvedProof[:], tmp[:sha256.Size])
				currentBlockBeingMinedMutex.Lock()
				if currentBlockBeingMined.BlockID == solvedBlockID {
					Info.Println("Locally Mined Block with ID:", solvedBlockID)
					currentBlockBeingMined.BlockProof = solvedProof
					addBlock(*currentBlockBeingMined, true)
				}
				currentBlockBeingMinedMutex.Unlock()
			} else if msgType == "INTRODUCE" {
				// Example: INTRODUCE node2 172.22.156.3 4567
				Info.Println(mp2ServiceMsg)
				node := new(nodeComm)
				node.nodeName = mp2ServiceMsgArr[1]
				node.address = mp2ServiceMsgArr[2] + ":" + mp2ServiceMsgArr[3]
				newConnMsg := new(ConnectionMessage)
				*newConnMsg = ConnectionMessage{
					NodeName: node.nodeName,
					IPaddr:   mp2ServiceMsgArr[2],
					Port:     mp2ServiceMsgArr[3],
				}
				nodeMutex.Lock()
				addNode(*newConnMsg)
				nodeMutex.Unlock()
				err := connectToNode(node)
				if err != nil {
					continue
				}
				neighborMutex.Lock()
				addNeighbor(node)
				neighborMutex.Unlock()
				go node.handleNodeComm(nil)
			} else if (msgType == "QUIT") || (msgType == "DIE") {
				// TODO: impelment a better quit or die handler
				Error.Println("QUIT or DIE received")
				panic(mp2ServiceMsg)
			}
		default:
			Error.Println("Received non-string data from service")
			panic(mp2ServiceMsg)
		}
	}
}

func handleServiceSending() {
	for incomingMsg := range mp2Service.outbox {
		switch m := incomingMsg.(type) {
		case string:
			_, err := mp2Service.conn.Write([]byte(m)) // sends m over TCP
			if err != nil {                            // NOTE: unnecessary
				// attempt 1 retry, then give up
				_, err := mp2Service.conn.Write([]byte(m)) // sends m over TCP
				check(err)
			}
		default:
			Error.Println("Someone pushed Non-string value to service outbox")
			panic(m)
		}
	}
}

func handleServiceReceiving() {
	buf := make([]byte, 65536) // Make a buffer to hold incoming data
	for {
		msglen, err := mp2Service.conn.Read(buf)
		check(err)
		mp2ServiceMsgs := strings.Split(string(buf[:msglen]), "\n")
		for _, msg := range mp2ServiceMsgs {
			if len(msg) > 0 {
				mp2Service.inbox <- msg
			}
		}
		logBandwidth(nil, msglen)
	}
}

func handleIncomingConns() {
	var conn net.Conn
	listener, err := net.Listen("tcp", ":"+localPort) // open port
	if err == nil {
		for {
			conn, err = listener.Accept()
			if err == nil {
				node, tcpDec := setupNeighbor(conn)
				if node == nil {
					continue
				}
				go node.handleNodeComm(tcpDec) // open up a go routine
			}
		}
	}
	_ = listener.Close()
	Error.Println("Stopped listening because of error")
	panic(err)
}
