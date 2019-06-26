package paxos

import (
	"fmt"
)

// Accept acceptor's accept(n, v) handler:
func (n *Node) Accept(args *AcceptRequest, reply *AcceptReply) error {

	n.mutex.Lock()
	defer n.mutex.Unlock()
	proposal, exist := n.proposals[args.Seq]
	if !exist {
		// if not exist,reply OK
		n.proposals[args.Seq] = n.newProposal()
		reply.Accepted = OK
		//fmt.Println(strings.Compare(args.ProposalID, proposal.proposalID))
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
		//fmt.Printf("%s:%d reject accept %v\n", n.neighbors[n.proposerNodeIndex], args.Seq, args.Value)
	}
	return nil
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
