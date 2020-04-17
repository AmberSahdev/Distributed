package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"
)

func main() {
	numPeers, _ := strconv.Atoi(os.Args[1])
	TTL, _ := strconv.Atoi(os.Args[2])
	nodes := make([]*exec.Cmd, numPeers)

	f, err := os.Create("launcher_logs/stdout.txt")
	fe, err := os.Create("launcher_logs/stderr.txt")
	if err != nil {
		fmt.Println(err)
	}

	buildCmd := exec.Command("make")
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	buildCmd.Run()

	base_port := 2000
	for i := 0; i < numPeers; i++ {
		fmt.Println("Starting node", i)
		port := strconv.Itoa(base_port + i)
		nodeName := "node" + strconv.Itoa(i)

		cmd := exec.Command("./main", nodeName, port)
		cmd.Stdout = f
		cmd.Stderr = fe
		err := cmd.Start()
		if err != nil {
			fmt.Println(err)
		}
		nodes[i] = cmd
	}

	go write_to_files(nodes, numPeers)

	time.Sleep(time.Duration(TTL) * time.Second)
	for i := 0; i < numPeers; i++ {
		err := nodes[i].Process.Kill()
		fmt.Printf("Killing node%v\n", i)
		if err != nil {
			fmt.Printf("Failed to kill process %v\n", i)
		}
	}
}

func write_to_files(nodes []*exec.Cmd, numPeers int) {
	for i := 0; i < numPeers; i++ {
		err := nodes[i].Wait()
		if err != nil {
			fmt.Println(err)
		}
	}
}
