package main

import (
	"fmt"
	"io"
	"log"
	"net"
)

func main() {
	listener, err := net.Listen("tcp", ":9900")
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

func handleConnection(conn net.Conn) {
	defer conn.Close()

	_, err := io.Copy(conn, conn)
	if err != nil {
		log.Println("Error copying:", err)
	}
}

// Based on https://gobyexample.com/tcp-server
// and https://protohackers.com/help
// PUZZLE: https://protohackers.com/problem/0
