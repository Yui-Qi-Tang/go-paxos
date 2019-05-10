package paxos
/* RPC message definition*/



// paxos prepare:
type prepareRequest struct {
	round  int
	proposalID string
}

type prepareReply struct {
	promise     string
	acceptedProposalID  string
	acceptValue interface{}
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
