package paxos

import (
	"sync"
	"sync/atomic"
	"net"
)

type rpcAddress string

type pState int
// pState:
const (
	decided   pState = iota + 1
	pending        // not yet decided.
	forgotten      // decided but forgotten.
)


type proposal struct {
	state pState        // proposal state
    proposalID   string      // proposal ID
	acceptID   string      // accepted propose ID
	acceptValue   interface{} // accept value
}

// Node Paxos中的運算節點(computing node in Paxos)
type Node struct {
	mutex sync.Mutex
	netListener  net.Listener
    neighbors []string
	proposerNodeIndex int // this node
	dead       int32 // for testing
	unreliable int32 // for testing
	rpcCount   int32 // for testing
	
	dones []int // 紀錄節點同意哪個seq
    proposals map[int]*proposal
}


func (n *Node) isdead() bool {
	return atomic.LoadInt32(&n.dead) != 0
}

func (n *Node) setunreliable(what bool) {
	if what {
		atomic.StoreInt32(&n.unreliable, 1)
	} else {
		atomic.StoreInt32(&n.unreliable, 0)
	}
}

func (n *Node) isunreliable() bool {
	return atomic.LoadInt32(&n.unreliable) != 0
}

// ShutItselfDown close rpc server and clean fd file
func (n *Node) ShutItselfDown() error {
	atomic.StoreInt32(&n.dead, 1)
	if n.netListener != nil {
		n.netListener.Close()
	}
	return nil
}