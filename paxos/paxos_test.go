package paxos

import (
	"testing"
	// "strconv"
	// "os"
	"runtime"
	"time"
)

func ndecided(t *testing.T, nodes []*Node, seq int) int {
	count := 0
	var v interface{}
	for i := 0; i < len(nodes); i++ {
		if nodes[i] != nil {
			decided, v1 := nodes[i].Status(seq)
			if decided == Decided {
				if count > 0 && v != v1 {
					t.Fatalf("decided values do not match; seq=%v i=%v v=%v v1=%v",
						seq, i, v, v1)
				}
				count++
				v = v1
			}
		}
	}
	return count
}

func waitn(t *testing.T, nodes []*Node, seq int, wanted int) {
	to := 10 * time.Millisecond
	for iters := 0; iters < 30; iters++ {
		if ndecided(t, nodes, seq) >= wanted {
			break
		}
		time.Sleep(to)
		if to < time.Second {
			to *= 2
		}
	}
	nd := ndecided(t, nodes, seq)
	if nd < wanted {
		t.Fatalf("too few decided; seq=%v ndecided=%v wanted=%v", seq, nd, wanted)
	}
}

func TestRunPaxos(t *testing.T) {
	runtime.GOMAXPROCS(4)
	t.Logf("Test: Run Paxos ...\n")
	const numberOfNodes = 3
	nodes := make([]*Node, numberOfNodes)
	rpcAddrs := make([]string, numberOfNodes)
	defer shutAllDown(nodes)

	// 建立RPC 位置
	for i := 0; i < numberOfNodes; i++ {
		rpcAddrs[i] = getPRCAddr("create_test", i)
	}

	// 建立節點
	for i := 0; i < numberOfNodes; i++ {
		nodes[i] = createNode(rpcAddrs, i, nil)
	}

	// Start testing
	t.Logf("Single proposer ...\n")
	nodes[0].Start(0, "value")
	waitn(t, nodes, 0, numberOfNodes)
	t.Logf(" ... Passed\n")

	t.Logf("Many proposers, different values ...\n")
	nodes[0].Start(2, 100)
	nodes[1].Start(2, 101)
	nodes[2].Start(2, 102)
	waitn(t, nodes, 2, numberOfNodes)
	t.Logf(" ... Passed\n")

	t.Logf("Out-of-order proposals ...\n")

	nodes[0].Start(7, 700)
	nodes[0].Start(6, 600)
	nodes[1].Start(5, 500)
	waitn(t, nodes, 7, numberOfNodes)
	nodes[0].Start(4, 400)
	nodes[1].Start(3, 300)
	waitn(t, nodes, 6, numberOfNodes)
	waitn(t, nodes, 5, numberOfNodes)
	waitn(t, nodes, 4, numberOfNodes)
	waitn(t, nodes, 3, numberOfNodes)

	if nodes[0].Max() != 7 {
		t.Fatalf("wrong Max()")
	}

	t.Logf(" ... Passed\n")

}
