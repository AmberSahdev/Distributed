package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

func check(e error) {
	if e != nil {
		fmt.Println("Error Detected:")
		panic(e)
	}
}

func connectToBranches() {
	file, err := os.Open("../branchAddresses.txt")
	check(err)
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		nameAddressPort := strings.Split(scanner.Text(), ":")
		branches[nameAddressPort[0]], err = net.Dial("tcp", nameAddressPort[1]+":"+nameAddressPort[2])
		// TODO: make goroutine to process input into a channel
		check(err)
	}
	fmt.Println("Connected to all branches in branchAddresses.txt")
}

func findInputTarget(input string) string {
	// TODO: parse target branch A, or B, or C ... from input
	return ""
}
