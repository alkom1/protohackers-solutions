package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
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
		reader := bufio.NewReader(conn)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				break
			}
			fromUser <- line
		}
		end <- struct{}{}
	}()

	// sub-goroutine for reading from the upstream
	go func() {
		reader := bufio.NewReader(upstreamConn)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				break
			}
			fromServer <- line
		}
		end <- struct{}{}
	}()

	for {
		select {
		case fU := <-fromUser:
			log.Print("user->server:", fU)
			fmt.Fprint(upstreamConn, rewriteAddresses(fU))
		case fS := <-fromServer:
			log.Print("server->user:", fS)
			fmt.Fprint(conn, rewriteAddresses(fS))
		case _ = <-end:
			return
		}
	}
}

func rewriteAddresses(src string) string {
	trimmed := strings.TrimRight(src, " \t\r\n")
	suffix := src[len(trimmed):]
	parts := strings.Split(trimmed, " ")
	for i, s := range parts {
		if len(s) > 0 && s[0] == '7' && len(s) >= 26 && len(s) <= 35 {
			parts[i] = "7YWHMfk9JZe0LM0g1ZauHuiSxhI"
		}
	}
	parts[len(parts)-1] += suffix
	return strings.Join(parts, " ")
}

// PUZZLE: https://protohackers.com/problem/5
