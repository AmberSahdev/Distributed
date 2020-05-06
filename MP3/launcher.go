package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	// numNodes := 5
	nodes := make(map[string]*exec.Cmd) //nodes := make([]*exec.Cmd, numNodes)

	file, _ := os.Open("./branchAddresses.txt")
	defer file.Close()

	brancheAddresses := make(map[string]string)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		nameAddressPort := strings.Split(scanner.Text(), ":")
		brancheAddresses[nameAddressPort[0]] = nameAddressPort[1] + ":" + nameAddressPort[2]
	}

	buildCmd := exec.Command("make branch")
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	buildCmd.Run()

	// basePort := 2001
	for name, addr := range brancheAddresses {
		fmt.Println("Starting", name, "on addr:", addr)

		node_f, err := os.Create("launcher_logs/" + name + "_stdout.txt")
		node_fe, err := os.Create("launcher_logs/" + name + "_stderr.txt")
		if err != nil {
			fmt.Println("Error node_f")
			fmt.Println(err)
		}

		port := strings.Split(addr, ":")[1]

		cmd := exec.Command("./branch", port)
		cmd.Stdout = node_f
		cmd.Stderr = node_fe
		err = cmd.Start()
		if err != nil {
			fmt.Println("Error cmd.Start()")
			fmt.Println(err)
		}
		nodes[name] = cmd
	}

	for name, _ := range nodes {
		err := nodes[name].Wait()
		if err != nil {
			fmt.Println(err)
		}
	}

}
