package main

import (
	"log"
	"net"
)

func main() {
	listener, err := net.Listen("tcp", ":9903")
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

	conn.Write([]byte("What is your name?\n"))
	// read the username from the connection
	// validate it

	// announce the new user to other users
	// send the new user list of all already connected users (starts with *)

	// whenever user sends a message, relay it to all other users
	// do not send it to: users that have not entered username, the sender
	// format: "[NAME] MESSAGE"

	// when user disconnects, send all other users an announcement (starts with *)
}

// PUZZLE: https://protohackers.com/problem/3
