package main

import (
	"bufio"
	"os"
	"strings"
)

// Client state enum
const (
	Playing = iota + 1
	Paused
)

// Pool is a collection of all connected clients.
type Pool struct {
	Clients []*Client

	// Notification channel shared by all clients, used to notify
	// other clients of incoming notifications such as one client being paused.
	Notification chan string

	// Global state of clients (Playing, paused..)
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

	var clients []*Client
	notifChan := make(chan string)
	stateChan := make(chan int)

	// Read line by line
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		// Skip if hashtag
		if scanner.Text()[0] == byte('#') {
			continue
		}

		identifier := strings.Split(scanner.Text(), ",")
		if err != nil {
			LogFatal(err)
		}

		newClient, err := NewClient(identifier[0], identifier[1], identifier[2],
			notifChan, stateChan)

		if err == nil {
			clients = append(clients, newClient)
		}
	}

	return &Pool{
		Clients:       clients,
		Notification:  notifChan,
		State:         Playing,
		StateInformer: stateChan,
	}
}
