package main

import (
	"fmt"
	"net"
)

var branches map[string]net.Conn
var inbox chan Message
var outbox chan Message

func main() {
	// Setup
	branches = make(map[string]net.Conn)
	inbox = make(chan Message, 1024)
	outbox = make(chan Message, 1024)
	initLogging()

	// Connections
	connectToBranches()
	go handleOutgoingMessages()
	go pipeKeyboardToInbox()
	// Communicate with User and Branches/Servers
	var input Message
	hasBegun := false
	waitingForBranchResponse := false
	waitingForBranchName := ""

	for input = range inbox {
		if input.src == "k" { // Message from keyboard
			if !hasBegun && input.val == "BEGIN" {
				waitingForBranchResponse = false
				waitingForBranchName = ""
				hasBegun = true

			} else if hasBegun && input.val == "ABORT" {
				sendToAll("ABORT")
				fmt.Println("ABORTED")
				hasBegun = false
				waitingForBranchResponse = false
				waitingForBranchName = ""

			} else if hasBegun && !waitingForBranchResponse {
				switch input.val {
				case "COMMIT":
					sendToAll("CHECK")
					if allSayCOMMIT_OK() {
						sendToAll("COMMIT")
						fmt.Println("COMMIT OK")
					} else {
						sendToAll("ABORT")
						fmt.Println("ABORTED") // waitForAborted()
					}
					hasBegun = false
					waitingForBranchResponse = false
					waitingForBranchName = ""

				default:
					targetBranchName := findInputTarget(input.val)
					outbox <- Message{targetBranchName, input.val}
					waitingForBranchResponse = true
					waitingForBranchName = targetBranchName
				}
			} else {
				Info.Println("Discarded input (1):", input)
			}

		} else if input.src != "k" { // Message over TCP
			fmt.Println(input.val)

			// DEBUG
			if waitingForBranchResponse && input.src != waitingForBranchName {
				Error.Println("Error\t input:", input)
				panic("Received message over tcp from unexpected source")
			}

			if input.val == "NOT FOUND" {
				sendToAll("ABORT")
				fmt.Println("ABORTED") // waitForAborted()
				hasBegun = false
			}

			waitingForBranchResponse = false
			waitingForBranchName = ""

		}
	}
}
