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
		// Only sync when the global state is Playing
		if p.State != Playing {
			// Wait until a new state is set
			<-p.StateInformer
			continue
		}

		p.sortClients()
		p.syncClients()
		time.Sleep(CheckInterval)
	}
}

// PauseHandler monitors the global notification channel
// and pauses/resumes all clients when someone pauses or resumes.
func (p *Pool) PauseHandler() {
	for {
		notif := <-p.Notification

		var play bool
		switch notif {
		case "Player.OnPause":
			LogInfo("Global pause triggered")
			play = false
			p.State = Paused
		case "Player.OnResume":
			LogInfo("Global play triggered")
			play = true
			p.State = Playing
		default:
			LogInfo("Unknown notification", notif)
		}

		var wg sync.WaitGroup
		for _, c := range p.Clients {
			c.ignoreStateNotification(play)
			wg.Add(1)
			go c.RequestWorker(c.playPayload(play), &wg)
		}

		// Wait until all clients are done and
		// inform every listener (if any) of a pause/resume
		wg.Wait()
		select {
		case p.StateInformer <- p.State:
		default:
		}
	}
}

// Play plays and pauses a client depending on
// boolean play.
func (c *Client) Play(play bool) {
	c.SendChannel <- c.playPayload(play)
}

// sortClients fetches all timestamps from each client,
// sets them and sorts clients by timestamp low->high
// (most behind->most ahead player).
func (p *Pool) sortClients() {
	var wg sync.WaitGroup

	for _, c := range p.Clients {
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
	sort.Sort(ByTimestamp(p.Clients))
}

// ignoreStateNotification is used to ignore a play/pause notification based on the
// current state of the client. The purpose of this is to avoid
// ignoring too many notifications since the client only sends an notification
// if the action was successful, e.g. if the client is playing and a
// pause request was sent.
// Play = true means ignore a play notification and false means ignore a pause
// notification.
func (c *Client) ignoreStateNotification(play bool) {
	params := map[string]interface{}{
		"playerid":   DefaultPlayerID,
		"properties": []string{"speed"},
	}

	payload := NewBaseSend("Player.GetProperties", params, PlayerGetPropertiesSpeed)

	var wg sync.WaitGroup
	wg.Add(1)
	go c.RequestWorker(payload, &wg)
	wg.Wait()

	// Paused and ignoring play
	if c.State == 0 && play {
		c.IgnoreCount++
	}

	// Playing and ignoring pause
	if c.State == 1 && !play {
		c.IgnoreCount++
	}
}

// playPayload creates a payload used for playing/pausing
// the client.
func (c *Client) playPayload(play bool) BaseSend {
	params := map[string]interface{}{
		"playerid": DefaultPlayerID,
		"play":     play,
	}

	return NewBaseSend("Player.PlayPause", params, 0)
}

// syncClients pauses the clients in a Pool (most behind to most ahead)
// in a way that they all sync up. Returns when all clients are synced up.
func (p *Pool) syncClients() {
	// Only sync when most ahead client - most behind > MaxDiff
	if p.Clients[len(p.Clients)-1].TimeDifference(p.Clients[0]) < MaxDiff {
		LogInfo("Not enough desync, do nothing")
		return
	}

	// Used to determine when all clients (except the one most behind)
	// have been sent a pause signal
	var wgDone sync.WaitGroup

	// Play client behind (in case it is paused)
	// (This will trigger a global play in PauseHandler which is fine since we're
	// pausing them anyway)
	p.Clients[0].Play(true)

	// Pause all clients ahead
	for i := 1; i < len(p.Clients); i++ {
		ahead := p.Clients[i]
		timeDiff := ahead.TimeDifference(p.Clients[0])
		LogInfof("%s desync between %s and %s\n", timeDiff.String(),
			p.Clients[0].Description(), ahead.Description())

		wgDone.Add(1)
		go ahead.pauseClient(timeDiff, &wgDone)
	}

	// Wait until all clients are done syncing
	wgDone.Wait()
	LogInfo("Syncing completed")
}

// pauseClient pauses a client for duration amount of time.
func (c *Client) pauseClient(duration time.Duration, wgDone *sync.WaitGroup) {
	defer wgDone.Done()

	// Pause
	c.ignoreStateNotification(false) // Ignore the pause notification
	LogInfof("Pausing %s for %s\n", c.Description(), duration.String())
	var wg sync.WaitGroup
	wg.Add(1)
	go c.RequestWorker(c.playPayload(false), &wg)
	wg.Wait()

	// Wait until the pause duration has passed
	// or until the global state has been changed
	select {
	case <-time.After(duration):
		break
	case <-c.Pool.StateInformer:
		return
	}

	// Play
	c.ignoreStateNotification(true) // Ignore the play notification
	wg.Add(1)
	go c.RequestWorker(c.playPayload(true), &wg)
	wg.Wait()
	LogInfof("Unpausing %s\n", c.Description())
}
