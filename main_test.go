package main

import (
	"net"
	"testing"
	"time"
)

func TestMain(t *testing.T) {

	// starts the main goroutine and waits for the newtwork to be up
	go main()
	time.Sleep(1 * time.Second)

	// try 5 connections
	for i := 0; i < 5; i++ {
		// try connecting
		conn, err := net.Dial("tcp", "localhost:5000")
		if err != nil {
			t.Fatal("Failed to connect to server: ", err)
		}
		defer conn.Close()

		if conn == nil {
			t.Fatal("Connection is nil")
		}

		// wait to start another connection
		time.Sleep(200 * time.Millisecond)
	}

	// wait for all connections to finish
	time.Sleep(4 * time.Second)
}
