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

	// goroutines?
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
	before, after, ok0 := strings.Cut(req, "=")

	if ok0 {
		// Insert
		if before != "version" {
			// log.Println("insert", key, value)
			db[before] = after
		}
		return ""
	}
	// Query
	value := db[req]
	// log.Println("query", req, ok, value)
	return req + "=" + value
}

// PUZZLE: https://protohackers.com/problem/4
