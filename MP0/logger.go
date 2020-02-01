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
	"net"
	"os"
)

func main() {
	arguments := os.Args
	if len(arguments) != 2 {
		fmt.Println("Please provide a port number!")
		return
	}

	port := ":" + arguments[1]
	listener, err := net.Listen("tcp", port)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer listener.Close() // Close after function returns

	for {
		conn, err := listener.Accept()
		_ = err
		// TODO: Expected print statement here of the form form "[time] - [node name] connected"
		//       Example: 1579666871.892629 - node1 connected
		go handleConnection(conn) // open up a goroutine
	}
}

func handleConnection(conn net.Conn) {
	buf := make([]byte, 1024) // Make a buffer to hold incoming data

	for {
		len, err := conn.Read(buf)
		_ = err

		fmt.Print(string(buf[:len]))
		// TODO: Expected print format: [time] [node name] [message]

		// conn.Close() if you read the close signal from somewhere node/ logger?
	}
}

/*
References:
1. https://golang.org/pkg/net/
2. https://opensource.com/article/18/5/building-concurrent-tcp-server-go
3. https://coderwall.com/p/wohavg/creating-a-simple-tcp-server-in-go
5. https://www.linode.com/docs/development/go/developing-udp-and-tcp-clients-and-servers-in-go/
*/
