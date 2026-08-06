package main

import (
	"fmt"
	"io"
	"log"
	"net"
)

func main() {
	listener, err := net.Listen("tcp", ":9908")
	if err != nil {
		log.Fatal("Error listening:", err)
	}

	defer listener.Close()

	fmt.Println("Running...")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Error accepting connection:", err)
			continue
		}

		go handleConnection(conn)
	}
}

type ISLStream struct {
	io.ReadWriteCloser
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	// 1. read and remember the cipher spec
	// 2. build a tree from the cipher spec and in-out simplify it
	// 3. if the cipher spec simplifies to no-op, close the connection
	// 4. build and remember the reverse of the cipher spec

	// 1. receive message from connection -> decrypt it -> send it to the application layer
	// 2. receive payload from application layer -> encrypt it -> send it through the channel
	// how many goroutines per connection?
	//  1? ignore the separation into layers
	//  2? one low level, one application layer
	//  2? one for each direction
	//  3? low level one for each direction, application layer only one
}

// CIPHER SPEC SOLVER
//TBD

// PUZZLE: https://protohackers.com/problem/8
