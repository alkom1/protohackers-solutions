package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
)

type response struct {
	Method string `json:"method"`
	Prime  bool   `json:"prime"`
}

func main() {
	listener, err := net.Listen("tcp", ":9901")
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
	text, err := reader.ReadString('\n')

	fmt.Println(text)

	if err != nil {
		log.Println("Error reading line:", err)
		return
	}

	var dat map[string]interface{}
	if err = json.Unmarshal([]byte(text), &dat); err != nil {
		log.Println("JSON decoding error:", err)
		return
	}

	//method := dat["method"].(string)
	numberFloat := dat["number"].(float64)
	number := (int)(numberFloat)

	res := &response{
		Method: "isPrime",
		Prime:  isPrime(number),
	}
	b, err := json.Marshal(res)
	if err != nil {
		log.Println("Error JSONing:", err)
		return
	}
	conn.Write(b)
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
