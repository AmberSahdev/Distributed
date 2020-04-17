package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"
)

func main() {
	numNodes, _ := strconv.Atoi(os.Args[1])
	TTL, _ := strconv.Atoi(os.Args[2])
	basePort, _ := strconv.Atoi(os.Args[3])
	nodes := make([]*exec.Cmd, numNodes)

	buildCmd := exec.Command("make")
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	buildCmd.Run()

	// basePort := 2001
	for i := 0; i < numNodes; i++ {
		port := strconv.Itoa(basePort + i)
		nodeName := "node" + strconv.Itoa(i+1)
		fmt.Println("Starting", nodeName, "on port:", port)

		node_f, err := os.Create("launcher_logs/" + nodeName + "_stdout.txt")
		node_fe, err := os.Create("launcher_logs/" + nodeName + "_stderr.txt")
		if err != nil {
			fmt.Println("Error node_f")
			fmt.Println(err)
		}

		cmd := exec.Command("./main", nodeName, port)
		cmd.Stdout = node_f
		cmd.Stderr = node_fe
		err = cmd.Start()
		if err != nil {
			fmt.Println("Error cmd.Start()")
			fmt.Println(err)
		}
		nodes[i] = cmd
	}

	go write_to_files(nodes, numNodes)

	time.Sleep(time.Duration(TTL) * time.Second)
	for i := 0; i < numNodes; i++ {
		err := nodes[i].Process.Kill()
		fmt.Printf("Killing node%v\n", i+1)
		if err != nil {
			fmt.Printf("Failed to kill process %v\n", i)
		}
	}
}

func write_to_files(nodes []*exec.Cmd, numNodes int) {
	for i := 0; i < numNodes; i++ {
		err := nodes[i].Wait()
		if err != nil {
			fmt.Println(err)
		}
	}
}
