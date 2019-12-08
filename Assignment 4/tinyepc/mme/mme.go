package mme

import (
	"fmt"
	"net"
	"net/http"
	"net/rpc"
	"sync"
	"tinyepc/rpcs"
)

type mme struct {
	// TODO: Implement this!
	state      map[uint64]rpcs.MMEState // Stores the net balance for
	myID       uint64                   // Hash assigned to MME from the LB
	myBackup   *rpc.Client              // Connection to the MME that is acting as a backup
	backupID   uint64                   // Hash assigned to MME's backup from the LB
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
	fmt.Println("CLOSE MME")

	m.ln.Close()
	m.stateMutex.Unlock()
}

func (m *mme) StartMME(hostPort string, loadBalancer string) error {
	// TODO: Implement this!
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
	backup, backupDialError := rpc.DialHTTP("tcp", joinReply.BackupAddress)
	if backupDialError != nil {
		return backupDialError
	}
	m.myBackup = backup
	m.backupID = joinReply.BackupHash
	m.replicas = append(m.replicas, joinReply.BackupAddress)
	m.hostport = hostPort
	m.ln = ln
	return nil
}

func (m *mme) RecvUERequest(args *rpcs.UERequestArgs, reply *rpcs.UERequestReply) error {
	// TODO: Implement this!
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
	// fmt.Println("MME ID = ", m.myID, "UID = ", user)

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
	fmt.Println(m.hostport, " - ", m.numServed)
	reply.NumServed = m.numServed
	reply.Replicas = m.replicas
	reply.State = m.state
	m.stateMutex.Unlock()
	return nil
}

// TODO: add additional methods/functions below!
