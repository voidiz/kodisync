package main

import (
	"sort"
	"sync"
)

// SortClients fetches all timestamps from each client,
// sets them and sorts clients by timestamp low->high
// (most behind->most ahead player).
func SortClients(clients []Client) {
	var wg sync.WaitGroup

	for _, c := range clients {
		params := map[string]interface{}{
			"playerid":   DefaultPlayerID,
			"properties": []string{"time"},
		}

		payload := NewBaseSend("Player.GetProperties", params)
		c.ActiveOperations[payload.ID] = PlayerGetPropertiesTime

		wg.Add(1)
		LogInfof("Create worker %d\n", payload.ID)
		go c.RequestWorker(payload, &wg)
	}

	// Wait until all responses are received and sort the clients
	wg.Wait()
	LogInfo("All workers finished, sorting")
	sort.Sort(ByTimestamp(clients))
}
