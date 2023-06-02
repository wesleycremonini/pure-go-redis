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

	// waits for the next connection on the network
	conn, err := newtwork.Accept()
	log.Println("There is a new connection: ", conn)
	if err != nil {
		log.Fatal(err)
	}

	// keep the main goroutine alive (only for testing purposes)
	select{}
}
