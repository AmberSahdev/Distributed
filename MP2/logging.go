package main

import (
	"fmt"
	"os"
	"time"
)

var fTransactions *os.File
var fBandwidth *os.File

func logging() {
	fTransactions, fBandwidth = create_logging_files()

	// Adds a new line to our bandwidth file after every 5 seconds
	for {
		SecondMark1 := time.Now()
		for {
			SecondMark2 := time.Now()
			if SecondMark2.Sub(SecondMark1).Seconds() >= 5 {
				SecondMark1 = time.Now()
				fBandwidth.WriteString("\n")
			}
		}
	}
}

func logTransaction(t TransactionMessage) {
	now := time.Now()
	loggingTime := fmt.Sprintf("%v ", float64(now.UnixNano())/1e9)

	tID := fmt.Sprintf("%x", t.TransactionID)
	tTimestamp := fmt.Sprintf("%v ", t.Timestamp)
	fTransactions.WriteString(loggingTime + " " + tID + " " + tTimestamp + "\n")
}

func logBandwidth(numBytes int) {
	fBandwidth.WriteString(string(numBytes) + " ")
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
