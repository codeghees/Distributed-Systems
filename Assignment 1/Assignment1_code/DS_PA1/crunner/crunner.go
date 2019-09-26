// package main

// import (
//     "fmt"
// )

// const (
//     defaultHost = "localhost"
//     defaultPort = 9999
// )

// // To test your server implementation, you might find it helpful to implement a
// // simple 'client runner' program. The program could be very simple, as long as
// // it is able to connect with and send messages to your server and is able to
// // read and print out the server's response to standard output. Whether or
// // not you add any code to this file will not affect your grade.
// func main() {
//     fmt.Println("Not implemented.")
// }

package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

func main() {

	// open a connection to the echo server
	rw, err := Open("localhost:9963")
	if err != nil {
		fmt.Printf("There was an error opening a connection to localhost on port 9999: %s\n", err)
		return
	}

	// get a reader from Stdin
	scanner := bufio.NewScanner(os.Stdin)
	go func() {
		fmt.Println("Starting Server Read...")
		for {
			msg, err := rw.ReadString('\n')
			if err != nil {
				fmt.Printf("There was an error reading from the server: %s\n", err)
				return
			}

			// print server's reply
			fmt.Println(msg)
			fmt.Println(len(msg))
		}
	}()

	for scanner.Scan() {

		// read from Stdin
		input := scanner.Text()

		// write it to the server

		_, err := rw.WriteString(input + "\n")
		if err != nil {
			fmt.Printf("There was an error writing to the server: %s\n", err)
			return
		}
		err = rw.Flush()
		if err != nil {
			fmt.Printf("There was an error writing to the server: %s\n", err)
			return
		}
		// read server's reply

	}
}

// Open opens a connection to addr and returns back any errors
func Open(addr string) (*bufio.ReadWriter, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	return bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn)), nil
}
