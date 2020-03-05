package main

import "net"

type nodeComms struct {
	senderMessageNum int64    //
	port             string   // outgoing node's port
	address          string   // outgoing node's address
	conn             net.Conn // TODO find out if pass by value or pointer is better here
	outbox           chan message
	isConnected      bool
}

// Todo define an actual encode & decode method for this and settle on a/many concrete
// types for the decoded result
type message bank_message

type bank_message struct {
	originalSender      uint8  // local node number of sender of original transaction
	senderMessageNumber int64  // index of the event at the process that generated the event
	transaction         string // sender's transaction generator string
	transactionId       uint64 // unique identifier for bank_message, usually going to be {originalSender, senderMessageNumber[55:0]}, if uninitialized set to -1
	sequenceNumber      int64  // -1 if uninitialized, used for proposal and final
	isFinal             bool   // distinguishes finalized vs proposed sequence Numbers
	isRMulticast        bool   // instructs receiver to multicast message
}
