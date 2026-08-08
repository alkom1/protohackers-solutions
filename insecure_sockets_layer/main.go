package main

import (
	"fmt"
	"io"
	"log"
	"net"
)

func main() {
	listener, err := net.Listen("tcp", ":9908")
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

type ISLStream struct {
	io.ReadWriteCloser
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	// 1. read and remember the cipher spec
	cipher, err := ParseCipherSpec(conn)
	if err != nil {
		log.Println("cipher error:", err)
		return
	}
	// 2. in-out simplify it
	cipher = SimplifyCipherSpec(cipher)
	// 3. if the cipher spec simplifies to no-op, close the connection
	if cipher.Kind == KindVariable {
		return
	}
	// 4. build and remember the reverse of the cipher spec

	// 1. receive message from connection -> decrypt it -> send it to the application layer
	// 2. receive payload from application layer -> encrypt it -> send it through the channel
	// how many goroutines per connection?
	//  1? ignore the separation into layers
}

func ParseCipherSpec(r io.Reader) (*Node, error) {
	x := Var()
	for {
		b, err := ReadU8(r)
		if err != nil {
			return nil, err
		}
		switch b {
		case uint8(OpInvalid):
			return x, nil
		case uint8(OpReverseBits):
			x = Call(OpReverseBits, x, 0)
		case uint8(OpXorN):
			n, err := ReadU8(r)
			if err != nil {
				log.Fatal(err)
			}
			x = Call(OpXorN, x, n)
		case uint8(OpXorPos):
			x = Call(OpXorPos, x, 0)
		case uint8(OpAddN):
			n, err := ReadU8(r)
			if err != nil {
				log.Fatal(err)
			}
			x = Call(OpAddN, x, n)
		case uint8(OpAddPos):
			x = Call(OpAddPos, x, 0)
		default:
			log.Fatal("unknown op", b)
		}
	}
}

func SimplifyCipherSpec(n *Node) *Node {
	if n == nil {
		return nil
	}

	if n.Kind == KindVariable {
		return n
	}

	n.Next = SimplifyCipherSpec(n.Next)

	// zero XORs disappear
	if n.Op == OpXorN && n.Arg == 0 {
		return n.Next
	}
	// zero additions disappear
	if n.Op == OpAddN && n.Arg == 0 {
		return n.Next
	}

	if n.Next.Kind != KindOperation {
		return n
	}

	// multiple additions in a row combine
	if n.Op == OpAddN && n.Next.Op == OpAddN {
		return Call(OpAddN, n.Next.Next, n.Arg+n.Next.Arg)
	}

	// multiple XORs in a row combine
	if n.Op == OpXorN && n.Next.Op == OpXorN {
		return Call(OpXorN, n.Next.Next, n.Arg^n.Next.Arg)
	}

	// two reversebits in a row disappear
	if n.Op == OpReverseBits && n.Next.Op == OpReverseBits {
		return n.Next.Next
	}

	return n
}

// MISC

func ReadU8(r io.Reader) (uint8, error) {
	buf := make([]byte, 1)
	n, err := io.ReadFull(r, buf)
	if err != nil {
		return 0, err
	}
	if n < 1 {
		return 0, fmt.Errorf("0 bytes read")
	}
	return uint8(buf[0]), nil
}

// CIPHER SPEC TREE IMPLEMENTATION

type NodeKind uint8

const (
	KindVariable NodeKind = iota
	KindOperation
)

type Op uint8

const (
	OpInvalid Op = iota
	OpReverseBits
	OpXorN
	OpXorPos
	OpAddN
	OpAddPos
)

type Node struct {
	Kind NodeKind

	Op   Op
	Next *Node

	Arg uint8
}

type OpInfo struct {
	name  string
	Arity uint8
}

var Operations = [...]OpInfo{
	OpInvalid:     {"<invalid>", 0},
	OpReverseBits: {"ReverseBits", 1},
	OpXorN:        {"XorN", 2},
	OpXorPos:      {"XorPos", 1},
	OpAddN:        {"AddN", 2},
	OpAddPos:      {"AddPos", 1},
}

func Var() *Node {
	return &Node{
		Kind: KindVariable,
	}
}

func Call(op Op, next *Node, arg uint8) *Node {
	info := Operations[op]

	switch info.Arity {
	case 0:
		return nil
	case 1:
		return &Node{
			Kind: KindOperation,
			Op:   op,
		}
	case 2:
		return &Node{
			Kind: KindOperation,
			Op:   op,
			Arg:  arg,
		}
	}

	log.Fatal("invalid operation arity")
	return nil
}

// how many goroutines per connection?
//  1? ignore the separation into layers
//  2? one low level, one application layer
//  2? one for each direction
//  3? low level one for each direction, application layer only one

// PUZZLE: https://protohackers.com/problem/8
