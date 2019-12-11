// This file contains the arguments and reply structs used to perform RPCs between
// the Load Balancer and MMEs.

package rpcs

// MMEState contains the state that is maintained for each UE
type MMEState struct {
	Balance float64
	// TODO: Implement this!
}

// TODO: add additional argument/reply structs here!

// MMEJoinArgs contains the arguments to pass to the MMEJoin RPC call
type MMEJoinArgs struct {
	Address string
}

// MMEJoinReply contains the reply from the Load Balancer to the MMEJoin RPC call
type MMEJoinReply struct {
	Hash          uint64              // Hash value for the new MME
	Replicas      []string            // Adress of the MME that is chosen to be the backup for the new MME
	ReplicaHashes []uint64            // Hashed value of the backup MME
	MMEState      map[uint64]MMEState //Relocated MME State
}

//TransferStateArgs contains information sent to an existing MME about a new MME that will be now responsible for some of its keys. Sent from LB to MME
type TransferStateArgs struct {
	NewMMEHash    uint64 // The hash for the new MME
	NewMMEAddress string // The address of the new MME
	MinHash       uint64 // The minimum hash value from where the state should be transferred
	MaxRingHash   uint64
	MinRingHash   uint64
	PrevMin       uint64
	PrevMax       uint64
}

//TransferStateReply contains the reply to a transfer state order from the LB
type TransferStateReply struct {
}

//RecvStateArgs contains the MME state that will be sent from an existing MME to a new MME
type RecvStateArgs struct {
	Key      uint64
	Val      MMEState
	Replicas []string
}

//RecvStateReply is the reply to a inter MME state transfer
type RecvStateReply struct {
}

//SendStateArgs contains the MME state that will be sent from MME to LB
type SendStateArgs struct {
	LBHostport string
}

//SendStateReply is the reply to a inter MME state transfer
type SendStateReply struct {
	State map[uint64]MMEState
}

//SendReplicaArgs used to send replicas
type SendReplicaArgs struct {
	Replicas []string
}

//SendReplicaReply placeholder
type SendReplicaReply struct {
}

// ========= DO NOT MODIFY ANYTHING BEYOND THIS LINE! =========

// Operation represents the different kinds of user operations (Call, SMS or Load)
type Operation int

const (
	// Call deducts 5 units from the user's balance
	Call Operation = iota
	// SMS deducts 1 unit from the user's balance
	SMS
	// Load adds 10 units to the user's balance
	Load
)

// UERequestArgs contains the arguments for MME.RecvUERequest RPC
// Each UE sends this to the Load Balancer which then hashes the UserID,
// replaces the UserID with the generated hash and then forwards it to the MME.
type UERequestArgs struct {
	UserID      uint64    // UserID (between UE and LB) or Hash (between LB and MME)
	UEOperation Operation // Call, SMS or Load
}

// UERequestReply contains the return values for MME.RecvUERequest RPC
type UERequestReply struct {
}

// LeaveArgs contains the arguments for LoadBalancer.RecvLeave RPC
// The tests use this to inform the Load Balancer to disconnect a MME (failure simulation)
type LeaveArgs struct {
	HostPort string // HostPort of MME to disconnect
}

// LeaveReply contains the return values for LoadBalancer.RecvLeave RPC
type LeaveReply struct {
}

// LBStatsArgs contains the arguments for LB.RecvLBStats RPC
type LBStatsArgs struct {
}

// LBStatsReply contains the return value for LB.RecvLBStats RPC
// The tests use this to fetch information about the consistent hash ring
type LBStatsReply struct {
	RingNodes     int      // Total number of nodes in the hash ring (physical + virtual)
	PhysicalNodes int      // Total number of physical nodes ONLY in the ring
	Hashes        []uint64 // Sorted List of all the nodes'(physical + virtual) hashes
	ServerNames   []string // List of all the physical nodes' hostPort string as they appear in the hash ring
}

// MMEStatsArgs contains the return value for MME.RecvMMEStats RPC
type MMEStatsArgs struct {
}

// MMEStatsReply contains the return value for MME.RecvMMEStats RPC
// The tests use this to fetch information about the MME
type MMEStatsReply struct {
	State     map[uint64]MMEState // Map of user states with hash of UserID as key and rpcs.MMEState as value
	Replicas  []string            // List of hostPort strings of replicas
	NumServed int                 // Number of user requests served by this MME
}
