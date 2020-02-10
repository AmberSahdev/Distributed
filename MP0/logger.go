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

	for {
		conn, err := listener.Accept()
		check(err)
		go handleConnection(conn) // open up a goroutine
	}
}

func handleConnection(conn net.Conn) {
	buf := make([]byte, 1024) // Make a buffer to hold incoming data

	// nodeName:= make([]byte, 1024)
	// len, err := conn.Read(nodeName)
	// TODO: Expected print statement here of the form form "[time] - [node name] connected"
	//       Example: 1579666871.892629 - node1 connected

	fDelay, fBandwidth := create_files()
	defer fDelay.Close()
	defer fBandwidth.Close()
	nodeName := ""
	oneSecMark1 := time.Now()
	for {
		inputLen, err := conn.Read(buf)
		now := time.Now()
		var nanoseconds = float64(now.UnixNano()) / 1e9
		if err == io.EOF {
			fmt.Printf("%f - ", nanoseconds)
			fmt.Println(nodeName + " disconnected")
			return
		}
		if nodeName == "" {
			nodeName = string(buf[:inputLen])
		}
		contents := strings.Split(string(buf[:inputLen]), " ")
		contentLen := len(contents)
		transmittedTime := 0.0
		message := ""
		if contentLen != 1 {
			transmittedTime, err = strconv.ParseFloat(contents[0], 64)
			check(err)
			message = contents[1]
		}

		fmt.Printf("%f ", nanoseconds)
		if len(message) == 0 {
			fmt.Println("- " + nodeName + " connected")
		} else {
			fmt.Println(nodeName + " " + message)
		}
		// Writing to files delay.txt and bandwidth.txt
		fDelay.WriteString(fmt.Sprintf("%f ", nanoseconds-transmittedTime))
		fBandwidth.WriteString(fmt.Sprintf("%d ", inputLen))
		oneSecMark2 := time.Now()
		if oneSecMark2.Sub(oneSecMark1).Seconds() >= 1 {
			oneSecMark1 = time.Now()
			fDelay.WriteString("\n")
			fBandwidth.WriteString("\n")
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

/*
References:
1. https://golang.org/pkg/net/
2. https://opensource.com/article/18/5/building-concurrent-tcp-server-go
3. https://coderwall.com/p/wohavg/creating-a-simple-tcp-server-in-go
5. https://www.linode.com/docs/development/go/developing-udp-and-tcp-clients-and-servers-in-go/
*/
