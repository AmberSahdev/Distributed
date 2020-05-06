package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

func check(e error) {
	if e != nil {
		fmt.Println("Error Detected:")
		panic(e)
	}
}

func connectToBranches() {
	file, err := os.Open("./branchAddresses.txt")
	check(err)
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		nameAddressPort := strings.Split(scanner.Text(), ":")
		branches[nameAddressPort[0]], err = net.Dial("tcp", nameAddressPort[1]+":"+nameAddressPort[2])
		// TODO: make goroutine to process input into a channel
		go pipeConnToInbox(branches[nameAddressPort[0]])
		check(err)
	}
	fmt.Println("Connected to all branches in branchAddresses.txt")
}

func findInputTarget(input string) string {
	// TODO: parse target branch A, or B, or C ... from input
	return ""
}

func pipeConnToInbox(conn net.Conn) {
	buf := make([]byte, 1024)
	for {
		inputLen, err := conn.Read(buf)
		check(err)
		msg := Message{"t", string(buf[:inputLen])}
		inbox <- msg
	}
}

func pipeKeyboardToInbox() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		msg := Message{"k", string(scanner.Text())}
		inbox <- msg
	}
}

func waitForAborted() {
	i := len(branches)
	for input := range inbox {
		if input.src == "k" {
			continue
		}
		if input.src == "t" && input.val == "ABORTED" {
			i--
		} else {
			panic("Unexpected message over tcp")
		}

		if i == 0 {
			break
		}
	}
	fmt.Println("ABORTED")
}

func sendToAll(msg string) {
	// send msg to all nodes. msg can be "ABORT", "COMMIT", "ROLLBACK"
	for _, branchConn := range branches {
		branchConn.Write([]byte(msg))
	}
}

func all_say_COMMIT_OK() bool {
	i := len(branches)
	for input := range inbox {
		if input.src == "k" {
			continue
		}
		if input.src == "t" && input.val == "COMMIT OK" {
			i--
		} else if input.src == "t" && input.val == "ABORTED" {
			return false
		} else {
			panic("Unexpected message over tcp")
		}

		if i == 0 {
			break
		}
	}
	return true
}
