package main

import (
	"os"
	"time"
)

func logging() {
	_, _ = create_files()
}

func create_files() (*os.File, *os.File) {
	// fTransactions: format: timeLogged transactionID trasactionID'sTimestamp
	fTransactions, err := os.Create("data/transactions_" + localNodeName + ".txt")
	check(err)
	// fBandwidth: add a new line every second, each line has a bunch of numbers corresponding to num of bytes written
	fBandwidth, err := os.Create("data/bandwidth_" + localNodeName + ".txt")
	check(err)
	return fTransactions, fBandwidth
}

// Graphing Utils
func update_files(filePointers [2]*os.File) {
	// Adds a new line to our two files after every 5 seconds
	for {
		oneSecMark1 := time.Now()
		for {
			oneSecMark2 := time.Now()
			if oneSecMark2.Sub(oneSecMark1).Seconds() >= 5 {
				oneSecMark1 = time.Now()
				filePointers[0].WriteString("\n")
				filePointers[1].WriteString("\n")
			}
		}
	}
}
