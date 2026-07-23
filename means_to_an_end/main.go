package main

import (
	"bufio"
	"encoding/binary"
	"io"
	"log"
	"net"
	"slices"
	"sort"
)

func main() {
	listener, err := net.Listen("tcp", ":9902")
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

type timestampedPrice struct {
	Timestamp int32
	Price     int32
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	var store []timestampedPrice
	buf := make([]byte, 9)
	reader := bufio.NewReader(conn)
	for {
		if _, err := io.ReadFull(reader, buf); err != nil {
			log.Println(err)
			break
		}

		operation := buf[0]
		value1 := int32(binary.BigEndian.Uint32(buf[1:5]))
		value2 := int32(binary.BigEndian.Uint32(buf[5:9]))

		switch operation {
		case 73: // Insert
			item := timestampedPrice{
				Timestamp: value1,
				Price:     value2,
			}
			store = insert(store, item)
			log.Println("inserted", value1, value2)

		case 81: // Query
			r := getMean(store, value1, value2)
			rBuf := make([]byte, 4)
			binary.BigEndian.PutUint32(rBuf, uint32(r))
			conn.Write(rBuf)
			log.Println("Query", value1, value2, r)

		default:
			log.Println("unknown operation", operation)
		}
	}

}

func getMean(store []timestampedPrice, start int32, end int32) int32 {
	startIndex := sort.Search(len(store), func(i int) bool { return store[i].Timestamp > start })
	endIndex := sort.Search(len(store), func(i int) bool { return store[i].Timestamp > end })
	log.Println("getMean", startIndex, endIndex)
	l := endIndex - startIndex
	s := 0
	for i := startIndex; i < endIndex; i++ {
		s += int(store[i].Price)
	}
	return int32(s / l)
}

func insert(s []timestampedPrice, item timestampedPrice) []timestampedPrice {
	index := sort.Search(len(s), func(i int) bool { return s[i].Timestamp > item.Timestamp })
	return slices.Insert(s, index, item)
}

// PUZZLE: https://protohackers.com/problem/2
