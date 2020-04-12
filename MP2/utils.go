package main

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	"net"
	"reflect"
	"strings"
)

/*
func (destNode *nodeComm) unicast(m TransactionMessage) {
	destNode.outbox <- m
}

// Sends TransactionMessage to all our neighbors in neighborList
func bMulticast(m TransactionMessage) {
	// TODO: change to neighborMap
	for _, node := range neighborList {
		node.outbox <- m
	}

}
*/

// Performs our current error handling
func check(e error) {
	if e != nil {
		fmt.Print("\n")
		panic(e)
	}
}

func max(x, y int64) int64 {
	if x > y {
		return x
	}
	return y
}

// Find takes a slice and looks for an element in it. If found it will
// return it's key, otherwise it will return -1 and a bool of false.
func find_transaction(slice []*TransactionMessage, val string) (int, bool) {
	for i, item := range slice {
		if item.TransactionID == val {
			return i, true
		}
	}
	return -1, false
}

func connect_to_node(node *nodeComm) {
	// called when this node is trying to connect to a neighbor after INTRODUCE message
	var err error
	node.conn, err = net.Dial("tcp", node.address)
	check(err) // TODO: maybe dont crash here

	// send ConnectionMessage
	/*
		m := "TRYNA CONNECT UP IN HERE"     // Send a message like "CONNECT node1 172.22.156.2 4444"
		_, err = node.conn.Write([]byte(m)) // sends m over TCP
		check(err)
	*/

	tcpEnc := gob.NewEncoder(node.conn)
	m := ConnectionMessage{
		NodeName: localNodeName,
		IPaddr:   localIPaddr,
		Port:     localPort,
	}
	fmt.Printf("connect_to_node \t type: %T\n", m)
	err = tcpEnc.Encode(m)
	check(err)
}

func (node *nodeComm) tcp_enc_struct(m Message) error {
	//fmt.Println("tcp_enc_struct")
	prefix := []uint8(reflect.TypeOf(m).String() + ":")
	mJSON, err := json.Marshal(m)
	check(err)
	mJSON = append(prefix, mJSON...) // appending a slice to a slice
	_, err = node.conn.Write([]byte(mJSON))
	//check(err) // checked on callee-side
	return err
}

func (node *nodeComm) tcp_dec_struct() (string, string) {
	buf := make([]byte, 1024)
	len, err := node.conn.Read(buf)
	check(err)
	mSlice := strings.SplitN(string(buf[:len]), ":", 2)
	structType := mSlice[0]
	structData := mSlice[1]
	return structType, structData
	/*
		bytes := []byte(structData)
		//var m *Message

		if structType == "main.TransactionRequest" {
			m := new(TransactionRequest)
			err = json.Unmarshal(bytes, m)
			fmt.Println("IN tcp_dec_struct m: ", m)
			check(err)
			return m
		} else {
			m := new(DiscoveryMessage)
			m = nil
			fmt.Println("\n ERROR ERROR ERROR tcp_dec_struct \n")
			return m
		}

		return nil
	*/
}
