package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
	"strconv"
	"time"
)

var fTransactions *os.File
var fBandwidth *os.File

func logging() {
	fTransactions, fBandwidth = create_logging_files()

	// Adds a new line to our bandwidth file after every resolution seconds
	resolution := 1.0
	for {
		SecondMark1 := time.Now()
		for {
			SecondMark2 := time.Now()
			if SecondMark2.Sub(SecondMark1).Seconds() >= resolution {
				SecondMark1 = time.Now()
				fBandwidth.WriteString("\n")
			}
		}
	}
}

func logTransaction(t TransactionMessage) {
	now := time.Now()
	loggingTime := fmt.Sprintf("%v", float64(now.UnixNano())/1e9)

	tID := fmt.Sprintf("%x", t.TransactionID)
	tTimestamp := fmt.Sprintf("%v", t.Timestamp)
	fTransactions.WriteString(loggingTime + " " + tID + " " + tTimestamp + "\n")
}

func logBandwidth(m *Message, numBytes int) {
	// Format to call it:
	// Either send it a *Message to compute the size of, or send it numBytes
	if m != nil && numBytes == 0 {
		var network bytes.Buffer // Stand-in for a network connection
		enc := gob.NewEncoder(&network)
		err := enc.Encode(*m)
		if err != nil {
			Error.Println("logBandwidth error in encoding to byte buffer")
		}
		numBytes = len(network.Bytes())
	}
	fBandwidth.WriteString(strconv.Itoa(numBytes) + " ")
}

func create_logging_files() (*os.File, *os.File) {
	// fTransactions: format: timeLogged transactionID trasactionID'sTimestamp
	fTransactions, err := os.Create("eval_logs/transactions_" + localNodeName + ".txt")
	check(err)
	// fBandwidth: add a new line every second, each line has a bunch of numbers corresponding to num of bytes written
	fBandwidth, err := os.Create("eval_logs/bandwidth_" + localNodeName + ".txt")
	check(err)
	return fTransactions, fBandwidth
}
