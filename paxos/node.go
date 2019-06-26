package paxos

import (
	"fmt"
	"net"
	"net/rpc"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

type rpcAddress string

type SeqState int

// SeqState:
const (
	Decided   SeqState = iota + 1
	Pending            // not yet decided.
	Forgotten          // decided but forgotten.
)

type proposal struct {
	state       SeqState    // proposal state
	proposalID  string      // proposal ID
	acceptID    string      // accepted propose ID
	acceptValue interface{} // accept value
}

// Node Paxos中的運算節點(computing node in Paxos)
type Node struct {
	mutex             sync.Mutex
	netListener       net.Listener
	neighbors         []string
	proposerNodeIndex int   // this node
	dead              int32 // for testing
	unreliable        int32 // for testing
	rpcCount          int32 // for testing

	dones     []int // 紀錄節點同意哪個議案seq
	proposals map[int]*proposal
}

// Done ok to forget all instance <= seq
func (n *Node) Done(seq int) {
	// Your code here.
	n.mutex.Lock()
	defer n.mutex.Unlock()

	if seq > n.dones[n.proposerNodeIndex] {
		n.dones[n.proposerNodeIndex] = seq
	}
}

// Start 發起一個議案, seq是議案的流水號, v是希望被共識的值
func (n *Node) Start(seq int, v interface{}) {
	go func() {
		if seq < n.Min() {
			return
		}
		n.propose(seq, v)
	}()
}

// Min 找出目前被決議的最小議案seq
func (n *Node) Min() int {
	// iterator servers, get min seq
	min := n.dones[n.proposerNodeIndex] // 抓出自己的當作最小
	for i := range n.dones {
		if n.dones[i] < min {
			min = n.dones[i]
		}
	}

	// delete all proposal smaller than min
	for k, proposal := range n.proposals {
		if k > min {
			continue
		}
		if proposal.state != Decided {
			continue
		}

		delete(n.proposals, k)
	}

	return min + 1
}

// Max 找出目前被同意的最大議案
func (n *Node) Max() int {
	max := 0
	for k := range n.proposals {
		if k > max {
			max = k
		}
	}
	return max
}

// Decide receive decide msg handler
func (n *Node) Decide(args *DecideRequest, reply *DecideReply) error {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	_, exist := n.proposals[args.Seq]
	if !exist {
		// learner
		n.proposals[args.Seq] = n.newProposal()
	}

	// update proposer number,accept num and value,state
	n.proposals[args.Seq].acceptValue = args.Value
	n.proposals[args.Seq].acceptID = args.ProposalID
	n.proposals[args.Seq].proposalID = args.ProposalID
	n.proposals[args.Seq].state = Decided
	// update the server done array
	n.dones[args.ProposerNodeIndex] = args.Done

	return nil
}

// Status node can check which proposal with seq that is reached a agreement
func (n *Node) Status(seq int) (SeqState, interface{}) {
	if seq < n.Min() {
		return Forgotten, nil
	}
	n.mutex.Lock()
	defer n.mutex.Unlock()
	proposal, exist := n.proposals[seq]
	if !exist {
		return Pending, nil
	}

	return proposal.state, proposal.acceptValue
}

// generate new proposal
func (n *Node) newProposal() *proposal {
	return &proposal{
		state:       Pending,
		proposalID:  "",
		acceptID:    "",
		acceptValue: nil,
	}
}

// generate a unique proposal num
func (n *Node) genProposalID() string {
	begin := time.Date(2019, time.May, 12, 00, 0, 0, 0, time.UTC)
	duration := time.Now().Sub(begin)
	return strconv.FormatInt(duration.Nanoseconds(), 10) + "-" + strconv.Itoa(n.proposerNodeIndex)
}

// 發送Decide給其他的節點
func (n *Node) sendDecide(seq int, proposalID string, v interface{}) {
	// update seq proposal
	n.mutex.Lock()
	n.proposals[seq].state = Decided
	n.proposals[seq].proposalID = proposalID
	n.proposals[seq].acceptID = proposalID
	n.proposals[seq].acceptValue = v
	n.mutex.Unlock()

	decideReq := DecideRequest{
		Seq:               seq,
		Value:             v,
		ProposalID:        proposalID,
		ProposerNodeIndex: n.proposerNodeIndex,
		Done:              n.dones[n.proposerNodeIndex],
	}
	for i, node := range n.neighbors {
		// send decide to all peers(igonre me)
		if i != n.proposerNodeIndex {
			var reply DecideReply
			call(node, "Node.Decide", &decideReq, &reply)
		}
	}

}

// RPC call
func call(srv string, name string, args interface{}, reply interface{}) bool {
	c, err := rpc.Dial("unix", srv)
	if err != nil {
		err1 := err.(*net.OpError)
		if err1.Err != syscall.ENOENT && err1.Err != syscall.ECONNREFUSED {
			fmt.Printf("paxos Dial() failed: %v\n", err1)
		}
		return false
	}
	defer c.Close()

	err = c.Call(name, args, reply)
	if err == nil {
		return true
	}

	fmt.Println(err)
	return false
}

// 超過半數; 任意兩個majority set必定有共通的成員，故，同一時間不可能有超過一個以上的議案被達成共識
func (n *Node) majority() int {
	return len(n.neighbors)/2 + 1
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
