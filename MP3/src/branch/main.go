package main

import (
	"fmt"
	"log"
	"net"
	"os"
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
	for incomingMsg := range curNode.inbox {
		// TODO: Parse message, acquire locks, apply update, Rollback!
		Info.Println("Received Message from client:", incomingMsg)
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
