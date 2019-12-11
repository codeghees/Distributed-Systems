package loadbalancer

import (
	"net"
	"net/http"
	"net/rpc"
	"sort"
	"strconv"
	"sync"
	"tinyepc/rpcs"
)

const maxUint64 = ^uint64(0)

/*
Roll No: 20100093-20100286
IMPLEMENTATION DETAILS:
	- Once the LoadBalancer starts, it listens to all MME joins and recomputes the state and replicas for the system; it sends back the state/replicas to MMEs.
	- Successors are found using the sorted HashList we have maintained.
	- We have an RPC map for all MMEs used to call respective functions.

*/
type loadBalancer struct {
	// TODO: Implement this!
	ringNodes     int                       // Total Number of nodes
	physicalNodes int                       // Total number of Physical nodes
	allHashes     []uint64                  // Sorted list of all Hashes
	serverNames   []string                  // HostPort strings of all Hashes
	ringWeight    int                       // Weight of a ring
	hashring      map[uint64]string         // Stores MME hashes and their addresses
	virtualNodes  map[string][]uint64       // Stores all virtual node hashes for a physical node
	replicas      map[string]map[string]int // Map of HostPort: Replicas
	hashes        map[uint64]bool           // Truth Map of Hashes generated
	hasher        *ConsistentHashing
	stateMutex    *sync.Mutex
	nodes         map[uint64]*rpc.Client // RPC client of all nodes of Phy+Virtual nodes
	phyNodes      map[uint64]*rpc.Client // RPC Client of all physical nodes
	hostport      string
	ln            net.Listener
}

// New returns a new instance of LoadBalancer, but does not start it
func New(ringWeight int) LoadBalancer {
	// TODO: Implement this!
	lb := new(loadBalancer)
	lb.ringWeight = ringWeight
	lb.hasher = new(ConsistentHashing)
	lb.hashring = make(map[uint64]string)
	lb.virtualNodes = make(map[string][]uint64)
	lb.replicas = make(map[string]map[string]int)
	lb.hasher.max = maxUint64
	lb.stateMutex = new(sync.Mutex)
	lb.nodes = make(map[uint64]*rpc.Client)
	lb.phyNodes = make(map[uint64]*rpc.Client)
	lb.hashes = make(map[uint64]bool)
	return lb
}

func (lb *loadBalancer) StartLB(port int) error {
	// - Starts the Server
	rpcLB := rpcs.WrapLoadBalancer(lb)
	rpc.Register(rpcLB)
	rpc.HandleHTTP()
	lb.hostport = ":" + strconv.Itoa(port)
	ln, err := net.Listen("tcp", lb.hostport)
	lb.ln = ln
	if err != nil {
		return err
	}
	go http.Serve(ln, nil)
	return nil
}

func (lb *loadBalancer) Close() {
	lb.ln.Close()
	// TODO: Implement this!
}

func (lb *loadBalancer) RecvUERequest(args *rpcs.UERequestArgs, reply *rpcs.UERequestReply) error {
	// TODO: Implement this!
	userID := args.UserID
	userHash := lb.hasher.Hash(strconv.Itoa(int(userID)))
	args.UserID = userHash

	lb.stateMutex.Lock()
	owner := lb.FindSuccessor(userHash)
	lb.nodes[owner].Call("MME.RecvUERequest", args, reply)
	lb.stateMutex.Unlock()
	return nil
}

