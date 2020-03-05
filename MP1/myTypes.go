package main

import "net"

type nodeComms struct {
	senderMessageNum int      //
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
	originalSender      int    // local node number of sender of original transaction
	senderMessageNumber int    // index of the event at the process that generated the event
	transaction         string // sender's transaction generator string
	sequenceNumber      int    // -1 if uninitialized, used for proposal and final
	isFinal             bool   // distinguishes finalized vs proposed sequence Numbers
	isRMulticast        bool   // instructs receiver to multicast message
}
