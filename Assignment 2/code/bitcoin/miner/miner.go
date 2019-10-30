package main

import (
	"bitcoin"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net"
	"os"
	"reflect"
)

// Attempt to connect miner as a client to the server.
func joinWithServer(hostport string) (net.Conn, error) {
	// TODO: implement this!
	client, err := net.Dial("tcp", hostport)
	if err != nil {
		fmt.Println("Failed to connect to server:", err)
		return nil, err
	}
	ServerReq := bitcoin.NewJoin()
	MessageByte, _ := json.Marshal(ServerReq)
	client.Write(MessageByte)

	return client, nil
}

// LOGF to print
var LOGF *log.Logger

func main() {

	const (
		name = "/mnt/f/Fall 2019/Distributed Systems/Assignment 2/code/miner.txt"
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

	const numArgs = 2
	if len(os.Args) != numArgs {
		fmt.Printf("Usage: ./%s <hostport>", os.Args[0])
		return
	}

	hostport := os.Args[1]
	miner, err := joinWithServer(hostport)
	if err != nil {
		fmt.Println("Failed to join with server:", err)
		return
	}

	SmallestHash := uint64(math.MaxInt64)
	var nonce uint64
	for {
		// var JobObj *bitcoin.Message
		JobObj := bitcoin.Message{Type: 1, Data: "", Lower: 0, Upper: 0, Hash: 0, Nonce: 0}
		JobBytes := make([]byte, reflect.TypeOf(JobObj).Size()*3)
		len, err := miner.Read(JobBytes)
		JobBytes = JobBytes[:len]
		if err != nil { //Break Contact With Server
			break
		}
		err = json.Unmarshal(JobBytes, &JobObj)
		LOGF.Println(JobObj)
		for i := JobObj.Lower; i <= JobObj.Upper; i++ {
			CurrentHash := bitcoin.Hash(JobObj.Data, i)
			if CurrentHash < SmallestHash {
				SmallestHash = CurrentHash
				nonce = i

			}

		}
		LOGF.Println(SmallestHash, nonce)
		ResultObj := bitcoin.NewResult(SmallestHash, nonce)
		ResultBytes, _ := json.Marshal(ResultObj)
		miner.Write(ResultBytes)
		SmallestHash = uint64(math.MaxInt64)

	}
	defer miner.Close()

	// TODO: implement this!
}
