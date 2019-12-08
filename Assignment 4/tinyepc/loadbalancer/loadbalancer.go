package loadbalancer

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/rpc"
	"sort"
	"strconv"
	"sync"
	"tinyepc/rpcs"
)

const maxUint64 = ^uint64(0)

type loadBalancer struct {
	// TODO: Implement this!
	ringNodes     int
	physicalNodes int
	allHashes     []uint64
	serverNames   []string
	ringWeight    int
	hashring      map[uint64]string // Stores MME hashes and their addresses
	replicas      map[uint64]uint64
	hasher        *ConsistentHashing
	stateMutex    *sync.Mutex
	virtualNodes  map[string]int
	nodes         map[uint64]*rpc.Client
}

// New returns a new instance of LoadBalancer, but does not start it
func New(ringWeight int) LoadBalancer {
	// TODO: Implement this!
	lb := new(loadBalancer)
	lb.ringWeight = ringWeight
	lb.hasher = new(ConsistentHashing)
	lb.hashring = make(map[uint64]string)
	lb.replicas = make(map[uint64]uint64)
	lb.hasher.max = maxUint64
	lb.stateMutex = new(sync.Mutex)
	lb.nodes = make(map[uint64]*rpc.Client)
	return lb
}

func (lb *loadBalancer) StartLB(port int) error {
	// TODO: Implement this!
	rpcLB := rpcs.WrapLoadBalancer(lb)
	rpc.Register(rpcLB)
	rpc.HandleHTTP()
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		return err
	}
	go http.Serve(ln, nil)
	return nil
}

func (lb *loadBalancer) Close() {
	// TODO: Implement this!
}

func (lb *loadBalancer) RecvUERequest(args *rpcs.UERequestArgs, reply *rpcs.UERequestReply) error {
	// TODO: Implement this!
	userID := args.UserID
	userHash := lb.hasher.Hash(strconv.Itoa(int(userID)))
	args.UserID = userHash

	lb.stateMutex.Lock()

	lb.sortHashes()
	if userHash > lb.allHashes[lb.ringNodes-1] || userHash <= lb.allHashes[0] {
		err := lb.nodes[lb.allHashes[0]].Call("MME.RecvUERequest", args, reply)
		if err != nil {
			fmt.Println(err)
		}
	} else {
		for _, id := range lb.allHashes {
			if id > userHash {
				err := lb.nodes[id].Call("MME.RecvUERequest", args, reply)
				if err != nil {
					fmt.Println(err)
				}
				break
			}
		}
	}
	lb.stateMutex.Unlock()
	return nil
}

func (lb *loadBalancer) RecvLeave(args *rpcs.LeaveArgs, reply *rpcs.LeaveReply) error {
	// TODO: Implement this!
	return errors.New("RecvLeave() not implemented")
}

func (lb *loadBalancer) RecvMMEJoin(args *rpcs.MMEJoinArgs, reply *rpcs.MMEJoinReply) error {
	lb.stateMutex.Lock()

	lb.ringNodes++
	lb.physicalNodes++
	lb.serverNames = append(lb.serverNames, args.Address)
	hash := lb.hasher.Hash(args.Address)
	lb.allHashes = append(lb.allHashes, hash)

	client, dialError := rpc.DialHTTP("tcp", args.Address)
	if dialError != nil {
		return dialError
	}
	lb.nodes[hash] = client
	lb.hashring[hash] = args.Address
	replicaAddress := lb.getBackupAddress(hash)
	reply.BackupAddress = replicaAddress
	reply.BackupHash = lb.hasher.Hash(replicaAddress)
	lb.replicas[hash] = reply.BackupHash

	for i := 1; i < lb.ringWeight; i++ {
		lb.ringNodes++
		vNodeHash := lb.hasher.VirtualNodeHash(args.Address, i)
		lb.allHashes = append(lb.allHashes, vNodeHash)
		lb.hashring[vNodeHash] = args.Address
		lb.nodes[vNodeHash] = client
	}
	reply.Hash = hash

	lb.stateMutex.Unlock()
	return nil
}

// RecvLBStats is called by the tests to fetch LB state information
// To pass the tests, please follow the guidelines below carefully.
//
// <reply> (type *rpcs.LBStatsReply) fields must be set as follows:
// RingNodes:			Total number of nodes in the hash ring (physical + virtual)
// PhysicalNodes:		Total number of physical nodes ONLY in the ring
// Hashes:				Sorted List of all the nodes'(physical + virtual) hashes
//						e.g. [5655845225, 789123654, 984545574]
// ServerNames:			List of all the physical nodes' hostPort string as they appear in
// 						the hash ring. e.g. [":5002", ":5001", ":5008"]
func (lb *loadBalancer) RecvLBStats(args *rpcs.LBStatsArgs, reply *rpcs.LBStatsReply) error {
	// TODO: Implement this!
	lb.stateMutex.Lock()

	lb.sortHashes()
	reply.Hashes = lb.allHashes
	reply.PhysicalNodes = lb.physicalNodes
	reply.RingNodes = lb.ringNodes
	reply.ServerNames = lb.serverNames

	lb.stateMutex.Unlock()
	return nil
}

// TODO: add additional methods/functions below!
func (lb *loadBalancer) getBackupAddress(id uint64) string {
	var address string
	lb.sortHashes()
	if id >= lb.allHashes[lb.ringNodes-1] {
		address = lb.hashring[lb.allHashes[0]]
	} else {
		var backupHash uint64
		for _, v := range lb.allHashes {
			if v > id {
				backupHash = v
				break
			}
		}
		address = lb.hashring[backupHash]
	}
	return address
}

func (lb *loadBalancer) sortHashes() {
	sort.Slice(lb.allHashes, func(i, j int) bool { return lb.allHashes[i] < lb.allHashes[j] })
}
