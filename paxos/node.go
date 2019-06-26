package paxos

import (
	"sync"
	"sync/atomic"
	"net"
	"time"
	"strconv"
	"net/rpc"
	"syscall"
	"fmt"
)

type rpcAddress string

type SeqState int
// SeqState:
const (
	Decided   SeqState = iota + 1
	Pending        // not yet decided.
	Forgotten      // decided but forgotten.
)


type proposal struct {
	state SeqState        // proposal state
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
	
	dones []int // 紀錄節點同意哪個議案seq
    proposals map[int]*proposal
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

// Prepare for RPC method
// RPC handler which must be satified rule: https://golang.org/pkg/net/rpc/, otherwise ignore
// A proposer chooses a new proposal number n and send a request to
// each member of some set of acceptors, asking it to respond with:
func (n *Node) Prepare(pr *PrepareRequest, reply *PrepareReply) error {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	if proposal, exist := n.proposals[pr.Seq]; !exist {
		// (a) A promise never again to accept a proposal number less than n, and
		n.proposals[pr.Seq] = n.newProposal() // reported no proposals
		proposal, _ = n.proposals[pr.Seq]
		reply.Promise = OK
	} else {
		if pr.ProposalID > proposal.proposalID {
			// (b) The proposal with highest number less than n that it has accepted
			reply.Promise = OK
		} else {
			// reject n less than highest number it has accepted
			reply.Promise = Reject
		}
	}

	// response accepted proposal content

	if reply.Promise == OK {
		proposal := n.proposals[pr.Seq]
		reply.AcceptedProposalID = proposal.proposalID
		reply.AcceptedValue = proposal.acceptValue
		n.proposals[pr.Seq].proposalID = pr.ProposalID
	} else {
		fmt.Printf("%s:%d reject prepare\n", n.neighbors[n.proposerNodeIndex], pr.Seq)
	}
	return nil
}

// Accept acceptor's accept(n, v) handler:
func (n *Node) Accept(args *AcceptRequest, reply *AcceptReply) error {
	
	n.mutex.Lock()
	defer n.mutex.Unlock()

	proposal, exist := n.proposals[args.Seq]
	if !exist {
		// if not exist,reply OK
		n.proposals[args.Seq] = n.newProposal()
		reply.Accepted = OK
	} else {
		if args.ProposalID >= proposal.proposalID {
			reply.Accepted = OK
		} else {
			reply.Accepted = Reject
		}
	}

	if reply.Accepted == OK {
		// update proposer ID,accept ID and accept value
		n.proposals[args.Seq].proposalID = args.ProposalID
		n.proposals[args.Seq].acceptID = args.ProposalID
		n.proposals[args.Seq].acceptValue = args.Value
	} else {
		fmt.Printf("%s:%d reject accept %v\n", n.neighbors[n.proposerNodeIndex], args.Seq, args.Value)
	}
	return nil
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

// 發起提案，送出prepare request
func (n *Node) propose(seq int, v interface{}) {
	decided := false
	// while not decided:
	for !decided {
		// HINT: replyValue 不一定是proposer提出的v，有可能是來自其他proposer被majority acceptors接受的v
		// 透過prepare去得知目前被接受v，西瓜倚大邊XDDD
		gotPromise, proposalID, replyValue := n.sendPrepare(seq, v) //choose n, unique and higher than any n seen so far send prepare(n) to all servers including self
		if gotPromise {                                              // if prepare_ok(n, n_a, v_a) from majority
			// v' = v_a with highest n_a or
			// HINT: proposer會在這邊(獲得promise絕對多數)決定要送出的v是啥; v' 可能是別人的v 也可能是自己的v
			if replyValue == nil {
				//choose own v
				replyValue = v
			}
			// send accept(n, v') to all; if accept_ok(n) from majority
			if n.gotMajorityAccepted(seq, proposalID, replyValue) {
				// send decided(v') to all
				n.sendDecide(seq, proposalID, replyValue) // 收尾的Done有點看不懂，得想想
				decided = true
			}

		}
		state, _ := n.Status(seq)
		if state == Decided {
			decided = true
		}

	} // for

}

// generate new proposal
func (n *Node) newProposal() *proposal {
	return &proposal{
		state: Pending,
		proposalID: "",
		acceptID: "",
		acceptValue: nil,
	}
}

// generate a unique proposal num
func (n *Node) genProposalID() string {
	begin := time.Date(2019, time.May, 12, 00, 0, 0, 0, time.UTC)
	duration := time.Now().Sub(begin)
	return strconv.FormatInt(duration.Nanoseconds(), 10) + "-" + strconv.Itoa(n.proposerNodeIndex)
}

// reutrn bool: got prepare; string: proposal id; interface{}: accept content
func (n *Node) sendPrepare(seq int, v interface{}) (bool, string, interface{}) {
	// v is chozen by proposer,
	myProposalID := n.genProposalID()

	prepareReq := &PrepareRequest{Seq: seq, ProposalID: n.genProposalID()}
	numOfPrepareOK := 0 // numbers of prepare ok
	replyProposalID := ""
	var acceptedValue interface{}
	acceptedValue = nil

	for _, node := range n.neighbors {
		reply := &PrepareReply{AcceptedValue: nil, AcceptedProposalID: "", Promise: Reject}

		call(node, "Node.Prepare", prepareReq, reply) // send prepare request to all nodes

		// collect Promise ok
		if reply.Promise == OK {
			numOfPrepareOK++

			if reply.AcceptedProposalID > replyProposalID {
				replyProposalID = reply.AcceptedProposalID // 關注較大的議案ID
				acceptedValue = reply.AcceptedValue      // 考慮majority set 尚未/已經 獲得accept request -> accept value會是 空的/有值
			}
		}
	}

	return numOfPrepareOK >= n.majority(), myProposalID, acceptedValue

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
		Seq: seq,
		Value: v,
		ProposalID: proposalID,
		ProposerNodeIndex: n.proposerNodeIndex,
		Done: n.dones[n.proposerNodeIndex],
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
    return len(n.neighbors) / 2 + 1
}

// 準備發送Accept給acceptors, 並獲取accepted 數量是否是絕對多數
func (n *Node) gotMajorityAccepted(seq int, proposalID string, v interface{}) bool {
	acceptReq := AcceptRequest{Seq: seq, ProposalID: proposalID, Value: v}
	numOfAcceptedOK := 0

	for _, node := range n.neighbors {
		var acceptorReplay AcceptReply
		call(node, "Node.Accept", &acceptReq, &acceptorReplay)
		if acceptorReplay.Accepted == OK {
			numOfAcceptedOK++
		}
	}

	return numOfAcceptedOK >= n.majority()
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
