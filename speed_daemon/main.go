package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"slices"
	"time"
)

type CarRoad struct {
	road uint16
	car  string
}
type RoadEvent struct {
	timestamp uint32
	mile      uint16
}

type CarDay struct {
	day uint32
	car string
}

type CarRoadEvent struct {
	CarRoad
	RoadEvent
}

type DispatcherSubscription struct {
	roads []uint16
	conn  net.Conn
}

type RoadSpeedLimit struct {
	road  uint16
	limit uint16
}

var (
	RoadEvents        = make(map[CarRoad][]RoadEvent)
	AlreadyTicketed   = make(map[CarDay]bool)
	TicketDispatchers = make(map[uint16][]net.Conn)
	RoadLimits        = make(map[uint16]uint16)
	plates            = make(chan CarRoadEvent)
	dispatchers       = make(chan DispatcherSubscription)
	roads             = make(chan RoadSpeedLimit)
	dispatcherDC      = make(chan DispatcherSubscription)
	ticketQ           = make(map[uint16][]TicketInformation)
)

func main() {
	listener, err := net.Listen("tcp", ":9906")
	if err != nil {
		log.Fatal("Error listening:", err)
	}

	defer listener.Close()

	log.Println("Running...")

	go router()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Error accepting connection:", err)
			continue
		}

		go handleConnection(conn)
	}
}

func router() {
	for {
		select {
		case p := <-plates: // new car sighting
			processNewPlateSighting(p)
		case d := <-dispatchers: // new ticket dispatcher
			for _, r := range d.roads {
				TicketDispatchers[r] = append(TicketDispatchers[r], d.conn)
				for _, x := range ticketQ[r] {
					issueTicket(x)
				}
				delete(ticketQ, r)
			}

		case r := <-roads: // just note down the road speed limit
			log.Println("road speed limit", r.road, r.limit)
			RoadLimits[r.road] = r.limit * 100

		case d := <-dispatcherDC: // dispatcher client disconnected
			log.Println("dispatcher disconnected", d.roads)
			for _, r := range d.roads {
				TicketDispatchers[r] = slices.DeleteFunc(TicketDispatchers[r], func(x net.Conn) bool {
					return x == d.conn
				})
			}

		}
	}
}

func processNewPlateSighting(e CarRoadEvent) {
	// add it to the list
	RoadEvents[e.CarRoad] = append(RoadEvents[e.CarRoad], e.RoadEvent)
	log.Println("plate seen", RoadEvents[e.CarRoad])
	// try all pairs to see if they should get a ticket
	for i := range RoadEvents[e.CarRoad] {
		for j := i + 1; j < len(RoadEvents[e.CarRoad]); j++ {
			event1 := RoadEvents[e.CarRoad][i]
			event2 := RoadEvents[e.CarRoad][j]
			speed := calculateSpeed(event1, event2)
			if speed > RoadLimits[e.road] {
				issueTicket(constructTicketInformation(event1, event2, e.CarRoad, speed))
			}
		}
	}
}

type TicketInformation struct {
	timestamp1 uint32
	mile1      uint16
	timestamp2 uint32
	mile2      uint16
	road       uint16
	car        string
	speed      uint16
}

func constructTicketInformation(e1 RoadEvent, e2 RoadEvent, c CarRoad, speed uint16) TicketInformation {
	if e1.timestamp > e2.timestamp {
		e1, e2 = e2, e1
	}

	return TicketInformation{
		timestamp1: e1.timestamp,
		mile1:      e1.mile,
		timestamp2: e2.timestamp,
		mile2:      e2.mile,
		road:       c.road,
		car:        c.car,
		speed:      speed,
	}
}

func issueTicket(ticket TicketInformation) {
	// find dispatcher for that road
	// send it
	// if none, add it to q
	dispatchers, r := TicketDispatchers[ticket.road]
	if !r || len(dispatchers) == 0 {
		// dispatcher not found, add it to q
		q := ticketQ[ticket.road]
		ticketQ[ticket.road] = append(q, ticket) // FIXME: nil slice
		return
	}

	cd1 := CarDay{
		car: ticket.car,
		day: ticket.timestamp1 / 86400,
	}

	cd2 := CarDay{
		car: ticket.car,
		day: ticket.timestamp2 / 86400,
	}

	if !AlreadyTicketed[cd1] {
		dispatcher := dispatchers[0]
		dispatcher.Write(buildTicketMessage(ticket))
		log.Println("ticket issued", ticket)
		AlreadyTicketed[cd1] = true
	}

	if !AlreadyTicketed[cd2] {
		dispatcher := dispatchers[0]
		dispatcher.Write(buildTicketMessage(ticket))
		log.Println("ticket issued", ticket)
		AlreadyTicketed[cd2] = true
	}
}

