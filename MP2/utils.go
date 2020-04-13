package main

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	"net"
	"reflect"
	"strings"
)

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

	tcpEnc := gob.NewEncoder(node.conn)
	m := ConnectionMessage{
		NodeName: localNodeName,
		IPaddr:   localIPaddr,
		Port:     localPort,
	}
	fmt.Printf("connect_to_node \t type: %T\n\n", m)
	err = tcpEnc.Encode(m)
	check(err)
}

func (node *nodeComm) tcp_enc_struct(m Message) error {
	// Send a struct over TCP node.Conn() using JSON encoding with the prefix of struct's name
	prefix := []uint8(reflect.TypeOf(m).String() + ":")
	mJSON, err := json.Marshal(m)
	check(err)
	mJSON = append(prefix, mJSON...) // appending a slice to a slice

	_, err = node.conn.Write([]byte(mJSON))
	//fmt.Println("\t sent ", string(mJSON))
	//check(err) // checked on callee-side
	return err
}

func (node *nodeComm) tcp_dec_struct(overflowData string) ([]string, []string, string) {
	/*
		Receive data over TCP node.Conn() using JSON encoding as detailed in tcp_enc_struct()
		Input:
			overflowData: data that wasn't a part of a complete struct the last time this function was called
		Output:
			retStructType: a list of struct names, index corresponds to the corresponding struct data in retstructData
			retstructData: list of data in Json format, corresponds to the struct at corresponding index of retStructType
			overflowData: data that was not a part of a complete struct on this conn.Read(). Feed it back in on the next call
	*/
	buf := make([]byte, 1024)
	l, err := node.conn.Read(buf)
	check(err)

	//fmt.Println("\nReceived the following on decoding side with overflowData: ", overflowData+string(buf[:l]))

	// have to do this because TCP coalesces messages
	ListOfMessages := strings.Split(overflowData+string(buf[:l]), "}")

	numMessages := len(ListOfMessages) - 1
	//fmt.Println("\n numMessages ", numMessages)

	retStructType := make([]string, numMessages)
	retstructData := make([]string, numMessages)

	for i := 0; i < numMessages; i++ {
		message := ListOfMessages[i]
		mSlice := strings.SplitN(string(message), ":", 2)
		retStructType[i] = mSlice[0]
		retstructData[i] = mSlice[1] + "}"
	}

	overflowData = ListOfMessages[numMessages] // the last index (the trailing data that wasn't a part of a complete struct)

	//fmt.Println("\n retStructType ", retStructType)
	//fmt.Println("\n retstructData ", retstructData)
	return retStructType, retstructData, overflowData
}
