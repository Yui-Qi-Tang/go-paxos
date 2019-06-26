package paxos
/* RPC message definition*/


const (
	OK     = "OK"
	Reject = "Reject"
)

// paxos prepare:
type PrepareRequest struct {
	Seq  int
	ProposalID string
}

type PrepareReply struct {
	Promise     string
	AcceptedProposalID  string
	AcceptedValue interface{}
}

// paxos accept:
type AcceptRequest struct {
	Seq   int
	ProposalID  string
	Value interface{}
}

type AcceptReply struct {
	Accepted string
}

// paxos decide
type DecideRequest struct {
	Seq   int
	Value interface{}
	ProposalID  string
	ProposerNodeIndex    int
	Done  int
}

type DecideReply struct {
	// this is blank
}
