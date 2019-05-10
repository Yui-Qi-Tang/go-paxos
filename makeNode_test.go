package paxos

import (
	"testing"
	"strconv"
	"os"
)


func shutAllDown(nodes []*Node) {
	for i := 0; i < len(nodes); i++ {
		if nodes[i] != nil {
			nodes[i].ShutItselfDown()
		}
	}
}

func getPRCAddr(tag string, host int) string {
	s := "./paxos-nodes/"
	// s += strconv.Itoa(os.Getuid()) + "/"
	os.Mkdir(s, 0777)
	s += "px-"
	s += strconv.Itoa(os.Getpid()) + "-"
	s += tag + "-"
	s += strconv.Itoa(host)
	return s
}

func TestCreateNode(t *testing.T) {

	t.Logf("Test: Create paxos nodes ...\n")
	const numberOfNodes = 3
	nodes := make([]*Node, numberOfNodes)
	rpcAddrs := make([]string, numberOfNodes)
	// defer shutAllDown(nodes)

	// 建立RPC 位置
	for i:=0; i<numberOfNodes; i++ {
		rpcAddrs[i] = getPRCAddr("create_test", i)
	}
	
	// 建立節點
	for i := 0; i < numberOfNodes; i++ {
		nodes[i] = createNode(rpcAddrs, i, nil)
	}

	t.Logf("...Passed\n")

}