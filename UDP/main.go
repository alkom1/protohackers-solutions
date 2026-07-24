package main

import (
	"log"
	"net"
	"strings"
)

var db = make(map[string]string)

func main() {
	udpAddr, err := net.ResolveUDPAddr("udp", ":9904")
	if err != nil {
		log.Fatal("udp address error", err)
	}
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Fatal("Error listening:", err)
	}

	db["version"] = "Alk's UDP database 1.0"
	log.Println("Running...")
	defer conn.Close()

	// we might have to setup goroutines for reading
	// https://ops.tips/blog/udp-client-and-server-in-go/#a-udp-server-in-go
	buf := make([]byte, 1024)
	for {
		n, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Println(err)
			return
		}

		res := handle(string(buf[:n]))
		if len(res) > 0 {
			conn.WriteToUDP([]byte(res), addr)
		}
	}
}

func handle(req string) string {
	// req = strings.TrimSpace(req)
	index := strings.Index(req, "=")

	if index > -1 {
		// Insert
		key := req[:index]
		value := req[index+1:]
		if key != "version" {
			log.Println("insert", key, value)
			db[key] = value
		}
		return ""
	}
	// Query
	value, ok := db[req]
	log.Println("query", req, ok, value)
	if ok {
		return req + "=" + value
	}
	return ""
}

// PUZZLE: https://protohackers.com/problem/4
