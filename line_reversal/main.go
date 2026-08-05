package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
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

		handleUdpPacket(conn, addr, string(buf[:n]))
		// conn.WriteToUDP([]byte(res), addr)
	}
}

// overview:
// 1. build TCP over UDP
// 2. use the protocol (LRCP) below
// 3. reverse \n separated lines of text (up to 10k characters long)
// suggested abstraction:
//  protocol layer and
//  application layer (that sees only two tcp-like byte streams)
// I will attempt to do that, exposing the stream as `io.ReadWriteCloser`

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

type LRCPStream struct {
	io.ReadWriteCloser
}

type LRCPSession struct {
	conn          *net.UDPConn
	addr          *net.UDPAddr
	id            int
	recvbuf       []byte
	sendbuf       []byte
	sendbufOffset int
	received      int
}

func (s LRCPSession) sendAck(len int) {
	err := sendResponse(s.conn, s.addr, MessageAck{
		ID:  s.id,
		LEN: len,
	})
	if err != nil {
		log.Println("send ack error:", err)
	}
}

func (s LRCPSession) sendData(payload string) {
	// TODO
	// send data
	// regularly check if we got ack
	// if not, resend
	// there should be max one data goroutine per session
}

var (
	sessions = make(map[int]LRCPSession)
)

func newSession(addr *net.UDPAddr, id int) LRCPSession {
	s := LRCPSession{
		addr:          addr,
		id:            id,
		recvbuf:       make([]byte, 0, 1024), // TODO: 10k?
		sendbuf:       make([]byte, 0, 1024), // TODO: 10k?
		sendbufOffset: 0,
		received:      0,
	}
	sessions[id] = s
	return s
}

func handleUdpPacket(conn *net.UDPConn, addr *net.UDPAddr, payload string) {
	msg, err := readMsg(payload)
	if err != nil {
		log.Println("read msg err:", err)
	}
	switch m := msg.(type) {
	case *MessageConnect:
		if _, exists := sessions[m.ID]; !exists {
			newSession(addr, m.ID)
			// TODO: start goroutine
		}
		sessions[m.ID].sendAck(sessions[m.ID].received)
	case *MessageData:
		// TODO
	case *MessageAck:
		// TODO
	case *MessageClose:
		// TODO
	}
}

func handleLRCPStream(s LRCPStream) {

}

func sendResponse(conn *net.UDPConn, addr *net.UDPAddr, msg Message) (err error) {
	_, err = conn.WriteToUDP([]byte(generateResponse(msg)), addr)
	return
}

type Message interface {
	Type() string
}

type MessageConnect struct {
	ID int
}

func (m MessageConnect) Type() string {
	return "connect"
}

type MessageData struct {
	ID   int
	POS  int
	DATA string
}

func (m MessageData) Type() string {
	return "data"
}

type MessageAck struct {
	ID  int
	LEN int
}

func (m MessageAck) Type() string {
	return "ack"
}

type MessageClose struct {
	ID int
}

func (m MessageClose) Type() string {
	return "close"
}

func readMsg(payload string) (Message, error) {
	splits := splitWithEscaping(payload)
	if len(splits) < 3 || len(splits[0]) != 0 || len(splits[len(splits)-1]) != 0 {
		return nil, fmt.Errorf("invalid format")
	}
	// 0 = empty
	// 1 = type
	// len(splits)-1 = empty
	// 2 = ID
	id, err := strconv.Atoi(splits[2])
	if err != nil {
		return nil, err
	}

	switch splits[1] {
	case "connect":
		return &MessageConnect{
			ID: id,
		}, nil
	case "data":
		// 2=ID, 3=POS, 4=DATA
		pos, err := strconv.Atoi(splits[3])
		if err != nil {
			return nil, err
		}
		return &MessageData{
			ID:   id,
			POS:  pos,
			DATA: splits[4],
		}, nil
	case "ack":
		// 2=ID, 3=LEN
		len, err := strconv.Atoi(splits[3])
		if err != nil {
			return nil, err
		}
		return MessageAck{
			ID:  id,
			LEN: len,
		}, nil
	case "close":
		// 2=ID
		return MessageClose{
			ID: id,
		}, nil
	}

	return nil, fmt.Errorf("unknown format")
}

func generateResponse(msg Message) string {
	switch m := msg.(type) {
	case *MessageConnect:
		return fmt.Sprintf("/connect/%d/", m.ID)
	case *MessageData:
		return fmt.Sprintf("/data/%d/%d/%s/", m.ID, m.POS, m.DATA)
	case *MessageAck:
		return fmt.Sprintf("/ack/%d/%d/", m.ID, m.LEN)
	case *MessageClose:
		return fmt.Sprintf("/close/%d/", m.ID)
	}

	// ERROR
	return ""
}

func splitWithEscaping(s string) []string {
	out := make([]string, 0, 8)
	buf := make([]byte, 0, len(s))
	var sep byte = '/'

	escaped := false
	for i := 0; i < len(s); i++ {
		c := s[i]

		if escaped {
			buf = append(buf, c)
			escaped = false
			continue
		}

		if c == '\\' {
			escaped = true
			continue
		}

		if c == sep {
			out = append(out, string(buf))
			buf = buf[:0]
			continue
		}

		buf = append(buf, c)
	}

	if escaped {
		buf = append(buf, '\\')
	}

	out = append(out, string(buf))
	return out
}

// PUZZLE: https://protohackers.com/problem/7
