package paxos

// proposer收到過半數的promise後，發送Accept給acceptors, 並確認獲取accepted 數量是否是絕對多數
func (n *Node) gotMajorityAccepted(seq int, proposalID string, v interface{}) bool {
	acceptReq := AcceptRequest{Seq: seq, ProposalID: proposalID, Value: v}
	numOfAcceptedOK := 0
	// fmt.Println("show proposal =>", n.proposals[seq])
	for _, node := range n.neighbors {
		var acceptorReplay AcceptReply
		call(node, "Node.Accept", &acceptReq, &acceptorReplay)
		if acceptorReplay.Accepted == OK {
			numOfAcceptedOK++
		}
	}

	return numOfAcceptedOK >= n.majority()
}

// 發起提案，送出prepare request
func (n *Node) propose(seq int, v interface{}) {
	decided := false
	// while not decided:
	for !decided {
		// HINT: replyValue 不一定是proposer提出的v，有可能是來自其他proposer被majority acceptors接受的v
		// 透過prepare去得知目前被接受v，西瓜倚大邊XDDD
		gotPromise, proposalID, replyValue := n.sendPrepare(seq, v) //choose n, unique and higher than any n seen so far send prepare(n) to all servers including self
		if gotPromise {                                             // if prepare_ok(n, n_a, v_a) from majority
			//fmt.Println("got promise proposal id:", proposalID)
			// v' = v_a with highest n_a or
			// HINT: proposer會在這邊(獲得promise絕對多數)決定要送出的v是啥; v' 可能是別人的v 也可能是自己的v
			if replyValue == nil {
				//choose own v
				replyValue = v
			}
			// send accept(n, v') to all; if accept_ok(n) from majority
			if n.gotMajorityAccepted(seq, proposalID, replyValue) {
				// send decided(v') to all
				n.sendDecide(seq, proposalID, replyValue)
				decided = true
			}

		}
		state, _ := n.Status(seq)
		if state == Decided {
			decided = true
		}

	} // for

}

// reutrn bool: got prepare; string: proposal id; interface{}: accept content
func (n *Node) sendPrepare(seq int, v interface{}) (bool, string, interface{}) {
	// v is chozen by proposer,
	myProposalID := n.genProposalID()

	prepareReq := &PrepareRequest{Seq: seq, ProposalID: myProposalID}
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
				acceptedValue = reply.AcceptedValue        // 考慮majority set 尚未/已經 獲得accept request -> accept value會是 空的/有值
			}
		}
	}

	return numOfPrepareOK >= n.majority(), myProposalID, acceptedValue

}
