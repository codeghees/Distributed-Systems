package mme

import (
	"net"
	"net/http"
	"net/rpc"
	"sort"
	"sync"
	"tinyepc/rpcs"
)

const maxHash = ^uint64(0)

type mme struct {
	// TODO: Implement this!
	state      map[uint64]rpcs.MMEState // Stores the net balance for
	myID       uint64                   // Hash assigned to MME from the LB
	replicas   []string                 // List of ports for this MME's replicas
	numServed  int                      // Number of requests served so far
	stateMutex *sync.Mutex
	hostport   string
	ln         net.Listener
}

var operationCosts map[rpcs.Operation]int

// New creates and returns (but does not start) a new MME.
func New() MME {
	// TODO: Implement this!
	operationCosts = make(map[rpcs.Operation]int)
	operationCosts[rpcs.SMS] = -1
	operationCosts[rpcs.Call] = -5
	operationCosts[rpcs.Load] = 10
	m := new(mme)
	m.state = make(map[uint64]rpcs.MMEState)
	m.stateMutex = new(sync.Mutex)
	return m
}

func (m *mme) Close() {
	// TODO: Implement this!
	m.stateMutex.Lock()
	// m.ln.Close()
	m.stateMutex.Unlock()
}

func (m *mme) StartMME(hostPort string, loadBalancer string) error {
	//  Starts MME
	rpcMME := rpcs.WrapMME(m)
	rpc.Register(rpcMME)
	rpc.HandleHTTP()
	ln, err := net.Listen("tcp", hostPort)
	if err != nil {
		return err
	}
	go http.Serve(ln, nil)

	client, dialError := rpc.DialHTTP("tcp", loadBalancer)
	if dialError != nil {
		return dialError
	}
	joinArgs := rpcs.MMEJoinArgs{}
	joinReply := rpcs.MMEJoinReply{}

	joinArgs.Address = hostPort

	client.Call("LoadBalancer.RecvMMEJoin", &joinArgs, &joinReply)
	m.myID = joinReply.Hash
	m.replicas = joinReply.Replicas
	m.hostport = hostPort
	m.ln = ln
	return nil
}

func (m *mme) RecvUERequest(args *rpcs.UERequestArgs, reply *rpcs.UERequestReply) error {
	// Processes UE Request
	user := args.UserID
	req := args.UEOperation

	m.stateMutex.Lock()
	m.numServed++
	balance, userExists := m.state[user]
	if userExists {
		m.state[user] = rpcs.MMEState{Balance: balance.Balance + float64(operationCosts[req])}
	} else {
		m.state[user] = rpcs.MMEState{Balance: 100 + float64(operationCosts[req])}
	}
	m.stateMutex.Unlock()

	return nil
}
func (m *mme) RecvReplica(args *rpcs.SendReplicaArgs, reply *rpcs.SendReplicaReply) error {
	//Receives replicas from LoadBalancer
	m.stateMutex.Lock()
	m.replicas = nil
	m.replicas = args.Replicas
	m.stateMutex.Unlock()
	return nil
}

func (m *mme) RecvState(args *rpcs.RecvStateArgs, reply *rpcs.RecvStateReply) error {
	//Receives state from LoadBalancer
	m.stateMutex.Lock()

	m.state[args.Key] = args.Val

	m.replicas = nil
	m.replicas = args.Replicas
	m.stateMutex.Unlock()
	return nil
}

func (m *mme) SendState(args *rpcs.SendStateArgs, reply *rpcs.SendStateReply) error {
	// Sends State to LoadBalancer
	m.stateMutex.Lock()
	currentState := m.state
	reply.State = currentState

	m.state = make(map[uint64]rpcs.MMEState)
	m.replicas = make([]string, 1)
	m.stateMutex.Unlock()
	return nil
}

// RecvMMEStats is called by the tests to fetch MME state information
// To pass the tests, please follow the guidelines below carefully.
//
// <reply> (type *rpcs.MMEStatsReply) fields must be set as follows:
// 		Replicas: 	List of hostPort strings of replicas
// 					example: [":4110", ":1234"]
// 		NumServed: 	Number of user requests served by this MME
// 					example: 5000
// 		State: 		Map of user states with hash of UserID as key and rpcs.MMEState as value
//					example: 	{
//								"3549791233": {"Balance": 563, ...},
//								"4545544485": {"Balance": 875, ...},
//								"3549791233": {"Balance": 300, ...},
//								...
//								}
func (m *mme) RecvMMEStats(args *rpcs.MMEStatsArgs, reply *rpcs.MMEStatsReply) error {
	// TODO: Implement this!
	m.stateMutex.Lock()
	reply.NumServed = m.numServed
	sort.Strings(m.replicas)

	reply.Replicas = m.replicas
	reply.State = m.state
	m.stateMutex.Unlock()

	return nil
}

// TODO: add additional methods/functions below!
func inRange(key, left, right, max uint64) bool {
	current := left

	for ; current != right; current = (current + 1) % max {
		if current == key {
			return true
		}
	}
	return false
}
