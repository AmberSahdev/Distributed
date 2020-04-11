package main

import "net"

// ----- Structure to store metadata about nodes introduced to us by mp2_service -----
type nodeComm struct {
	nodeName string
	address  string // outgoing node's address:port string
	conn     net.Conn
	outbox   chan TransactionMessage // channel to send TransactionMessage received from mp2_service to neighbors
	// isConnected bool // Don't need it because we can just remove it from the list on disconnection
}

// ----- Structures for messages received over mp2_service -----
type ConnectionMessage struct { // Ex: INTRODUCE node2 172.22.156.3 4567
	NodeName string
	IPaddr   string
	Port     string
}

type TransactionMessage struct { // Ex: TRANSACTION 1551208414.204385 f78480653bf33e3fd700ee8fae89d53064c8dfa6 183 99 10
	Timestamp     float64
	TransactionID string // 128-bit unique transaction ID
	Source        uint32
	Dest          uint32
	Amount        uint64
}

// ----- Structures for neighbor discovery messages for gossip protocol -----
type discoveryMessage struct {
	NeighborAddresses []string // list of node's address:port string
}
