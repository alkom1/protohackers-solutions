package main

import (
	"log"
	"net"
)

func main() {
	listener, err := net.Listen("tcp", ":9901")
	if err != nil {
		log.Fatal("Error listening:", err)
	}

	defer listener.Close()

	log.Println("Running...")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Error accepting connection:", err)
			continue
		}

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
}

// PUZZLE: https://protohackers.com/problem/2
