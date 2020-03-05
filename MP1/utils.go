package main

// Performs our current error handling
func check(e error) {
	if e != nil {
		panic(e)
	}
}

func (destNode nodeComms) unicast(m Message) {
	destNode.outbox <- m
}

// Pushes outgoing data to all channels so that our outgoing networking threads can push it out to other nodes
func bMulticast(m Message) {
	var i uint8
	for i = 0; i < numNodes; i++ {
		if nodeList[i].isConnected && i != localNodeNum {
			nodeList[i].outbox <- m
		}
	}
}

// Pushes outgoing data to all channels so that our outgoing networking threads can push it out to other nodes
func rMulticast(m Message) {
	m.IsRMulticast = true
	bMulticast(m)
}

func max(x, y int64) int64 {
	if x > y {
		return x
	}
	return y
}
