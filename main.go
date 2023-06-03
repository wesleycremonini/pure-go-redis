package main

import (
	"log"
	"net"
	"sync"
)

// cache stores the keys
var cache sync.Map

func main() {
	// starts a new listener
	newtwork, err := net.Listen("tcp", ":5000")
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Listening on tcp://0.0.0.0:5000")

	// starts a loop to handle multiple connections
	for {
		// waits for the next connection on the network
		conn, err := newtwork.Accept()
		log.Println("Connected: ", conn)
		if err != nil {
			log.Fatal(err)
		}

		go newSession(conn)
	}
}

// newSession handles the new client's session.
// Also responsible for listening and responding to commands.
func newSession(conn net.Conn) {
	// defer the close to guarantee the connection is closed when the session finishes
	defer func() {
		log.Println("Disconnected: ", conn)
		conn.Close()
	}()

	// try recovering from an unexpected error (panics) in the current session
	// this will prevent the server from dying
	defer func() {
		if err := recover(); err != nil {
			log.Println("ERR: trying to recover: ", err)
		}
	}()

	p := NewParser(conn)
	for {
		cmd, err := p.command()
		if err != nil {
			log.Println("Error", err)
			conn.Write([]uint8("-ERR " + err.Error() + "\r\n"))
			break
		}
		if !cmd.handle() {
			// close the sesseion if cmd returns false
			break
		}
	}
}
