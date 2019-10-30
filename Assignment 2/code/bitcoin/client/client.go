package main

import (
	"bitcoin"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"reflect"
	"strconv"
)

// LOGF used to capture output
var LOGF *log.Logger

func main() {
	// LOG
	const (
		name = "/mnt/f/Fall 2019/Distributed Systems/Assignment 2/code/client.txt"
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

	const numArgs = 4
	if len(os.Args) != numArgs {
		fmt.Printf("Usage: ./%s <hostport> <message> <maxNonce>", os.Args[0])
		return
	}
	hostport := os.Args[1]
	message := os.Args[2]

	maxNonce, err := strconv.ParseUint(os.Args[3], 10, 64)
	if err != nil {
		fmt.Printf("%s is not a number.\n", os.Args[3])
		return
	}

	client, err := net.Dial("tcp", hostport)
	if err != nil {
		fmt.Println("Failed to connect to server:", err)
		return
	}

	defer client.Close()
	// TODO: implement this!

	//Sending Request Message
	ServerReq := bitcoin.NewRequest(message, 0, maxNonce)
	MessageByte, _ := json.Marshal(ServerReq)
	client.Write(MessageByte)
	ResultObj := bitcoin.Message{Type: 2, Data: "", Lower: 0, Upper: 0, Hash: 0, Nonce: 0}
	ResultBytes := make([]byte, reflect.TypeOf(ResultObj).Size()*3)
	len, err := client.Read(ResultBytes)
	ResultBytes = ResultBytes[:len]
	len = len + 1 - 1 //Compiler Happy
	if err != nil {
		printDisconnected()
		return
	}

	err = json.Unmarshal(ResultBytes, &ResultObj)

	printResult(ResultObj.Hash, ResultObj.Nonce)
}

// printResult prints the final result to stdout.
func printResult(hash, nonce uint64) {

	LOGF.Println("Result", hash, nonce)
	fmt.Println("Result", hash, nonce)
}

// printDisconnected prints a disconnected message to stdout.
func printDisconnected() {
	fmt.Println("Disconnected")
}
