package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"regexp"
)

var (
	joined = make(chan JoinRequest)
	left   = make(chan net.Conn)
	say    = make(chan Say)
	users  = make(map[net.Conn]string)
)

type JoinRequest struct {
	User     net.Conn
	nickname string
}
type Say struct {
	User net.Conn
	msg  string
}

func main() {
	listener, err := net.Listen("tcp", ":9903")
	if err != nil {
		log.Fatal("Error listening:", err)
	}

	defer closeAll()
	defer listener.Close()

	go router()

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

func closeAll() {
	for user := range users {
		user.Close()
	}
}

func router() {
	for {
		select {
		case uJoin := <-joined:
			// announce the new user to other users
			// send the new user list of all already connected users (starts with *)
			fmt.Fprintf(uJoin.User, "* present users: ")
			for i := range users {
				fmt.Fprintf(i, "* %s has joined.\n", uJoin.nickname)
				fmt.Fprintf(uJoin.User, "%s ", users[i])
			}
			fmt.Fprintf(uJoin.User, "\n")
			users[uJoin.User] = uJoin.nickname
		case uLeft := <-left:
			name := users[uLeft]
			delete(users, uLeft)
			// when user disconnects, send all other users an announcement (starts with *)
			for i := range users {
				fmt.Fprintf(i, "* %s has left.\n", name)
			}
		case s := <-say:
			// whenever user sends a message, relay it to all other users
			// do not send it to: users that have not entered username, the sender
			// format: "[NAME] MESSAGE"
			for i := range users {
				if i != s.User {
					fmt.Fprintf(i, "[%s] %s\n", users[s.User], s.msg)
				}
			}
		}
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	conn.Write([]byte("What is your name?\n"))

	// read the username from the connection
	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		log.Println("couldnt get username")
		return
	}
	name := scanner.Text()

	// validate it
	if len(name) == 0 || !is_alphanum(name) {
		// invalid name
		conn.Write([]byte("invalid username\n"))
		return
	}

	// join the chat room
	joined <- JoinRequest{
		User:     conn,
		nickname: name,
	}

	for scanner.Scan() {
		line := scanner.Text()
		say <- Say{
			User: conn,
			msg:  line,
		}
	}
	if err := scanner.Err(); err != nil {
		log.Println("scanner error", err)
	}

	left <- conn
}

func is_alphanum(word string) bool {
	return regexp.MustCompile(`^[a-zA-Z0-9]*$`).MatchString(word)
} // https://www.geeksforgeeks.org/go-language/golang-program-to-check-if-the-string-is-alphanumeric/

// INSPIRED BY:
//  https://chriswilcox.dev/blog/2024/04/09/Scan-vs-Read-in-bufio.html
//  https://stackoverflow.com/questions/36417199/how-to-broadcast-message-using-channel/49877632#49877632
//  https://github.com/alexballas/random-scripts/blob/main/tcpchatserver/cmd/v1/server/server.go

// PUZZLE: https://protohackers.com/problem/3
