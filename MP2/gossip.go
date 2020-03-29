package main

const MAXNEIGHBORS = 50
const POLLINGPERIOD = 10 // poll neighbors for their neighors every 10 seconds

var localNodeNum uint8      // tracks local node's number
var neighborList []nodeComm // undirected graph
var numConns uint8          // tracks number of other nodes connected to this node for bookkeeping
var localReceivingChannel chan TransactionMessage

/* TODO
1. pull gossip
2.
*/

func main() {
}