func (lb *loadBalancer) RecvLeave(args *rpcs.LeaveArgs, reply *rpcs.LeaveReply) error {
	// TODO: Implement this!
	lb.stateMutex.Lock()
	completeState := make(map[uint64]rpcs.MMEState)
	for _, mme := range lb.allHashes {
		sendStateArgs := rpcs.SendStateArgs{}
		sendStateReply := rpcs.SendStateReply{}
		sendStateArgs.LBHostport = lb.hostport

		lb.nodes[mme].Call("MME.SendState", &sendStateArgs, &sendStateReply)

		for k, v := range sendStateReply.State {
			completeState[k] = v
		}
	}

	DeleteMMEHash := lb.virtualNodes[args.HostPort]
	DeleteMMEHash = append(DeleteMMEHash, lb.hasher.Hash(args.HostPort))
	var newhashlist []uint64
	for i := 0; i < len(lb.allHashes); i++ {
		Delete := false
		for j := 0; j < len(DeleteMMEHash); j++ {
			if lb.allHashes[i] == DeleteMMEHash[j] {
				Delete = true
				lb.ringNodes--
				_, exists := lb.phyNodes[DeleteMMEHash[j]]
				if exists {
					lb.physicalNodes--
					delete(lb.phyNodes, DeleteMMEHash[j])
				}
				break
			}
		}
		if Delete == false {
			newhashlist = append(newhashlist, lb.allHashes[i])
		}

	}

	lb.allHashes = newhashlist
	lb.sortHashes()
	//__________________RECOMPUTE replicas___________________//

	for _, mmeHash := range lb.allHashes {
		mmeAddress := lb.hashring[mmeHash]
		mmeReplicaHash := lb.FindSuccessor(mmeHash)
		mmeReplicaAddress := lb.hashring[mmeReplicaHash]
		lb.replicas[mmeAddress] = make(map[string]int)
		if lb.physicalNodes > 1 {
			for {
				if mmeAddress == mmeReplicaAddress && (mmeAddress != args.HostPort) {
					mmeReplicaHash = lb.FindSuccessor(mmeReplicaHash)
					mmeReplicaAddress = lb.hashring[mmeReplicaHash]
				} else {
					lb.replicas[mmeAddress][mmeReplicaAddress] = 1
					break
				}
			}

		}
	}

	//________________RECOMPUTE REPLICAS_____________________//
	for k, v := range completeState {
		ownerMME := lb.FindSuccessor(k)
		mmeReplicas := lb.replicas[lb.hashring[ownerMME]]
		recvStateArgs := rpcs.RecvStateArgs{}
		recvStateArgs.Key = k
		recvStateArgs.Val = v
		recvStateReply := rpcs.RecvStateReply{}
		var updatedReplicas []string
		for k := range mmeReplicas {
			updatedReplicas = append(updatedReplicas, k)
		}
		recvStateArgs.Replicas = updatedReplicas
		updatedReplicas = nil

		lb.nodes[ownerMME].Call("MME.RecvState", &recvStateArgs, &recvStateReply)
	}
	lb.stateMutex.Unlock()
	return nil
}

