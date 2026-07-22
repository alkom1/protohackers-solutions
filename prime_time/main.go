package main

import (
	"bufio"
	"encoding/json"
	"log"
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

		var number int
		if err := json.Unmarshal(req.NumberS, &number); err != nil {
			log.Println("error int parsing:", err)
			conn.Write([]byte("{}\n"))
			return
		}

		res := &response{
			Method: "isPrime",
			Prime:  isPrime(number),
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

func isPrime(n int) bool {
	if n <= 1 {
		return false
	}
	for i := 2; i*i <= n; i++ {
		if n%i == 0 {
			return false
		}
	}
	return true
} // https://www.geeksforgeeks.org/go-language/how-to-find-prime-number-in-golang/

// Based on https://gobyexample.com/json
