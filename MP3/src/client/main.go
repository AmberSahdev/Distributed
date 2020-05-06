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
	for input = range inbox {
		if input.val == "BEGIN" && input.src == "k" {
			for input = range inbox {
				if input.val == "ABORT" && input.src == "k" {
					sendToAll("ABORT")
					break
				} else if input.val == "COMMIT" && input.src == "k" {
					sendToAll("CHECK")
					if all_say_COMMIT_OK() {
						sendToAll("COMMIT")
						fmt.Println("COMMIT OK")
					} else {
						sendToAll("ABORT")
						fmt.Println("ABORTED") // waitForAborted()
					}
					break
				} else if input.src == "k" {
					targetBranchName := findInputTarget(input.val)
					outbox <- Message{targetBranchName, input.val}
					for input = range inbox {
						if input.val == "ABORT" && input.src == "k" {
							sendToAll("ABORT")
							fmt.Println("ABORTED") // waitForAborted()
							break
						} else if input.src == targetBranchName {
							fmt.Println(input.val)
							if input.val == "NOT FOUND" {
								sendToAll("ABORT")
								fmt.Println("ABORTED") // waitForAborted()
							}
							break
						} else {
							Error.Println("Error\t input:", input)
							panic("Received message over tcp from unexpected source")
						}
					}
				} else {
					panic("Unforeseen Control Flow")
				}
			}
		} else {
			Info.Println("Discarded input:", input)
		}
	}
}
