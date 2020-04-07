package main

import (
	"bufio"
	"os"
	"strings"
)

// Client state enum (starts at 1)
const (
	// All clients playing
	Playing = iota + 1

	// All clients paused
	Paused
)

// Pool is a collection of all connected clients.
type Pool struct {
	Clients []*Client

	// Notification channel shared by all clients, used to notify
	// other clients of incoming notifications such as one client being paused.
	Notification chan string

	// Used to keep track of the global state of the clients
	State int

	// Channel to inform clients of pool State
	StateInformer chan int
}

// NewPoolFromFile creates a Pool from client
// identifiers written in a text file located at path.
// Format: hostname:port,username,password
func NewPoolFromFile(path string) *Pool {
	file, err := os.Open(path)
	if err != nil {
		LogFatal(err)
	}

	pool := &Pool{
		Clients:       []*Client{},
		Notification:  make(chan string),
		State:         Playing,
		StateInformer: make(chan int),
	}

	// Read line by line
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		// Skip if hashtag
		if scanner.Text()[0] == byte('#') {
			continue
		}

		identifier := strings.Split(scanner.Text(), ",")
		pool.NewClient(identifier[0], identifier[1], identifier[2])
	}

	return pool
}
