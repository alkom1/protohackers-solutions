package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net"
	"strings"
)

type response struct {
	Method string `json:"method"`
	Prime  bool   `json:"prime"`
}

type request struct {
	Method  string          `json:"method"`
	NumberS json.RawMessage `json:"number"`
}

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

	reader := bufio.NewReader(conn)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		text := scanner.Text()
		log.Println(text)

		var req request
		if err := json.Unmarshal([]byte(text), &req); err != nil {
			log.Println("JSON decoding error:", err)
			conn.Write([]byte("{}\n"))
			return
		}

		if req.Method != "isPrime" {
			log.Println("not isPrime")
			conn.Write([]byte("{}\n"))
			return
		}

		if strings.ContainsAny(string(req.NumberS), "\"'") {
			log.Println("not number")
			conn.Write([]byte("{}\n"))
			return
		}

		if strings.ContainsAny(string(req.NumberS), ".") {
			log.Println("not integer")
			conn.Write([]byte("{\"method\":\"isPrime\",\"prime\":false}\n"))
			return
		}

		number, err := parseBigInt(req.NumberS)
		if err != nil {
			log.Println("error int parsing:", err)
			conn.Write([]byte("{}\n"))
			return
		}

		res := &response{
			Method: "isPrime",
			Prime:  number.ProbablyPrime(20),
		}
		b, err := json.Marshal(res)
		if err != nil {
			log.Println("Error JSONing:", err)
			return
		}
		log.Println("sending:", string(b))
		conn.Write(b)
		conn.Write([]byte{'\n'})
	}

	if err := scanner.Err(); err != nil {
		log.Fatal("scanner err:", err)
	}
}

func parseBigInt(raw json.RawMessage) (*big.Int, error) {
	// Reject JSON strings
	if len(raw) == 0 || raw[0] == '"' {
		return nil, fmt.Errorf("expected JSON number")
	}

	n, ok := new(big.Int).SetString(string(raw), 10)
	if !ok {
		return nil, fmt.Errorf("invalid integer")
	}

	return n, nil
}

// PUZZLE: https://protohackers.com/problem/1
