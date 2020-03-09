package main

import "net"

type nodeComms struct {
	senderMessageNum int64    //
	address          string   // outgoing node's address:port string
	conn             net.Conn // TODO find out if pass by value or pointer is better here
	outbox           chan BankMessage
	isDead           []bool
	isConnected      bool
}

// Todo define an actual encode & decode method for this and settle on a/many concrete

// types for the decoded result
type Message interface{}

type BankMessage struct {
	OriginalSender      uint8  // local node number of sender of original Transaction
	SenderMessageNumber int64  // index of the event at the process that generated the event
	Transaction         string // sender's Transaction generator string
	TransactionId       uint64 // unique identifier for BankMessage, usually going to be {OriginalSender, SenderMessageNumber[55:0]}, if uninitialized set to -1
	SequenceNumber      int64  // -1 if uninitialized, used for proposal and final
	IsFinal             bool   // distinguishes finalized vs proposed sequence Numbers
	IsRMulticast        bool   // instructs receiver to multicast Message
}

type ConnUpdateMessage struct {
	isConnected bool
	nodeNumber  uint8
}
