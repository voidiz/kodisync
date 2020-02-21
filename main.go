package main

import (
	"time"
)

const (
	// Path is the path to identifiers.
	Path = "identifiers.txt"

	// MaxDiff is the max timestamp difference.
	MaxDiff = 2 // Seconds

	// DefaultPlayerID is the default Kodi player id.
	DefaultPlayerID = 1

	// CheckInterval is a duration that defines how often
	// to check for desyncs.
	CheckInterval = time.Second * 2
)

func main() {
	clients := InitializeClients(Path)
	ticker := time.NewTicker(time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				LogInfo("Fetching timestamps")
				SortClients(clients)
			}
		}
	}()

	select {}
}
