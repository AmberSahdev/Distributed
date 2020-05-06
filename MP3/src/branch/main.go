package main

import (
	"fmt"
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
var branchName string
var localPort string
var localAccounts map[string]*Account
var localAccountsMutex sync.RWMutex
var clientCounter int

func main() {
	fmt.Println("I'm a Branch")
	initLogging()
	arguments := os.Args
	if len(arguments) != 3 {
		Error.Println("Expected Format: ./main [Local Node Name] [port]")
		return
	}
	branchName = arguments[1]
	localPort = arguments[2]
	localAccounts = make(map[string]*Account)
	handleIncomingConns()
}

func (curNode *clientNode) receiveIncomingMessages() {
	var buf [1024]byte
	for {
		len, err := curNode.conn.Read(buf[:])
		if err != nil {
			Error.Println("Failed to read message from Client!")
			break
		}
		msg := string(buf[:len])
		curNode.inbox <- msg
	}
	Error.Println("No longer receiving from client!")
	close(curNode.inbox)
	curNode.conn.Close()
	return
}

func (curNode *clientNode) sendOutgoingMessages() {
	for msg := range curNode.outbox {
		_, err := curNode.conn.Write([]byte(msg))
		if err != nil {
			_, err := curNode.conn.Write([]byte(msg))
			if err != nil {
				Error.Println("Failed to send message:", msg)
			}
		}
	}
	Error.Println("No Longer sending Messages to client!")
	return
}

func handleClientComm(conn net.Conn) {
	curNode := clientNode{conn, make(chan string, 1024), make(chan string, 1024)}
	go curNode.receiveIncomingMessages()
	go curNode.sendOutgoingMessages()
	isWriteLockedAccount := make(map[string]bool) // if a lock is in the map it is at least a read lock. if true, write lock
	balanceChanges := make(map[string]int)        // tracks net changes in account balances during transactions!
	for incomingMsg := range curNode.inbox {
		// TODO: Parse message, acquire locks, apply update, Rollback!
		incomingMsgArr := strings.Split(incomingMsg, " ")
		switch incomingMsgArr[0] {
		case "DEPOSIT":
			accNameArr := strings.Split(incomingMsgArr[1], ".")
			accName := accNameArr[1]
			change, err := strconv.Atoi(incomingMsgArr[2])
			check(err)
			localAccountsMutex.Lock()
			curAccount, exist := localAccounts[accName]
			if !exist {
				curAccount = new(Account) // relying on zero values for initialization
				localAccounts[accName] = curAccount
			}
			localAccountsMutex.Unlock()
			if canWrite, canRead := isWriteLockedAccount[accName]; canRead {
				if !canWrite {
					curAccount.Lock.PromoteLock()
				}
			} else {
				curAccount.Lock.Lock()
			}
			isWriteLockedAccount[accName] = true
			balanceChanges[accName] += change
			curAccount.Balance += change
		case "WITHDRAW":
			accNameArr := strings.Split(incomingMsgArr[1], ".")
			accName := accNameArr[1]
			change, err := strconv.Atoi(incomingMsgArr[2])
			check(err)
			localAccountsMutex.Lock()
			curAccount, exist := localAccounts[accName]
			if !exist {
				localAccountsMutex.Unlock()
				curNode.outbox <- "NOT FOUND"
				continue
			}
			localAccountsMutex.Unlock()
			if canWrite, canRead := isWriteLockedAccount[accName]; canRead {
				if !canWrite {
					curAccount.Lock.PromoteLock()
				}
			} else {
				curAccount.Lock.Lock()
			}
			isWriteLockedAccount[accName] = true
			balanceChanges[accName] -= change
			curAccount.Balance -= change
		case "BALANCE":
			// TODO: Get Read Lock on account
			// TODO: return account balance
			accNameArr := strings.Split(incomingMsgArr[1], ".")
			accName := accNameArr[1]
			localAccountsMutex.Lock()
			curAccount, exist := localAccounts[accName]
			if !exist {
				localAccountsMutex.Unlock()
				curNode.outbox <- "NOT FOUND"
				continue
			}
			localAccountsMutex.Unlock()
			if _, canRead := isWriteLockedAccount[accName]; !canRead {
				curAccount.Lock.RLock()
				isWriteLockedAccount[accName] = false
			}
			curNode.outbox <- incomingMsgArr[1] + " = " + string(curAccount.Balance)
		case "ABORT":
			// TODO: release all acquired locks
			// only need read lock on map itself
			localAccountsMutex.RLock()
			for accName, balanceChange := range balanceChanges {
				localAccounts[accName].Balance -= balanceChange
			}
			localAccountsMutex.RUnlock()
			balanceChanges = make(map[string]int)
			localAccountsMutex.RLock()
			for accName, canWrite := range isWriteLockedAccount {
				if canWrite {
					localAccounts[accName].Lock.Unlock()
				} else {
					localAccounts[accName].Lock.RUnlock()
				}
			}
			localAccountsMutex.RUnlock()
			isWriteLockedAccount = make(map[string]bool)
		case "CHECK":
			localAccountsMutex.Lock()
			hasBadAccountBalance := false
			for accName := range balanceChanges {
				if localAccounts[accName].Balance < 0 {
					hasBadAccountBalance = true
					break
				}
			}
			if hasBadAccountBalance {
				curNode.outbox <- "ABORTED"
			} else {
				curNode.outbox <- "COMMIT OK"
			}
			localAccountsMutex.Unlock()
			// TODO: ensure account balances are positive
		case "COMMIT":
			// TODO: release all acquired locks and reset maps
			balanceChanges = make(map[string]int)
			localAccountsMutex.RLock()
			for accName, canWrite := range isWriteLockedAccount {
				if canWrite {
					localAccounts[accName].Lock.Unlock()
				} else {
					localAccounts[accName].Lock.RUnlock()
				}
			}
			localAccountsMutex.RUnlock()
			isWriteLockedAccount = make(map[string]bool)
		default:
			Error.Println("Unknown value sent by client!")
		}
	}
	close(curNode.outbox)
	return
}

func handleIncomingConns() {
	var conn net.Conn
	listener, err := net.Listen("tcp", ":"+localPort) // open port
	defer listener.Close()
	if err == nil {
		for {
			conn, err = listener.Accept()
			if err != nil {
				Error.Println("Failed to Connect to a Client!")
			} else {
				go handleClientComm(conn) // open up a go routine
			}
		}
	}
	Error.Println("Stopped listening because of error")
	panic(err)
}

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
