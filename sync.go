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
		if p.State() != Playing {
			state := <-p.StateInformer
			if state != Playing {
				LogInfo(state)
				continue // Check again
			}
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

		// Don't pause/resume when in the middle
		// of an operation affecting all clients
		if p.State() == Busy {
			continue
		}

		var play bool
		switch notif {
		case "Player.OnPause":
			LogInfo("Trigger global pause with state", p.State())
			play = false
			p.ChangeState(Paused)
		case "Player.OnResume":
			LogInfo("Trigger global resume with state", p.State())
			play = true
			p.ChangeState(Playing)
		default:
			continue
		}

		var wg sync.WaitGroup
		for _, c := range p.Clients {
			wg.Add(1)
			go c.PlayAwait(play, &wg)
		}

		// Wait until all clients are done and
		// inform every listener of a pause/resume
		wg.Wait()
		p.StateInformer <- p.State()
	}
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

// PlayBlock plays/pauses and waits until the client has
// started playing/paused.
func (c *Client) PlayBlock(play bool) {
	// Check for state notification not reliable
	// since if the player is already in the same
	// state it won't change.
	// for {
	// 	method := <-c.Notification
	// 	if (play && method == "Player.OnResume") ||
	// 		(!play && method == "Player.OnPause") {
	// 		return
	// 	}
	// }

	var wg sync.WaitGroup
	wg.Add(1)
	go c.RequestWorker(c.playPayload(play), &wg)
	wg.Wait()
}

// PlayAwait plays/pauses and waits until the client has
// started playing/paused and executes wg.Done().
func (c *Client) PlayAwait(play bool, wg *sync.WaitGroup) {
	c.PlayBlock(play)
	wg.Done()
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

// Play plays and pauses a client depending on
// boolean play.
func (c *Client) Play(play bool) {
	c.SendChannel <- c.playPayload(play)
}

// syncClients pauses the clients in a Pool (most behind to most ahead)
// in a way that they all sync up. Returns when all clients are synced up.
func (p *Pool) syncClients() {
	// Only sync when most ahead client - most behind > MaxDiff
	if p.Clients[len(p.Clients)-1].TimeDifference(p.Clients[0]) < MaxDiff {
		LogInfo("Not enough desync, do nothing")
		return
	}

	// Set state to busy to avoid picking up notifications
	// caused by this method
	p.ChangeState(Busy)

	// Used to determine when all clients (except the one most behind)
	// have been sent a pause signal
	var wgDone sync.WaitGroup

	// Used to determine when all clients have been unpaused again
	var wgPaused sync.WaitGroup

	for i := 1; i < len(p.Clients); i++ {
		ahead := p.Clients[i]
		timeDiff := ahead.TimeDifference(p.Clients[0])
		LogInfof("%s desync between %s and %s\n", timeDiff.String(),
			p.Clients[0].Description(), ahead.Description())

		wgDone.Add(1)
		wgPaused.Add(1)
		go ahead.pauseClient(timeDiff, &wgDone, &wgPaused)
	}

	// Set state to paused when ALL clients have been paused
	// so we can begin listening to global notifications
	go func() {
		wgPaused.Wait()
		p.ChangeState(Paused)
	}()

	interrupt := make(chan struct{})

	// Interrupt handler for global pause
	go func() {
		for {
			state := <-p.StateInformer
			if state == Paused {
				LogInfo("Someone paused, interrupting sync")
				p.ChangeState(Paused)
				interrupt <- struct{}{}
				return
			}
		}
	}()

	// Interrupt handler for when all players resumed
	go func() {
		wgDone.Wait()
		p.ChangeState(Playing)
		LogInfo("Syncing completed")
		interrupt <- struct{}{}
	}()

	// Wait until global pause or all clients resumed
	<-interrupt
}

// pauseClient pauses a client for duration amount of time.
func (c *Client) pauseClient(duration time.Duration, wgDone *sync.WaitGroup,
	wgPaused *sync.WaitGroup) {
	defer wgDone.Done()

	// Pause
	LogInfof("Pausing %s for %s\n", c.Description(), duration.String())
	c.PlayAwait(false, wgPaused)
	LogInfo("Done pausing hehe")

	// If the player is paused, break out to avoid unpausing
	// else just wait until duration is over to unpause.
	timeUp := time.After(duration)

continueWait:
	select {
	case state := <-c.Pool.StateInformer:
		LogInfo("got a state", state, c.Description())
		if state == Paused {
			LogInfo("Someone paused, interrupting sync")
			return
		}
		break continueWait
	case <-timeUp:
		break
	}

	// Play (change state temporarily to ignore notification)
	saveState := c.Pool.State()
	c.Pool.ChangeState(Busy)
	c.PlayBlock(true)
	c.Pool.ChangeState(saveState)
	LogInfof("Unpausing %s\n", c.Description())
}
