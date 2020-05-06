package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

var branches map[string]net.Conn

func main() {
	fmt.Println("I'm a Client")
	branches = make(map[string]net.Conn)

	connectToBranches()

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		input := scanner.Text()

		if input == "BEGIN" {
			for scanner.Scan() {
				input = scanner.Text()

				if input == "ABORT" {
					// TODO
				} else {
					targetBranchName := findInputTarget(input)
					_, err := branches[targetBranchName].Write([]byte(input))
					check(err)

					for {
						// if received an OK, exit loop. (wait for lock + tcp reply)
						// else see if user typed in "Abort"
					}
				}

			}
		} else {
			//Discard Input
		}
	}

	/*
		1. accept user input of BEGIN, DEPOSIT, BALANCE, WITHDRAW, COMMIT, ABORT
		2. wait for relevant branch to reply "OK", a balance, "NOT FOUND", "ABORTED", "COMMIT OK"
			- if branch replies
		3. repeat
	*/
}
