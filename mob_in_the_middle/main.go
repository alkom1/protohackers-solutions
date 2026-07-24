package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
)

func main() {
	listener, err := net.Listen("tcp", ":9905")
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

	// for starters:
	// mirror all messages 1:1
	upstreamAddr, err := net.ResolveTCPAddr("tcp", "chat.protohackers.com:16963")
	if err != nil {
		log.Fatal(err)
	}

	upstreamConn, err := net.DialTCP("tcp", nil, upstreamAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer upstreamConn.Close()

	fromUser := make(chan string)
	fromServer := make(chan string)
	end := make(chan struct{})

	// sub-goroutine for reading from the client
	go func() {
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			line := scanner.Text()
			fromUser <- line
		}
		if err := scanner.Err(); err != nil {
			log.Println(err)
		}
		end <- struct{}{}
	}()

	// sub-goroutine for reading from the upstream
	go func() {
		scanner := bufio.NewScanner(upstreamConn)
		for scanner.Scan() {
			line := scanner.Text()
			fromServer <- line
		}
		if err := scanner.Err(); err != nil {
			log.Println(err)
		}
		end <- struct{}{}
	}()

	for {
		select {
		case fU := <-fromUser:
			log.Println("user->server:", fU)
			fmt.Fprintf(upstreamConn, "%s\n", fU)
		case fS := <-fromServer:
			log.Println("server->user:", fS)
			fmt.Fprintf(conn, "%s\n", fS)
		case _ = <-end:
			return
		}
	}

	// later:
	//  detect b-addresses (regex?)
	//  replace them
}

// PUZZLE: https://protohackers.com/problem/5
