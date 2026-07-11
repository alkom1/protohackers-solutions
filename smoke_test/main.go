package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
)

func main() {
	listener, err := net.Listen("tcp", ":9090")
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

	reader := bufio.NewReader(conn)

	for {
		b, err := reader.ReadByte()
		if err != nil {
			break
		}
		_, err = conn.Write([]byte{b})
		if err != nil {
			break
		}
	}

	/*
		bytes, err := reader.ReadBytes()
		if err != nil {
			log.Printf("Read error: %v", err)
			return
		}

		_, err := conn.Write(bytes)
		if err != nil {
			log.Printf("Server write error: %v", err)
		}
	*/
}

// Based on https://gobyexample.com/tcp-server
