package paxos

import (
	"log"
	"math/rand"
	"net"
	"net/rpc"
	"os"
	"sync/atomic"
	"syscall"
)

func createNode(addresses []string, me int, rpcsrv *rpc.Server) *Node {

	// init settings
	pxNode := &Node{}
	pxNode.neighbors = addresses
	pxNode.proposerNodeIndex = me

	pxNode.proposals = map[int]*proposal{} // seq: proposal
	pxNode.dones = make([]int, len(pxNode.neighbors))

	for i := range pxNode.neighbors {
		pxNode.dones[i] = -1 // init nodes decided seq
	}
	// end of node settings

	// RPC server settings
	if rpcsrv != nil {
		// caller will create socket &c
		rpcsrv.Register(pxNode)
	} else {
		rpcsrv = rpc.NewServer()
		rpcsrv.Register(pxNode)

		// prepare to receive connections from clients.
		// change "unix" to "tcp" to use over a network.
		os.Remove(addresses[me]) // only needed for "unix"
		l, e := net.Listen("unix", addresses[me])
		if e != nil {
			log.Fatal("listen error: ", e)
		}
		pxNode.netListener = l

		// please do not change any of the following code,
		// or do anything to subvert it.

		// create a thread for RPC connection
		go func() {
			for pxNode.isdead() == false {
				conn, err := pxNode.netListener.Accept()
				if err == nil && pxNode.isdead() == false {
					if pxNode.isunreliable() && (rand.Int63()%1000) < 100 {
						// discard the request.
						conn.Close()
					} else if pxNode.isunreliable() && (rand.Int63()%1000) < 200 {
						// process the request but force discard of reply.
						c1 := conn.(*net.UnixConn)
						f, _ := c1.File()
						err := syscall.Shutdown(int(f.Fd()), syscall.SHUT_WR)
						if err != nil {
							log.Printf("shutdown: %v\n", err)
						}
						atomic.AddInt32(&pxNode.rpcCount, 1)
						go rpcsrv.ServeConn(conn)
					} else {
						atomic.AddInt32(&pxNode.rpcCount, 1)
						go rpcsrv.ServeConn(conn)
					}
				} else if err == nil {
					conn.Close()
				}
				if err != nil && pxNode.isdead() == false {
					log.Printf("Paxos(%v) accept: %v\n", me, err.Error())
				}
			}
		}()
	}

	return pxNode
}
