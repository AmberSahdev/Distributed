package main

import (
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

const MAXNEIGHBORS = 50  // probably too high
const POLLINGPERIOD = 10 // poll neighbors for their neighbors, transactions every POLLINGPERIOD*2 seconds

var localNodeName string             // tracks local node's name
var neighborMap map[string]*nodeComm // undirected graph // var neighborList []nodeComm // undirected graph
var numConns uint8                   // tracks number of other nodes connected to this node for bookkeeping
var localIPaddr string
var localPort string

var mp2ServiceAddr string

var transactionList []*TransactionMessage // List of TransactionMessage // TODO: have a locking mechanism for this bc both handle_service_comms and node.handle_node_comm accessing it
var transactionMap map[string]*TransactionMessage

var neighborMapMutex sync.Mutex
var transactionMapMutex sync.Mutex
var transactionListMutex sync.Mutex

func main() {
	Init_Logging(os.Stdout, os.Stdout, os.Stderr)
	arguments := os.Args
	if len(arguments) != 3 {
		Error.Println("Expected Format: ./gossip [Local Node Name] [port]")
		return
	}

	localNodeName = arguments[1]
	localIPaddr = GetOutboundIP()
	Info.Println("Found local IP to be " + localIPaddr)
	localPort = arguments[2]

	mp2ServiceAddr = parseServiceTextfile("serviceAddr.txt")[0]
	Info.Println("Found MP2 Service Address to be:", mp2ServiceAddr)
	transactionMap = make(map[string]*TransactionMessage)

	neighborMap = make(map[string]*nodeComm)
	neighborMap[localNodeName] = nil // To avoid future errors

	neighborMapMutex = sync.Mutex{}
	transactionMapMutex = sync.Mutex{}
	transactionListMutex = sync.Mutex{}

	go listen_for_conns()
	connect_to_service()
	go handle_service_comms()

	go debug_print_transactions() // TODO: remove later

	// handle the ever updating list of transactions (do any kind of reordering, maintenance work here)
	go blockchain()
	for {
	}
}

/**************************** Setup Functions ****************************/
// from https://www.ardanlabs.com/blog/2013/11/using-log-package-in-go.html
func Init_Logging(
	infoHandle io.Writer,
	warningHandle io.Writer,
	errorHandle io.Writer) {
	Info = log.New(infoHandle,
		"INFO: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Warning = log.New(warningHandle,
		"WARNING: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Error = log.New(errorHandle,
		"ERROR: ",
		log.Ldate|log.Ltime|log.Lshortfile)
}

// Get preferred outbound ip of this machine
func GetOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		Error.Println("Failed to get local IP")
		log.Fatal(err)
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

func connect_to_service() {
	// Connect to mp2_service.py (over TCP)
	var err error = nil
	mp2Service := new(nodeComm)

	mp2Service.nodeName = "mp2Service"
	mp2Service.address = mp2ServiceAddr
	mp2Service.inbox = nil

	mp2Service.conn, err = net.Dial("tcp", mp2Service.address)
	check(err)

	neighborMapMutex.Lock()
	neighborMap["mp2Service"] = mp2Service
	neighborMapMutex.Unlock()
}

/**************************** Go Routines ****************************/
func handle_service_comms() {
	mp2Service := neighborMap["mp2Service"]
	m := "CONNECT " + localNodeName + " " + localIPaddr + " " + localPort + "\n" // Send a message like "CONNECT node1 172.22.156.2 4444"
	Info.Printf("handle_service_comms \t type: %T\n", m)
	_, err := mp2Service.conn.Write([]byte(m)) // sends m over TCP
	check(err)

	// handle messages
	//    1.1 Upto 3 INTRODUCE
	//    1.2 TRANSACTION messages
	//    1.3 QUIT/DIE
	buf := make([]byte, 1024) // Make a buffer to hold incoming data
	for {
		len, err := mp2Service.conn.Read(buf)
		check(err)
		mp2ServiceMsg := strings.Split(string(buf[:len]), "\n")[0]

		msgType := strings.Split(mp2ServiceMsg, " ")[0]
		if msgType == "INTRODUCE" {
			// Example: INTRODUCE node2 172.22.156.3 4567
			print(mp2ServiceMsg, "\n")
			node := new(nodeComm)
			node.nodeName = strings.Split(mp2ServiceMsg, " ")[1]
			node.address = strings.Split(mp2ServiceMsg, " ")[2] + ":" + strings.Split(mp2ServiceMsg, " ")[3]
			node.inbox = make(chan Message, 65536)
			connect_to_node(node)
			neighborMapMutex.Lock()
			neighborMap[node.nodeName] = node
			neighborMapMutex.Unlock()
			go node.handle_node_comm()
		} else if msgType == "TRANSACTION" {
			// Example: TRANSACTION 1551208414.204385 f78480653bf33e3fd700ee8fae89d53064c8dfa6 183 99 10
			//fmt.Println("received mp2_service transaction")
			transactiontime, _ := strconv.ParseFloat(strings.Split(mp2ServiceMsg, " ")[1], 64)
			transactionID := strings.Split(mp2ServiceMsg, " ")[2]
			transactionSrc, _ := strconv.Atoi(strings.Split(mp2ServiceMsg, " ")[3])
			transactionDest, _ := strconv.Atoi(strings.Split(mp2ServiceMsg, " ")[4])
			transactionAmt, _ := strconv.Atoi(strings.Split(mp2ServiceMsg, " ")[5])
			transaction := new(TransactionMessage)
			*transaction = TransactionMessage{transactiontime, transactionID, uint32(transactionSrc), uint32(transactionDest), uint64(transactionAmt)}
			add_transaction(*transaction) // transactionList = append(transactionList, transaction) // TODO: make this more efficient
		} else if (msgType == "QUIT") || (msgType == "DIE") {
			// TODO:
		}
	}
}

func listen_for_conns() {
	var conn net.Conn
	listener, err := net.Listen("tcp", ":"+localPort) // open port
	check(err)
	for err == nil {
		conn, err = listener.Accept()
		node := setup_neighbor(conn)
		go node.handle_node_comm() // open up a go routine
	}
	_ = listener.Close()
	Error.Println("Stopped listening because of error")
	panic(err)
}