func (lb *loadBalancer) RecvMMEJoin(args *rpcs.MMEJoinArgs, reply *rpcs.MMEJoinReply) error {
	lb.stateMutex.Lock()
	lb.ringNodes++
	lb.physicalNodes++
	var replicas []string
	lb.serverNames = append(lb.serverNames, args.Address)
	hash := lb.hasher.Hash(args.Address)
	lb.replicas[args.Address] = make(map[string]int)

	lb.allHashes = append(lb.allHashes, hash)
	lb.hashes[hash] = true
	lb.hashring[hash] = args.Address

	client, dialError := rpc.DialHTTP("tcp", args.Address)
	if dialError != nil {
		return dialError
	}

	lb.nodes[hash] = client
	lb.phyNodes[hash] = client

	// Update current MMEs replicas
	if lb.physicalNodes > 1 {
		physicalReplicaID := lb.FindSuccessor(hash)
		physicalReplicaAddress := lb.hashring[physicalReplicaID]

		for {
			if physicalReplicaAddress == args.Address {
				physicalReplicaID = lb.FindSuccessor(physicalReplicaID)
				physicalReplicaAddress = lb.hashring[physicalReplicaID]
			} else {
				lb.replicas[args.Address][physicalReplicaAddress] = 1

				break
			}
		}
	}

	for i := 1; i < lb.ringWeight; i++ {
		lb.ringNodes++
		vNodeHash := lb.hasher.VirtualNodeHash(args.Address, i)
		lb.allHashes = append(lb.allHashes, vNodeHash)
		lb.hashes[vNodeHash] = true
		lb.hashring[vNodeHash] = args.Address
		lb.nodes[vNodeHash] = client

		virtualReplicaID := lb.FindSuccessor(vNodeHash)
		virtualReplicaAddress := lb.hashring[virtualReplicaID]
		// Update current MMEs replicas
		if lb.physicalNodes > 1 {
			for {
				if virtualReplicaAddress == args.Address {
					virtualReplicaID = lb.FindSuccessor(virtualReplicaID)
					virtualReplicaAddress = lb.hashring[virtualReplicaID]
				} else {
					lb.replicas[args.Address][virtualReplicaAddress] = 1

					break
				}
			}
		}
		lb.virtualNodes[args.Address] = append(lb.virtualNodes[args.Address], vNodeHash)
	}

	// Update all replicas
	var ReplicaMap map[string][]string
	ReplicaMap = make(map[string][]string)
	for _, mmeHash := range lb.allHashes {
		mmeAddress := lb.hashring[mmeHash]
		mmeReplicaHash := lb.FindSuccessor(mmeHash)
		mmeReplicaAddress := lb.hashring[mmeReplicaHash]
		if len(lb.virtualNodes[mmeAddress]) == 0 {
			lb.replicas[mmeAddress] = make(map[string]int)
		}
		if lb.physicalNodes > 1 {
			for {
				if mmeAddress == mmeReplicaAddress {
					mmeReplicaHash = lb.FindSuccessor(mmeReplicaHash)
					mmeReplicaAddress = lb.hashring[mmeReplicaHash]
				} else {
					lb.replicas[mmeAddress][mmeReplicaAddress] = 1
					ReplicaMap[mmeAddress] = lb.CheckDuplicates(ReplicaMap[mmeAddress], mmeReplicaAddress, args.Address)
					break
				}
			}

		}
	}

	for k := range lb.replicas[args.Address] {
		replicas = append(replicas, k)
	}

	reply.Hash = hash
	reply.Replicas = replicas

	// Get state from all MMEs
	completeState := make(map[uint64]rpcs.MMEState)
	for _, mme := range lb.phyNodes {
		sendStateArgs := rpcs.SendStateArgs{}
		sendStateReply := rpcs.SendStateReply{}
		sendStateArgs.LBHostport = lb.hostport

		mme.Call("MME.SendState", &sendStateArgs, &sendStateReply)
		for k, v := range sendStateReply.State {
			completeState[k] = v
		}
	}

	// Recompute State for all MMEs and send it to them

	var updatedReplicas []string
	if len(completeState) == 0 {
		for k := 0; k < len(lb.allHashes); k++ {

			ownerMME := lb.FindSuccessor(lb.allHashes[k])
			mmeReplicas := lb.replicas[lb.hashring[ownerMME]]
			for k := range mmeReplicas {
				updatedReplicas = append(updatedReplicas, k)
			}
			recvStateArgs := rpcs.SendReplicaArgs{}

			recvStateArgs.Replicas = updatedReplicas

			recvStateReply := rpcs.SendReplicaReply{}
			lb.nodes[ownerMME].Call("MME.RecvReplica", &recvStateArgs, &recvStateReply)

			updatedReplicas = nil

		}

	}

	for k, v := range completeState {

		ownerMME := lb.FindSuccessor(k)
		mmeReplicas := lb.replicas[lb.hashring[ownerMME]]
		// var replicas []string
		for k := range mmeReplicas {
			updatedReplicas = append(updatedReplicas, k)
		}
		recvStateArgs := rpcs.RecvStateArgs{}

		recvStateArgs.Replicas = updatedReplicas
		recvStateArgs.Key = k
		recvStateArgs.Val = v
		recvStateReply := rpcs.RecvStateReply{}

		lb.nodes[ownerMME].Call("MME.RecvState", &recvStateArgs, &recvStateReply)
		updatedReplicas = nil
	}
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
	var Nodes []string
	for i := 0; i < len(lb.allHashes); i++ {
		checkPhyNode := false
		for k := range lb.phyNodes {
			if k == lb.allHashes[i] {
				checkPhyNode = true
				break
			}

		}
		if checkPhyNode == true {
			Nodes = append(Nodes, lb.hashring[lb.allHashes[i]])
		}
	}
	reply.ServerNames = Nodes
	lb.stateMutex.Unlock()
	return nil
}

func (lb *loadBalancer) sortHashes() {
	sort.Slice(lb.allHashes, func(i, j int) bool { return lb.allHashes[i] < lb.allHashes[j] })
}
func (lb *loadBalancer) FindSuccessor(hash uint64) uint64 {
	lb.sortHashes()
	numHashes := lb.ringNodes
	for i := 0; i < numHashes; i++ {
		if lb.allHashes[i] > hash {
			return lb.allHashes[i]
		}
	}
	return lb.allHashes[0]
}

func (lb *loadBalancer) FindPred(hash uint64) uint64 {
	numHashes := lb.ringNodes
	for i := numHashes - 1; i >= 0; i-- {
		if lb.allHashes[i] < hash {
			return lb.allHashes[i]
		}
	}
	return lb.allHashes[numHashes-1]
}
func (lb *loadBalancer) CheckDuplicates(list []string, item string, item2 string) []string {
	if item == item2 {
		return list
	}
	for i := 0; i < len(list); i++ {
		if list[i] == item {
			return list
		}
	}

	list = append(list, item)
	return list
}
