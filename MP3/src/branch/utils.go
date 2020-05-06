package main

import "net"

type clientNode struct {
	conn                     net.Conn
	inbox                    chan string // channel to receive Messages received from neighbor
	outbox                   chan string // channel to send Messages to neighbor
}

import "fmt"

func check(e error) {
	if e != nil {
		fmt.Println("Error Detected:")
		panic(e)
	}
}
