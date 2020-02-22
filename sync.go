package main

import (
	"sort"
	"sync"
	"time"
)

// SyncHandler constantly checks and attempts to sync
// client timestamps.
func (p *Pool) SyncHandler() {
	for {
		// Only sync when playing
		if p.State != Playing {
			select {
			case state := <-p.StateInformer:
				if state != Playing {
					continue
				}
			}
		}

		sortClients(p.Clients)
		syncClients(p.Clients)
		time.Sleep(CheckInterval)
	}
}

// PauseHandler monitors the global notification channel
// and pauses/resumes all clients when someone pauses or resumes.
func (p *Pool) PauseHandler() {
	// Buffered channel to ignore the notifications triggered
	// by pausing/playing all
	excess := make(chan struct{}, len(p.Clients))

	for {
		select {
		case <-excess:
			continue
		case notif := <-p.Notification:
			var play bool
			switch notif {
			case "Player.OnPause":
				LogInfo("Trigger global pause")
				play = false
				p.State = Paused
			case "Player.OnResume":
				LogInfo("Trigger global resume")
				play = true
				p.State = Playing
			}

			for _, c := range p.Clients {
				excess <- struct{}{}
				c.play(play)
			}

			p.StateInformer <- p.State
		}
	}
}

// play plays and pauses a client depending on
// boolean play.
func (c *Client) play(play bool) {
	params := map[string]interface{}{
		"playerid": DefaultPlayerID,
		"play":     play,
	}
	payload := NewBaseSend("Player.PlayPause", params, 0)
	c.SendChannel <- payload
}

// pauseClient pauses a client for duration amount of time.
func (c *Client) pauseClient(duration time.Duration, wg *sync.WaitGroup) {
	defer wg.Done()

	// Pause
	LogInfof("Pausing %s for %s\n", c.Description(), duration.String())
	c.play(false)

	// If the player is paused, break out to avoid unpausing
	// else just wait until duration is over to unpause.
	timeUp := time.After(duration)

keepWaiting:
	select {
	case state := <-c.StateInformer:
		LogInfo("got a state", state)
		if state == Paused {
			LogInfo("Someone paused, interrupting sync")
			return
		}
		break keepWaiting
	case <-timeUp:
		break
	}

	// Play
	c.play(true)
	LogInfof("Unpausing %s\n", c.Description())
}

// syncClients pauses a sorted list (most behind to most ahead)
// of clients in a way that they all sync up. Returns when
// all clients are synced up.
func syncClients(clients []*Client) {
	// Only sync when most ahead client - most behind > MaxDiff
	if clients[len(clients)-1].TimeDifference(clients[0]) < MaxDiff {
		LogInfo("Not enough desync, do nothing")
		return
	}

	LogInfof("Unpausing %s\n", clients[0].Description())
	clients[0].play(true)

	var wg sync.WaitGroup
	for i := 1; i < len(clients); i++ {
		ahead := clients[i]
		timeDiff := ahead.TimeDifference(clients[0])
		LogInfof("%s desync between %s and %s\n", timeDiff.String(),
			clients[0].Description(), ahead.Description())

		wg.Add(1)
		go ahead.pauseClient(timeDiff, &wg)
	}

	wg.Wait()
	LogInfo("Syncing final")
}

// sortClients fetches all timestamps from each client,
// sets them and sorts clients by timestamp low->high
// (most behind->most ahead player).
func sortClients(clients []*Client) {
	var wg sync.WaitGroup

	for _, c := range clients {
		params := map[string]interface{}{
			"playerid":   DefaultPlayerID,
			"properties": []string{"time"},
		}

		payload := NewBaseSend("Player.GetProperties", params, PlayerGetPropertiesTime)

		wg.Add(1)
		go c.RequestWorker(payload, &wg)
	}

	// Wait until all responses are received and sort the clients
	wg.Wait()

	// LogInfo("All workers finished, sorting")
	sort.Sort(ByTimestamp(clients))
}
