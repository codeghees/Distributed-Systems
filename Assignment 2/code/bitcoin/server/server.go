package main

/*
- I use two channels, one for active miners, the other for active jobs.
- When a job is active, it activates the scheduler thread which in turn waits for an active miner, if an active miner is found it fragments the jobs
and sends it in a newly spawned go routine. A result handler is spawned and waits for all the slices to return and send result back to client.
If miner fails, job is inserted back into channel, if it is finished miner is inserted back.

*/
import (
	"bitcoin"
	"container/heap"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net"
	"os"
	"reflect"
	"strconv"
)

type server struct {
	listener net.Listener
}

// JobHeap ensures fairness
type JobHeap []JobStruct

func (h JobHeap) Len() int           { return len(h) }
func (h JobHeap) Less(i, j int) bool { return h[i].slices < h[j].slices }
func (h JobHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

// Push appends in Heap
func (h *JobHeap) Push(x interface{}) {
	*h = append(*h, x.(JobStruct))
}

// Pop removes in Heap
func (h *JobHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// ErrorStruct is used for incomplete jobs
type ErrorStruct struct {
	Lower uint64
	Upper uint64
}

// JobStruct would have all fields related to a job
type JobStruct struct {
	Request  bitcoin.Message
	socket   net.Conn //Client
	ClientID int
	slices   uint64
}

//MinerStruct would help in keeping track of active miners
type MinerStruct struct {
	socket net.Conn //Miner
	status bool     //True if busy
	id     int
}

const jobsize = 1000

func startServer(port int) (*server, error) {
	var ServerObj server
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		return nil, err
	}
	ServerObj.listener = ln

	return &ServerObj, nil
}

// LOGF to capture output
var LOGF *log.Logger

// MinerChan keeps tracks of active miners
var MinerChan chan MinerStruct

// JobChan keeps track of active jobs
var JobChan chan JobStruct

// SortedJobChan keeps track of jobs in terms of size
var SortedJobChan chan JobStruct

// SmallJobChan keeps track of small requests
var SmallJobChan chan JobStruct

func main() {
	// You may need a logger for debug purpose
	const (
		name = "/mnt/f/Fall 2019/Distributed Systems/Assignment 2/code/server.txt"
		flag = os.O_RDWR | os.O_CREATE
		perm = os.FileMode(0666)
	)

	file, err := os.OpenFile(name, flag, perm)
	if err != nil {
		return
	}
	defer file.Close()

	LOGF = log.New(file, "", log.Lshortfile|log.Lmicroseconds)
	// Usage: LOGF.Println() or LOGF.Printf()

	// Code starts here
	nMiners := 0
	nClients := 0
	MinerChan = make(chan MinerStruct, 500)
	JobChan = make(chan JobStruct, 500)
	SortedJobChan = make(chan JobStruct, 500)
	SmallJobChan = make(chan JobStruct, 500)
	go JobSort(SortedJobChan)
	go Scheduler(SortedJobChan)
	const numArgs = 2
	if len(os.Args) != numArgs {
		fmt.Printf("Usage: ./%s <port>", os.Args[0])
		return
	}

	port, err := strconv.Atoi(os.Args[1])
	if err != nil {
		fmt.Println("Port must be a number:", err)
		return
	}

	srv, err := startServer(port)
	LOGF.Println("Server Started! ")

	if err != nil {
		fmt.Println(err.Error())
		LOGF.Println(err.Error())
		return
	}
	fmt.Println("Server listening on port", port)

	for {
		conn, err := srv.listener.Accept()
		if err != nil {
			fmt.Printf("Couldn't accept a client connection: %s\n", err)
			continue
		}
		ConnMsg := bitcoin.Message{Type: 0, Data: "", Lower: 0, Upper: 0, Hash: 0, Nonce: 0}
		ConnBytes := make([]byte, reflect.TypeOf(ConnMsg).Size()*3)
		len, err := conn.Read(ConnBytes)
		ConnBytes = ConnBytes[:len]
		err = json.Unmarshal(ConnBytes, &ConnMsg)
		defer srv.listener.Close()

		if ConnMsg.Type == 0 {

			nMiners++

			Miner := MinerStruct{conn, false, nMiners}

			MinerChan <- Miner
			// LOGF.Println("Miner sent")

		}
		if ConnMsg.Type == 1 {

			nClients++

			slices := (ConnMsg.Upper - ConnMsg.Lower) / jobsize
			if (ConnMsg.Upper-ConnMsg.Lower)%jobsize != 0 {
				slices++
			}
			LOGF.Println("JOB SIZE", slices)
			Client := JobStruct{ConnMsg, conn, nClients, slices}

			JobChan <- Client

			LOGF.Println("JD -- >", Client)
		}

	}
}

// JobSort is a thread that sorts jobs according to size
func JobSort(SortedJobChan chan JobStruct) {
	// Implement PQ
	// LOGF.Println("JobSort")
	h := &JobHeap{}
	heap.Init(h)
	for {
		select {
		case job := <-JobChan:
			heap.Push(h, job)
			SortedJobChan <- heap.Pop(h).(JobStruct)
			// fmt.Println()

		case job := <-SmallJobChan:
			SortedJobChan <- job

		}

	}

}

// Scheduler manages all miners and jobs
func Scheduler(SortedJobChan chan JobStruct) {
	var TempLower uint64
	var TempUpper uint64

	for {
		Job := <-SortedJobChan
		// LOGF.Println("START SCHED")

		Slices := Job.slices
		Upper := Job.Request.Upper
		Lower := Job.Request.Lower
		ResultChan := make(chan bitcoin.Message)
		ErrChan := make(chan ErrorStruct)
		JobComplete := make(chan bool)
		go ResultHandler(ResultChan, Job, JobComplete)
		TempLower = Lower
		// Fragmenting Jobs
		LOGF.Println("STARTING JOBS", Slices)
		for i := uint64(0); i < Slices; i++ {
			if TempLower+jobsize < Upper {
				TempUpper = TempLower + jobsize

			} else {
				TempUpper = TempLower + (Upper - TempLower)
			}
			// LOGF.Println(TempLower, TempUpper)

			AvailableMiner := <-MinerChan
			go ExecuteJob(AvailableMiner, TempLower, TempUpper, ResultChan, ErrChan, Job.Request) // Start Job
			TempLower = TempUpper
			if TempUpper == Upper {
				break
			}
		}
		// Waiting for job to finish and/or crash
		JobEnd := false
		for {
			if JobEnd == true {
				LOGF.Println("JOB ENDED")
				break
			}
			select {
			case FailedJob := <-ErrChan:
				AvailableMiner := <-MinerChan
				go ExecuteJob(AvailableMiner, FailedJob.Lower, FailedJob.Upper, ResultChan, ErrChan, Job.Request)
			case Finish := <-JobComplete:
				if Finish == true {
					JobEnd = true
					break
				}

			}
		}

	}
}

//ResultHandler compiles the results for a particular job
func ResultHandler(Result chan bitcoin.Message, Job JobStruct, CompleteJob chan bool) {
	/*
		- Needs to keep track of all the slices.
		- Needs to calculate lowest hash.
		- Send result back to client.
		- Send done message on chan CompleteJob
	*/
	TotalSlices := Job.slices
	SlicesRecv := uint64(0)
	var LowestHash, Nonce uint64
	LowestHash = uint64(math.MaxInt64)
	for {
		select {
		case msg := <-Result:
			if msg.Hash < LowestHash {
				LowestHash = msg.Hash
				Nonce = msg.Nonce
			}
			SlicesRecv++

			if SlicesRecv == TotalSlices {
				// JOB COMPLETE

				ResultObj := bitcoin.NewResult(LowestHash, Nonce)
				ResultBytes, _ := json.Marshal(ResultObj)
				_, err := Job.socket.Write(ResultBytes)
				LOGF.Println("Sending Result", ResultObj)
				CompleteJob <- true
				if err != nil {
					return
				}
				return

			}

		}
	}

}

//ExecuteJob connects with Miner and sends result
func ExecuteJob(Miner MinerStruct, Lower uint64, Upper uint64, Result chan bitcoin.Message, Error chan ErrorStruct, Job bitcoin.Message) {
	/*
		- Connect with Miner and send job -
		- Wait for Result
		- Send Result on Result channel
		- Reinsert Miner if job finished
		- if Miner fails, terminate thread, send err back to channel.
	*/

	//Sending Job
	ServerReq := bitcoin.NewRequest(Job.Data, Lower, Upper)
	MessageByte, _ := json.Marshal(ServerReq)
	_, err1 := Miner.socket.Write(MessageByte)
	if err1 != nil {
		Error <- ErrorStruct{Lower, Upper}
		return
	}
	ResultObj := bitcoin.Message{Type: 2, Data: "", Lower: 0, Upper: 0, Hash: 0, Nonce: 0}
	ResultBytes := make([]byte, reflect.TypeOf(ResultObj).Size()*3)
	len, err2 := Miner.socket.Read(ResultBytes)
	if err2 != nil {
		Error <- ErrorStruct{Lower, Upper}
		return
	}

	ResultBytes = ResultBytes[:len]
	err1 = json.Unmarshal(ResultBytes, &ResultObj)
	Result <- ResultObj
	MinerChan <- Miner //Reinsert Miner

}
