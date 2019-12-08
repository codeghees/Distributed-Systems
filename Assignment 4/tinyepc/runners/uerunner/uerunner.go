package main

import (
	"flag"
	"fmt"
	"log"
	"net/rpc"
	"strconv"
	"tinyepc/rpcs"
)

var (
	lbPort = flag.Int("port", 8000, "Port of the LoadBalancer")
)

func init() {
	log.SetFlags(log.Lshortfile | log.Lmicroseconds)
}

func main() {
	flag.Parse()

	lbAddr := ":" + strconv.Itoa(*lbPort)
	_ = lbAddr

	// TODO: Implement this!

	client, dialError := rpc.DialHTTP("tcp", lbAddr)
	if dialError != nil {
		return
	}
	joinArgs := rpcs.UERequestArgs{}
	joinReply := rpcs.UERequestReply{}
	joinArgs.UEOperation = 0
	joinArgs.UserID = 55
	client.Call("LoadBalancer.RecvUERequest", &joinArgs, &joinReply)

	joinArgs1 := rpcs.UERequestArgs{}
	joinReply1 := rpcs.UERequestReply{}
	joinArgs1.UEOperation = 0
	joinArgs1.UserID = 56
	client.Call("LoadBalancer.RecvUERequest", &joinArgs1, &joinReply1)

	joinArgs2 := rpcs.UERequestArgs{}
	joinReply2 := rpcs.UERequestReply{}
	joinArgs2.UEOperation = 0
	joinArgs2.UserID = 57
	client.Call("LoadBalancer.RecvUERequest", &joinArgs2, &joinReply2)

	fmt.Println(joinReply)
}