func calculateSpeed(e RoadEvent, o RoadEvent) uint16 {
	smallerTimestamp := e.timestamp
	largerTimestamp := o.timestamp
	if smallerTimestamp > largerTimestamp {
		smallerTimestamp, largerTimestamp = largerTimestamp, smallerTimestamp
	}
	smallerMile := e.mile
	largerMile := o.mile
	if smallerMile > largerMile {
		smallerMile, largerMile = largerMile, smallerMile
	}
	return uint16(uint32(largerMile-smallerMile) * 360000 / (largerTimestamp - smallerTimestamp))
}

const (
	UNKNOWN    = 1
	CAMERA     = 2
	DISPATCHER = 3
)

func handleConnection(conn net.Conn) {
	defer conn.Close()

	client_type := UNKNOWN
	heartbeat_interval := -1
	var camera_info *MessageIAmCamera = nil

	for {
		msg, err := buildMsg(conn)
		if err != nil {
			conn.Write(buildErrorMessage("you did an oopsie"))
			break
		}

		switch m := msg.(type) {
		case *MessageIAmCamera:
			if client_type != UNKNOWN {
				conn.Write(buildErrorMessage("you already selected type"))
				break
			}
			client_type = CAMERA
			camera_info = m
			roads <- RoadSpeedLimit{
				road:  m.road,
				limit: m.limit,
			}
			log.Println("Client identified as camera.")
		case *MessageIAmDispatcher:
			if client_type != UNKNOWN {
				conn.Write(buildErrorMessage("you already selected type"))
				break
			}
			client_type = DISPATCHER
			dispatchers <- DispatcherSubscription{
				roads: m.roads,
				conn:  conn,
			}
			log.Println("Client identified as dispatcher.")
			defer func() {
				dispatcherDC <- DispatcherSubscription{
					roads: m.roads,
					conn:  conn,
				}
			}()
		case *MessagePlate:
			if client_type != CAMERA {
				conn.Write(buildErrorMessage("you're not a camera"))
				break
			}
			plates <- CarRoadEvent{
				CarRoad{
					car:  m.plate,
					road: camera_info.road,
				},
				RoadEvent{
					timestamp: m.timestamp,
					mile:      camera_info.mile,
				},
			}
		case *MessageWantHeartbeat:
			if heartbeat_interval != -1 {
				conn.Write(buildErrorMessage("you already said this"))
				break
			}
			heartbeat_interval = int(m.interval)
			log.Println("heartbeat interval request", heartbeat_interval)
			if heartbeat_interval <= 0 {
				continue
			}
			ticker := time.NewTicker(time.Duration(heartbeat_interval) * 100 * time.Millisecond)
			tickerEnd := make(chan struct{})
			go func() {
				defer ticker.Stop()
				for {
					select {
					case <-ticker.C:
						conn.Write(buildHeartbeatMessage())
						log.Println("sent heartbeat")
					case <-tickerEnd:
						return
					}
				}
			}()
			defer close(tickerEnd)
		}
	}
}

type Message interface {
	Type() uint8
}

type MessagePlate struct {
	plate     string
	timestamp uint32
}

func (m MessagePlate) Type() uint8 {
	return 0x20
}

type MessageWantHeartbeat struct {
	interval uint32
}

func (m MessageWantHeartbeat) Type() uint8 {
	return 0x40
}

type MessageIAmCamera struct {
	road  uint16
	mile  uint16
	limit uint16
}

func (m MessageIAmCamera) Type() uint8 {
	return 0x80
}

type MessageIAmDispatcher struct {
	roads []uint16
}

func (m MessageIAmDispatcher) Type() uint8 {
	return 0x81
}

