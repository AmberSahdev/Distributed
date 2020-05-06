package main

import (
	"fmt"
	"net"
)

var branches map[string]net.Conn
var inbox chan Message

func main() {
	branches = make(map[string]net.Conn)
	inbox = make(chan Message, 65536)

	connectToBranches()
	go pipeKeyboardToInbox()

	var input Message
	for input = range inbox {
		if input.val == "BEGIN" && input.src == "k" {
			for input = range inbox {
				if input.val == "ABORT" && input.src == "k" {
					sendToAll("ABORT")
					waitForAborted()
					break
				} else if input.val == "COMMIT" && input.src == "k" {
					// 2 stage commit
					sendToAll("COMMIT")
					if all_say_COMMIT_OK() {
						fmt.Println("COMMIT OK")
					} else {
						sendToAll("ROLLBACK")
						fmt.Println("ABORTED")
					}

					break
				} else if input.src == "k" {
					targetBranchName := findInputTarget(input.val)
					_, err := branches[targetBranchName].Write([]byte(input.val))
					check(err)

					for input = range inbox {
						if input.val == "ABORT" && input.src == "k" {
							sendToAll("ABORT")
							waitForAborted()
							break
						} else if input.src == "t" {
							fmt.Println(input.val)
							// TODO check if need to abort on some input
							if input.val == "NOT FOUND" {
								sendToAll("ABORT")
								waitForAborted()
							}
							break
						}
					}
				} else {
					panic("Unforeseen Control Flow")
				}
			}
		} else {
			fmt.Println("Discarded input:", input)
		}
	}
}
