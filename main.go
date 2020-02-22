package main

import (
	"time"
)

const (
	// Path is the path to identifiers.
	Path = "identifiers.txt"

	// MaxDiff is the max timestamp difference.
	MaxDiff = 2 * time.Second

	// DefaultPlayerID is the default Kodi player id.
	DefaultPlayerID = 1

	// CheckInterval is a duration that defines how often
	// to check for desyncs.
	CheckInterval = 2 * time.Second
)

func main() {
	pool := NewPoolFromFile(Path)
	if len(pool.Clients) < 1 {
		LogFatal("No available clients")
	}

	// Register global pause handler
	go pool.PauseHandler()

	// Register client sync handler
	go pool.SyncHandler()

	select {}
}