func buildMsg(r io.Reader) (Message, error) {
	msgType, err := ReadU8(r)
	if err != nil {
		return nil, err
	}
	switch msgType {
	case 0x20:
		plate, err := ReadString(r)
		if err != nil {
			return nil, err
		}
		timestamp, err := ReadU32(r)
		if err != nil {
			return nil, err
		}
		return &MessagePlate{
			plate:     plate,
			timestamp: timestamp,
		}, nil
	case 0x40:
		interval, err := ReadU32(r)
		if err != nil {
			return nil, err
		}
		return &MessageWantHeartbeat{
			interval: interval,
		}, nil
	case 0x80:
		road, err := ReadU16(r)
		if err != nil {
			return nil, err
		}
		mile, err := ReadU16(r)
		if err != nil {
			return nil, err
		}
		limit, err := ReadU16(r)
		if err != nil {
			return nil, err
		}
		return &MessageIAmCamera{
			road:  road,
			mile:  mile,
			limit: limit,
		}, nil
	case 0x81:
		roads, err := readU16Array(r)
		if err != nil {
			return nil, err
		}
		return &MessageIAmDispatcher{
			roads: roads,
		}, nil
	default:
		return nil, fmt.Errorf("invalid message type")
	}
}

func buildErrorMessage(s string) []byte {
	n := len(s)
	if n == 0 || n > 255 {
		log.Fatal("error msg empty or too long")
	}
	buf := make([]byte, n+2)
	buf[0] = 0x10
	buf[1] = uint8(n)
	for i := range n {
		c := s[i]
		buf[i+2] = c
	}
	return buf
}

func buildTicketMessage(ticket TicketInformation) []byte {
	n := len(ticket.car)
	if n == 0 || n > 255 {
		log.Fatal("plate number empty or too long")
	}
	buf := make([]byte, n+18)
	buf[0] = 0x21
	buf[1] = uint8(n)
	for i := range n {
		buf[i+2] = ticket.car[i]
	}
	binary.BigEndian.PutUint16(buf[n+2:], ticket.road)
	binary.BigEndian.PutUint16(buf[n+4:], ticket.mile1)
	binary.BigEndian.PutUint32(buf[n+6:], ticket.timestamp1)
	binary.BigEndian.PutUint16(buf[n+10:], ticket.mile2)
	binary.BigEndian.PutUint32(buf[n+12:], ticket.timestamp2)
	binary.BigEndian.PutUint16(buf[n+16:], ticket.speed)
	return buf
}

func buildHeartbeatMessage() []byte {
	return []byte{0x41}
}

func ReadU8(r io.Reader) (uint8, error) {
	buf := make([]byte, 1)
	n, err := io.ReadFull(r, buf)
	if err != nil {
		return 0, err
	}
	if n < 1 {
		return 0, errors.New("0 bytes read")
	}
	return uint8(buf[0]), nil
}

func ReadU16(r io.Reader) (uint16, error) {
	buf := make([]byte, 2)
	n, err := io.ReadFull(r, buf)
	if err != nil {
		return 0, err
	}
	if n < 2 {
		return 0, errors.New("less than 2 bytes read")
	}
	val := binary.BigEndian.Uint16(buf)
	return val, nil
}

func ReadU32(r io.Reader) (uint32, error) {
	buf := make([]byte, 4)
	n, err := io.ReadFull(r, buf)
	if err != nil {
		return 0, err
	}
	if n < 4 {
		return 0, errors.New("less than 4 bytes read")
	}
	val := binary.BigEndian.Uint32(buf)
	return val, nil
}

func ReadString(r io.Reader) (s string, e error) {
	s = ""
	l, e := ReadU8(r)
	if e != nil {
		return
	}
	if l == 0 {
		e = errors.New("zero length string")
		return
	}
	buf := make([]byte, l)
	n, e := io.ReadFull(r, buf)
	if e != nil {
		return
	}
	if n < int(l) {
		e = errors.New("couldn't read full string")
		return
	}
	s = string(buf)
	return
}

func readU16Array(r io.Reader) ([]uint16, error) {
	n, err := ReadU8(r)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, fmt.Errorf("empty array")
	}
	buf := make([]byte, 2*n)
	bytesRead, err := io.ReadFull(r, buf)
	if err != nil {
		return nil, err
	}
	if bytesRead < 2*int(n) {
		return nil, fmt.Errorf("couldn't read full array")
	}
	res := make([]uint16, n)
	for i := range n {
		res[i] = binary.BigEndian.Uint16(buf[2*i : 2*(+1)])
	}
	return res, nil
}

// PUZZLE: https://protohackers.com/problem/6
