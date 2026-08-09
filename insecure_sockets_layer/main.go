package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
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

type EncryptedStream struct {
	src    io.ReadWriteCloser
	spec   *Node
	inPos  byte
	outPos byte
}

func (r *EncryptedStream) Read(p []byte) (int, error) {
	// Read reads up to len(p) bytes into p. Returns number of bytes read and any error encountered.
	n, err := r.src.Read(p)
	if err != nil {
		return 0, err
	}

	for i := range n {
		p[i] = DecryptByte(p[i], r.spec, r.inPos)
		r.inPos += 1
	}

	return n, nil
}

func (r *EncryptedStream) Write(p []byte) (int, error) {
	buf := make([]byte, len(p))
	for i, b := range p {
		buf[i] = EncryptByte(b, r.spec, r.outPos)
		r.outPos += 1
	}

	return r.src.Write(buf) // this can fail and cause desync
}

func (r *EncryptedStream) Close() error {
	return r.src.Close()
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
		log.Println("no-op cipher spec")
		return
	}

	encryptedStream := &EncryptedStream{
		src:  conn,
		spec: cipher,
	}

	scanner := bufio.NewScanner(encryptedStream)
	for scanner.Scan() {
		line := scanner.Text()
		order := processLine(line)
		fmt.Fprintf(encryptedStream, "%s\n", order)
	}
	if err := scanner.Err(); err != nil {
		log.Println(err)
	}
}

func processLine(s string) string {
	split := strings.Split(s, ",")

	largest := 0
	largestName := ""
	for _, t := range split {
		cStr, _, found := strings.Cut(t, "x")
		if !found {
			log.Fatal("x not found")
		}
		c, err := strconv.Atoi(cStr)
		if err != nil {
			log.Fatal(err)
		}
		if c > largest {
			largest = c
			largestName = t
		}
	}
	return largestName
}

// CIPHER STUFF

func ParseCipherSpec(r io.Reader) (*Node, error) {
	x := &Node{
		Kind: KindVariable,
	}
	for {
		b, err := ReadU8(r)
		if err != nil {
			return nil, err
		}
		switch b {
		case 0: // end
			return x, nil
		case 1: // reverse bits
			x = &Node{
				Kind: KindOperation,
				Op:   OpReverseBits,
				Next: x,
			}
		case 2: // xor n
			n, err := ReadU8(r)
			if err != nil {
				log.Fatal(err)
			}
			x = &Node{
				Kind: KindOperation,
				Op:   OpXor,
				Arg:  n,
				Next: x,
			}
		case 3: // xor pos
			x = &Node{
				Kind: KindOperation,
				Op:   OpXor,
				Pos:  1,
				Next: x,
			}
		case 4: // add n
			n, err := ReadU8(r)
			if err != nil {
				log.Fatal(err)
			}
			x = &Node{
				Kind: KindOperation,
				Op:   OpAddN,
				Arg:  n,
				Next: x,
			}
		case 5: // add pos
			x = &Node{
				Kind: KindOperation,
				Op:   OpAddPos,
				Next: x,
			}
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
	if n.Op == OpXor && n.Arg == 0 && n.Pos == 0 {
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
		return SimplifyCipherSpec(&Node{
			Kind: KindOperation,
			Op:   OpAddN,
			Next: n.Next.Next,
			Arg:  n.Arg + n.Next.Arg,
		})
	}

	// multiple XORs in a row combine
	if n.Op == OpXor && n.Next.Op == OpXor {
		return SimplifyCipherSpec(&Node{
			Kind: KindOperation,
			Op:   OpXor,
			Next: n.Next.Next,
			Arg:  n.Arg ^ n.Next.Arg,
			Pos:  n.Pos ^ n.Next.Pos,
		})
	}

	// two reversebits in a row disappear
	if n.Op == OpReverseBits && n.Next.Op == OpReverseBits {
		return n.Next.Next
	}

	return n
}

func EncryptByte(b byte, c *Node, p byte) byte {
	if c == nil || c.Kind == KindVariable {
		return b
	}

	b = EncryptByte(b, c.Next, p)

	switch c.Op {
	case OpReverseBits:
		return reverseBits(b)
	case OpXor:
		if c.Pos > 0 {
			b = b ^ p
		}
		return b ^ c.Arg
	case OpAddN:
		return b + c.Arg
	case OpAddPos:
		return b + p
	}

	return b
}

func DecryptByte(b byte, c *Node, p byte) byte {
	if c == nil || c.Kind == KindVariable {
		return b
	}

	switch c.Op {
	case OpReverseBits:
		b = reverseBits(b)
	case OpXor:
		if c.Pos > 0 {
			b = b ^ p
		}
		b = b ^ c.Arg
	case OpAddN:
		b = b - c.Arg
	case OpAddPos:
		b = b - p
	}

	return DecryptByte(b, c.Next, p)
}

func reverseBits(b byte) byte {
	var r byte
	for i := range 8 {
		r |= ((b >> i) & 1) << (7 - i)
	}
	return r
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
	OpXor
	OpAddN
	OpAddPos
)

type Node struct {
	Kind NodeKind

	Op   Op
	Next *Node

	Pos uint8 // for xor pos
	Arg uint8 // for xor n, add n
}

// how many goroutines per connection?
//  1? ignore the separation into layers
//  2? one low level, one application layer
//  2? one for each direction
//  3? low level one for each direction, application layer only one

// PUZZLE: https://protohackers.com/problem/8
