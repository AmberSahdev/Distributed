/*
Your centralized logger should
  1. listen on a port, specified on a command line,
  2. allow nodes to connect to it and start sending it events.
  3. print out the events, along with the name of the node sending
      the events, to standard out.
(If you want to include diagnostic messages, make sure those are sent to stderr)

You do not need to implement an explicit failure detector;
it is sufficient to create a TCP connection from the nodes to the logger and
have the logger report when it closes.
*/
package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	arguments := os.Args
	if len(arguments) != 2 {
		fmt.Println(os.Stderr, "Please provide a port number!")
		return
	}

	port := arguments[1]
	listener, err := net.Listen("tcp", ":"+port)
	check(err)
	defer listener.Close() // Close after function returns

	fDelay, fBandwidth := create_files()
	//defer fDelay.Close()
	//defer fBandwidth.Close()
	filePointers := [2]*os.File{fDelay, fBandwidth}

	go update_files(filePointers)

	for {
		conn, err := listener.Accept()
		check(err)
		go handleConnection(conn, filePointers) // open up a goroutine
	}
}

func handleConnection(conn net.Conn, filePointers [2]*os.File) {
	buf := make([]byte, 1024) // Make a buffer to hold incoming data

	nodeName := ""
	for {
		inputLen, err := conn.Read(buf)
		now := time.Now()
		nanoseconds := float64(now.UnixNano()) / 1e9
		if err == io.EOF {
			if nodeName == "" {
				nodeName = "Unknown"
			}
			fmt.Printf("%f - ", nanoseconds)
			fmt.Println(nodeName + " disconnected")
			return
		}
		eventList := strings.Split(string(buf[:inputLen]), "\n")
		for i := 0; i < len(eventList); i++ {
			if len(eventList[i]) == 0 {
				if i != len(eventList)-1 {
					panic("Why is this an empty event?")
				}
				continue
			}
			now = time.Now()
			nanoseconds = float64(now.UnixNano()) / 1e9
			if nodeName == "" {
				nodeName = eventList[i]
			}
			contents := strings.Split(eventList[i], " ")
			contentLen := len(contents)
			transmittedTime := 0.0
			fmt.Printf("%f ", nanoseconds)
			if contentLen == 2 {
				transmittedTime, err = strconv.ParseFloat(contents[0], 64)
				check(err)
				fmt.Println(nodeName + " " + contents[1])
			} else if contentLen == 1 {
				fmt.Println("- " + nodeName + " connected")
			} else {
				fmt.Println("\n%d", contentLen)
				panic("Why not be length 1 or 2")
			}

			// Writing to files delay.txt and bandwidth.txt
			filePointers[0].WriteString(fmt.Sprintf("%f ", nanoseconds-transmittedTime))
			filePointers[1].WriteString(fmt.Sprintf("%d ", inputLen))
		}
	}
}

func check(err error) {
	if err != nil {
		fmt.Println(os.Stderr, err)
		panic(err)
	}
}

func create_files() (*os.File, *os.File) {
	fDelay, err := os.Create("delay.txt")
	check(err)
	fBandwidth, err := os.Create("bandwidth.txt")
	check(err)
	return fDelay, fBandwidth
}

func update_files(filePointers [2]*os.File) {
	// Adds a new line to our two files after every second
	oneSecMark1 := time.Now()
	for {
		oneSecMark2 := time.Now()
		if oneSecMark2.Sub(oneSecMark1).Seconds() >= 1 {
			oneSecMark1 = time.Now()
			filePointers[0].WriteString("\n")
			filePointers[1].WriteString("\n")
		}
	}
}

/*
References:
1. https://golang.org/pkg/net/
2. https://opensource.com/article/18/5/building-concurrent-tcp-server-go
3. https://coderwall.com/p/wohavg/creating-a-simple-tcp-server-in-go
5. https://www.linode.com/docs/development/go/developing-udp-and-tcp-clients-and-servers-in-go/
*/
