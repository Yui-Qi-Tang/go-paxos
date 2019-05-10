package paxos
/* RPC message definition*/


const (
	OK     = "OK"
	Reject = "Reject"
)

// paxos prepare:
type prepareRequest struct {
	Seq  int
	ProposalID string
}

type PrepareReply struct {
	Promise     string
	AcceptedProposalID  string
	AcceptedValue interface{}
}

// paxos accept:
type acceptRequest struct {
	round   int
	proposalID  string
	value interface{}
}

type acceptReply struct {
	accepted string
}

// paxos decide
type decideRequest struct {
	round   int
	value interface{}
	proposalID  string
	proposerNodeIndex    int
	done  int
}

type decideReply struct {
	// this is blank
}
