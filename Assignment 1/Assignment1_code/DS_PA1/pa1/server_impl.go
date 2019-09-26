// Implementation of a KeyValueServer. Students should write their code in this file.
//Some code taken from the tutorials

package pa1

import (
	"DS_PA1/rpcs"
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
)

type keyValueServer struct {
	// TODO: implement this!
	// Count for connected.
	// Channels for sending info
	// Port Number
	count       int
	port        int
	close       bool
	PutChan     chan string
	GetChan     chan string
	GetBChan    chan string
	GetMap      map[int](chan string)
	GetBroad    map[int](chan string)
	mapIndex    int
	countchan   chan int
	ln          net.Listener
	CountSync   chan string //Syncing Channel
	CountUpdate chan int
}

// New creates and returns (but does not start) a new KeyValueServer.
func New() KeyValueServer {
	// TODO: implement this!
	/*- Initiate the keyValueServer
	  - Return the pointer
	*/
	ptr := new(keyValueServer)
	ptr.count = 0
	ptr.port = 0
	ptr.mapIndex = 0
	ptr.close = false
	ptr.PutChan = make(chan string)
	ptr.GetChan = make(chan string)
	ptr.countchan = make(chan int)
	ptr.GetBChan = make(chan string)
	ptr.CountSync = make(chan string)
	ptr.CountUpdate = make(chan int)
	ptr.GetMap = make(map[int](chan string))
	ptr.GetBroad = make(map[int](chan string))
	initDB() // initialize Database

	return ptr
}

func (kvs *keyValueServer) ServerHandling() {

	go kvs.PutThread()
	go kvs.countR()

	for {
		conn, err := kvs.ln.Accept()

		if err != nil {
			// fmt.Printf("Couldn't accept a client connection: %s\n", err)
		} else {
			kvs.GetMap[kvs.mapIndex] = make(chan string, 500)
			kvs.GetBroad[kvs.mapIndex] = make(chan string, 500)

			rw := kvs.ConnectionToRW(conn)
			go kvs.ClientConn(conn, rw, kvs.mapIndex)
			go kvs.GetBThread(kvs.mapIndex, rw)
			go kvs.GetThread(kvs.mapIndex, rw)

			kvs.mapIndex = kvs.mapIndex + 1

			kvs.countchan <- 1
		}
	}

	return
}

func parse(base string, split string) [5]string {
	// fmt.Println(base, split)
	var theArray [5]string
	index := 0
	temp := ""
	for i := 0; i < len(base); i++ {
		if string(base[i]) == split {
			theArray[index] = temp
			fmt.Println(temp)
			temp = ""
			index = index + 1
			continue

		}
		temp = temp + string((base[i]))
		if i+1 == len(base) {
			theArray[index] = temp
		}
	}
	return theArray

}
func (kvs *keyValueServer) PutThread() {

	for {
		select {
		case msg := <-kvs.PutChan:
			values := strings.Split(msg, ",")
			key := values[1]
			value := values[2]
			put(key, []byte(value))
			// fmt.Println(values)
		case key := <-kvs.GetChan:

			value := get(key)
			Svalue := string([]byte(value))
			GetS := key + "," + Svalue
			// fmt.Println(GetS)
			for _, element := range kvs.GetBroad {

				if len(element) < 500 {
					element <- GetS

				}
			}

		}
	}
}

func (kvs *keyValueServer) GetThread(index int, rw *bufio.ReadWriter) {

	// fmt.Println("GET")
	for {

		select {

		case msg := <-kvs.GetMap[index]:

			if msg[:3] == "put" {
				msg = msg[:len(msg)-1]

				kvs.PutChan <- msg

			}
			if msg[:3] == "get" {
				msg = msg[:len(msg)-1]
				values := strings.Split(msg, ",")
				key := values[1]

				kvs.GetChan <- key

			}

		}

	}
}
func (kvs *keyValueServer) countR() {
	for {
		select {
		case c := <-kvs.countchan:
			kvs.count = kvs.count + c

		case msg := <-kvs.CountSync:
			if msg == "sync" {

				kvs.CountUpdate <- kvs.count

			}

		}

	}
}
func (kvs *keyValueServer) GetBThread(index int, rw *bufio.ReadWriter) {
	for {

		select {
		case msg := <-kvs.GetBroad[index]:
			rw.WriteString(msg + "\n")
			rw.Flush()

		}
	}

}
func (kvs *keyValueServer) ClientConn(conn net.Conn, rw *bufio.ReadWriter, index int) {

	for {
		// get client message

		msg, err := rw.ReadString('\n')
		if err != nil {
			// fmt.Printf("Client Left")
			kvs.countchan <- -1
			return
		}
		kvs.GetMap[index] <- msg
		// It is fine at this point

	}

	// echo back the same message to the client

}
func (kvs *keyValueServer) ConnectionToRW(conn net.Conn) *bufio.ReadWriter {
	return bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
}

func (kvs *keyValueServer) StartModel1(port int) error {
	// TODO: implement this!
	//Use Kvs
	//Initiate server
	//Call a go routine that is listening on the 'port'
	if kvs.close == true {

		return nil
	}
	kvs.port = port
	var sPort string
	sPort = ":" + strconv.Itoa(kvs.port)
	var err error
	kvs.ln, err = net.Listen("tcp", sPort)
	if err != nil {
		// fmt.Printf("Couldn't listen on port %s: %s\n", sPort, err)
		return err
	}
	go kvs.ServerHandling()
	return nil
}

func (kvs *keyValueServer) Close() {
	// TODO: implement this!
	kvs.ln.Close()
}

func (kvs *keyValueServer) Count() int {
	// TODO: implement this!
	// fmt.Println(kvs.count)

	kvs.CountSync <- "sync"
	count := <-kvs.CountUpdate
	return count
}

func (kvs *keyValueServer) StartModel2(port int) error {
	// TODO: implement this!
	//
	// Do not forget to call rpcs.Wrap(...) on your kvs struct before
	// passing it to <sv>.Register(...)
	//
	// Wrap ensures that only the desired methods (RecvGet and RecvPut)
	// are available for RPC access. Other KeyValueServer functions
	// such as Close(), StartModel1(), etc. are forbidden for RPCs.
	//
	// Example: <sv>.Register(rpcs.Wrap(kvs))
	return nil
}

func (kvs *keyValueServer) RecvGet(args *rpcs.GetArgs, reply *rpcs.GetReply) error {
	// TODO: implement this!
	return nil
}

func (kvs *keyValueServer) RecvPut(args *rpcs.PutArgs, reply *rpcs.PutReply) error {
	// TODO: implement this!
	return nil
}

// TODO: add additional methods/functions below!
