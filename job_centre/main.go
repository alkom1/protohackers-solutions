package main

import (
	"bufio"
	"encoding/json" // could use /v2 instead
	"fmt"
	"log"
	"net"
)

type Job struct {
	id       int
	priority int
	index    int // for heap purposes
	data     any
}

var (
	queues  = make(map[string]PriorityQueue)
	waiting = make(map[string][]*net.Conn) // could use channels instead for queues
	working = make(map[*net.Conn][]int)
)

func main() {
	listener, err := net.Listen("tcp", ":9909")
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

	// 1. decode to generic json with field "request"
	// 2. based on "request" look for more fields
	//    and do appropriate action

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Bytes()

		var r RequestBasic
		if err := json.Unmarshal(line, &r); err != nil {
			log.Println(err)
			return
		}

		switch r.Request {
		case "put":
			// TODO
			var r RequestPut
			if err := json.Unmarshal(line, &r); err != nil {
				log.Println(err)
				return
			}
		case "get":
			// TODO
			var r RequestGet
			if err := json.Unmarshal(line, &r); err != nil {
				log.Println(err)
				return
			}
		case "delete":
			// TODO
			var r RequestDelete
			if err := json.Unmarshal(line, &r); err != nil {
				log.Println(err)
				return
			}
		case "abort":
			// TODO
			var r RequestAbort
			if err := json.Unmarshal(line, &r); err != nil {
				log.Println(err)
				return
			}
		default:
			// TODO: invalid request type
			return
		}
	}
	if err := scanner.Err(); err != nil {
		log.Println(err)
	}
}

// PROTOCOL:
//  tcp -> json
//   put: put new job into named priority q
//   get: get highest priority job from the named q
//    waiting vs non-waiting
//   delete: delete a job based on id, no clients working on it anymore
//   abort: cancel the client's job and return it to q

// JSON stuff
type RequestBasic struct {
	Request string `json:"request"`
}

type RequestPut struct {
	Request   string         `json:"request"`
	QueueName string         `json:"queue"`
	Priority  int            `json:"pri"`
	Data      map[string]any `json:"job"`
}

type RequestGet struct {
	Request    string   `json:"request"`
	QueueNames []string `json:"queues"`
	Wait       bool     `json:"wait"`
}

type RequestDelete struct {
	Request string `json:"request"`
	Id      int    `json:"id"`
}

type RequestAbort struct {
	Request string `json:"request"`
	Id      int    `json:"id"`
}

// PUZZLE: https://protohackers.com/problem/9
