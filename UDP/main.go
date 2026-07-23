package main

import (
	"log"
	"net"
)

func main() {
	udpAddr, err := net.ResolveUDPAddr("udp", ":9904")
	if err != nil {
		log.Fatal("udp address error", err)
	}
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Fatal("Error listening:", err)
	}

	log.Println("Running...")
	defer conn.Close()

	for {
		var buf [512]byte
		_, addr, err := conn.ReadFromUDP(buf[0:])
		if err != nil {
			log.Println(err)
			return
		}

		conn.WriteToUDP([]byte("hi\n"), addr)
	}

}

// PUZZLE: https://protohackers.com/problem/4
