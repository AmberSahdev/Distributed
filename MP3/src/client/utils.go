package main

import (
	"bufio"
	"log"
	"net"
	"os"
	"strings"
)

var (
	Debug   *log.Logger
	Info    *log.Logger
	Warning *log.Logger
	Error   *log.Logger
)

func check(e error) {
	if e != nil {
		Error.Println("Error Detected:")
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
		check(err)
		go pipeConnToInbox(nameAddressPort[0])
	}
	Info.Println("Connected to all branches in branchAddresses.txt")
}

func findInputTarget(input string) string {
	// parse target branch A, or B, or C ... from input
	split := strings.SplitN(input, " ", 2)
	target := strings.SplitN(split[1], ".", 2)
	//Info.Println("input was:", input)
	//Info.Println("extracted target:", target[0])
	return target[0]
}

func pipeConnToInbox(branchName string) {
	conn := branches[branchName]
	buf := make([]byte, 1024)
	for {
		inputLen, err := conn.Read(buf)
		check(err)
		str := string(buf[:inputLen])
		msgArr := strings.Split(str, "\n")
		for _, msg := range msgArr {
			if len(msg) > 0 {
				m := new(Message)
				m.src = branchName
				m.val = msg
				Info.Println("Adding to inbox:", m)
				inbox <- *m
			}
		}
	}
}

func pipeKeyboardToInbox() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		m := Message{"k", scanner.Text()}
		Info.Println("Adding to inbox:", m)
		inbox <- m
	}
}

/*
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
*/

func sendToAll(msg string) {
	// send msg to all nodes. msg can be "ABORT", "COMMIT"
	for branchName := range branches {
		outbox <- Message{branchName, msg} //branchConn.Write([]byte(msg))
	}
}

func allSayCOMMIT_OK() bool {
	i := len(branches)
	ret := true
	for input := range inbox {
		if input.src == "k" {
			continue
		} else if input.val == "COMMIT OK" {
			i--
		} else if input.val == "ABORTED" {
			i--
			ret = false
		} else {
			Error.Println("Error \t input:", input)
			panic("Unexpected message over tcp")
		}

		if i == 0 {
			break
		}
	}
	return ret
}

func handleOutgoingMessages() {
	for m := range outbox {
		Info.Println("adding to outbox:", m)
		_, err := branches[m.src].Write([]byte(m.val + "\n"))
		Debug.Println("Sent Message:", m.val, "to branch:", m.src)
		check(err)
	}
	panic("Outbox has closed!")
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
