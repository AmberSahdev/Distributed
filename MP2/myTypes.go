package main

import (
	"crypto/sha256"
	"net"
)

// ----- Structure to store metadata about nodes introduced to us by mp2_service -----
type TransID [TranSize]byte

type AccountID uint32

type BlockID [sha256.Size]byte

type nodeComm struct {
	nodeName    string
	address     string // neighboring node's address:port string
	conn        net.Conn
	inbox       chan Message // channel to receive Messages received from neighbor
	outbox      chan Message // channel to send Messages to neighbor
	isConnected bool         // Don't need it because we can just remove it from the list on disconnection
}

// ----- Structures for messages received over mp2_service -----
type Message interface{}

type ConnectionMessage struct { // Ex: INTRODUCE node2 172.22.156.3 4567
	NodeName string
	IPaddr   string
	Port     string
}

type TransactionMessage struct { // Ex: TRANSACTION 1551208414.204385 f78480653bf33e3fd700ee8fae89d53064c8dfa6 183 99 10
	Timestamp     float64
	TransactionID TransID // 128-bit unique transaction ID
	Src           AccountID
	Dest          AccountID
	Amount        uint64
}

// ----- Structures for neighbor discovery messages for gossip protocol -----
type DiscoveryMessage struct {
	// if request = true, send back NeighborAddresses.
	// if request = false, you just received NeighborAddresses.
	Request bool
	//NeighborAddresses []ConnectionMessage // list of node's address:port string
}

type DiscoveryReplyMessage struct {
	NodesPendingTransmission        []string
	BlocksPendingTransmission       []BlockID
	TransactionsPendingTransmission []TransID
}

type TransactionRequest struct {
	// if request = true && len(TransactionIDs) == 0, send back all TransactionIDs (TODO: store index of last sent TransactionID)
	// if request = true && len(TransactionIDs) != 0, TransactionIDs has a list of TransactionIDs you need to send TransactionMessages of
	// if request = false, TransactionIDs has a list of TransactionIDs it has (TODO: make it new TransactionIDs)
	//
	Request        bool
	TransactionIDs []TransID
}

/********************************* Blockchain *********************************/
type Block struct {
	blockID         BlockID
	transactions    []TransactionMessage // TODO: you do not need the timestamp in block, make a new struct altogether, or just discard timestamp when you receive it from mp2Service
	parentBlockID   BlockID
	accountBalances map[AccountID]uint64
}
