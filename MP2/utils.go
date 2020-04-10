package main

func (destNode *nodeComm) unicast(m TransactionMessage) {
	destNode.outbox <- m
}

// Sends TransactionMessage to all our neighbors in neighborList
func bMulticast(m TransactionMessage) {
	for _, node := range neighborList {
		node.outbox <- m
	}
}

// Performs our current error handling
func check(e error) {
	if e != nil {
		panic(e)
	}
}

func max(x, y int64) int64 {
	if x > y {
		return x
	}
	return y
}

func connect_to_node(ip_addr string, port string) nodeComm {

}
