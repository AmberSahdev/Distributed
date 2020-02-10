/*
Your node should receive events from the standard input (as sent by the
generator) and send them to the centralized logger.

% python3 -u generator.py 0.1 | node node1 10.0.0.1 1234

% node [name of the node] [address of centralized logging server] [port of centralized logging server] %

This should be the address of your VM running the centralized server (e.g., VM0) and the port.
*/
package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

func main() {
	arguments := os.Args
	if len(arguments) != 4 {
		fmt.Println(os.Stderr, "Expected Format: node [name of the node] [address of centralized logging server] [port of centralized logging server]")
		return
	}
	nodeName := arguments[1] + "\n"
	address := arguments[2]
	port := arguments[3]

	conn, err := net.Dial("tcp", address+":"+port)
	if err != nil {
		fmt.Println(os.Stderr, err)
		panic(err)
	}
	defer conn.Close() // Close after function returns

	// TODO: conn.Write(nodeName) for the logger
	_, err = conn.Write([]byte(nodeName))
	if err != nil {
		panic(err)
	}
	// read stuff from stdin infinitely
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		_, err := conn.Write([]byte(scanner.Text()))
		if err != nil {
			panic(err)
		}
	}
	// TODO: Add exit handling code, i.e. send connection closed message on ctrl+c or
}
