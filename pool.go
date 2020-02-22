package main

import (
	"bufio"
	"os"
	"strings"
	"sync"
)

// Client state enum (starts at 1)
const (
	// All clients playing
	Playing = iota + 1

	// All clients paused
	Paused

	// Used to ignore notifications during certain operations
	// such as pausing all clients
	Busy
)

// Pool is a collection of all connected clients.
type Pool struct {
	Clients []*Client

	// Notification channel shared by all clients, used to notify
	// other clients of incoming notifications such as one client being paused.
	Notification chan string

	// Global state of clients (Playing, paused..)
	// Controlled and checked using methods State and ChangeState
	state int
	mux   sync.Mutex

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
		StateInformer: make(chan int),
		state:         Playing,
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

// ChangeState is used to change the state from multiple goroutines
// without encountering conflicts.
func (p *Pool) ChangeState(state int) {
	p.mux.Lock()
	p.state = state
	p.mux.Unlock()
}

// State returns the global client state.
func (p *Pool) State() int {
	p.mux.Lock()
	defer p.mux.Unlock()
	return p.state
}
