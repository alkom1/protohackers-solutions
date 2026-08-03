package main

import (
	"log"
	"net"
)

func main() {
	udpAddr, err := net.ResolveUDPAddr("udp", ":9907")
	if err != nil {
		log.Fatal("udp address error", err)
	}
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Fatal("Error listening:", err)
	}

	log.Println("Running...")
	defer conn.Close()

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

// overview:
// 1. build TCP over UDP
// 2. use the protocol (LRCP) below
// 3. reverse \n separated lines of text (up to 10k characters long)
// suggested abstraction:
//  protocol layer and
//  application layer (that sees only two tcp-like byte streams)

// message types:
// /connect/ID/ - start a new session with ID
//   save the associated (id, ip, port)
//   send /ack/ID/0
// /data/ID/POS/DATA/
//   ID - session id
//   POS - position in unescaped byte stream
//   DATA - actual data stream (slashes are escaped '\/' '\\')
//   send /ack/ with how much uninterrupted unescaped data we have already received
// /ack/ID/LEN/
// /close/ID/ - close session with ID

func handle(_ string) string {
	return "test"
}

// PUZZLE: https://protohackers.com/problem/7
