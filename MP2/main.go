package main

import (
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
	Info    *log.Logger
	Warning *log.Logger
	Error   *log.Logger
)

const MAXNEIGHBORS = 12              // probably too high
const POLLINGPERIOD = 10             // poll neighbors for their neighbors, transactions every POLLINGPERIOD*2 seconds
const TranSize = 16                  // transactionID Size in bytes (128 bit IDs)
var localNodeName string             // tracks local node's name
var neighborMap map[string]*nodeComm // undirected graph // var neighborList []nodeComm // undirected graph
var numConns uint8                   // tracks number of other nodes connected to this node for bookkeeping
var localIPaddr string
var localPort string

var mp2Service nodeComm

var transactionList []*TransactionMessage // List of TransactionMessage // TODO: have a locking mechanism for this bc both handle_service_comms and node.handle_node_comm accessing it
var transactionMap map[TransID]*TransactionMessage

var neighborMapMutex sync.Mutex
var transactionMapMutex sync.Mutex
var transactionListMutex sync.Mutex

func main() {
	initLogging(os.Stdout, os.Stdout, os.Stderr)
	initGob()
	arguments := os.Args
	if len(arguments) != 3 {
		Error.Println("Expected Format: ./gossip [Local Node Name] [port]")
		return
	}
	numConns = 0
	localNodeName = arguments[1]
	localIPaddr = GetOutboundIP()
	Info.Println("Found local IP to be " + localIPaddr)
	localPort = arguments[2]

	mp2ServiceAddr := parseServiceTextfile("serviceAddr.txt")[0]
	Info.Println("Found MP2 Service Address to be:", mp2ServiceAddr)
	transactionMap = make(map[TransID]*TransactionMessage)

	neighborMap = make(map[string]*nodeComm)
	neighborMap[localNodeName] = nil // To avoid future errors

	neighborMapMutex = sync.Mutex{}
	transactionMapMutex = sync.Mutex{}
	transactionListMutex = sync.Mutex{}

	go handleIncomingConns()

	go configureGossipProtocol()

	go debugPrintTransactions() // TODO: remove later

	handleServiceComms(mp2ServiceAddr)
}

/**************************** Setup Functions ****************************/
// from https://www.ardanlabs.com/blog/2013/11/using-log-package-in-go.html
func initLogging(infoHandle io.Writer, warningHandle io.Writer, errorHandle io.Writer) {
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
	gob.Register(TransactionRequest{})
}

// Get preferred outbound ip of this machine
func GetOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		Error.Println("Failed to get local IP")
		panic(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

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
			msgType := strings.Split(mp2ServiceMsg, " ")[0]
			// TODO: replace this with a case statement
			if msgType == "TRANSACTION" {
				// Example: TRANSACTION 1551208414.204385 f78480653bf33e3fd700ee8fae89d53064c8dfa6 183 99 10
				//fmt.Println("received mp2_service transaction")
				var transactionID TransID
				mp2ServiceMsgArr := strings.Split(mp2ServiceMsg, " ")
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
				addTransaction(*transaction) // transactionList = append(transactionList, transaction) // TODO: make this more efficient
			} else if msgType == "INTRODUCE" {
				// Example: INTRODUCE node2 172.22.156.3 4567
				print(mp2ServiceMsg, "\n")
				node := new(nodeComm)
				node.nodeName = strings.Split(mp2ServiceMsg, " ")[1]
				node.address = strings.Split(mp2ServiceMsg, " ")[2] + ":" + strings.Split(mp2ServiceMsg, " ")[3]
				connectToNode(node) // TODO: do handle node comms inside this routine
				neighborMapMutex.Lock()
				neighborMap[node.nodeName] = node
				neighborMapMutex.Unlock()
				go node.handleNodeComm()
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
	}
}

func handleIncomingConns() {
	var conn net.Conn
	listener, err := net.Listen("tcp", ":"+localPort) // open port
	check(err)
	for err == nil {
		conn, err = listener.Accept()
		node := setupNeighbor(conn)
		go node.handleNodeComm() // open up a go routine
	}
	_ = listener.Close()
	Error.Println("Stopped listening because of error")
	panic(err)
}
