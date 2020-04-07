package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Client defines a Kodi player.
type Client struct {
	Host        string          // Hostname to connect to
	User        string          // Username
	Password    string          // Password
	Timestamp   time.Duration   // Video timestamp
	State       int             // Playing state, paused = 0, playing = 1
	Connection  *websocket.Conn // WS connection
	SendChannel chan BaseSend   // Channel used to send messages over WS

	// ActiveOperations is a map of SendRecv.ID: <operation type constant>
	// used to keep track of responses to requests (recv to send)
	ActiveOperations map[int]int

	// OperationDone is a channel used to notify listeners of completed operations.
	// The integer contains the SendRecv ID associated with the operation.
	OperationDone chan int

	// Pool contains all clients and information shared by all
	// clients such as a global notification channel and the global state.
	Pool *Pool

	// Notification is a channel specific to the client used
	// to transport the notification method.
	Notification chan string

	// IgnoreCount is used to prevent sending notifications triggered
	// by the program (e.g. when pausing/playing to sync clients) to
	// all clients (the pool).
	IgnoreCount int
}

// ByTimestamp implements sort.Interface for []Client
// based on the Timestamp field.
type ByTimestamp []*Client

func (bt ByTimestamp) Len() int           { return len(bt) }
func (bt ByTimestamp) Swap(i, j int)      { bt[i], bt[j] = bt[j], bt[i] }
func (bt ByTimestamp) Less(i, j int) bool { return bt[i].Timestamp < bt[j].Timestamp }

// NewClient creates a connected client from credentials and adds
// it to the pool.
func (p *Pool) NewClient(host, user, pass string) {
	newClient := Client{
		Host:             host,
		User:             user,
		Password:         pass,
		SendChannel:      make(chan BaseSend),
		State:            1,
		ActiveOperations: map[int]int{},
		OperationDone:    make(chan int),
		Pool:             p,
		Notification:     make(chan string),
		IgnoreCount:      0,
	}

	if err := newClient.Connect(); err != nil {
		LogFatalf("Failed connecting to %s\n\tReason: %s\n",
			newClient.Description(), err)
	}

	p.Clients = append(p.Clients, &newClient)
	LogInfof("Connected to %s\n", newClient.Description())
}

// Connect establishes a websocket connection and sets it in the
// Client struct.
func (c *Client) Connect() error {
	URI := c.wsURI()
	header := &http.Header{}

	// TODO: not even sure if this does anything :thinking:
	c.addAuthHeader(header)

	var err error
	c.Connection, _, err = websocket.DefaultDialer.Dial(URI.String(), *header)
	if err != nil {
		return err
	}

	go c.readHandler()
	go c.sendHandler()

	return nil
}

// RequestWorker is a worker that performs a request to Client using payload
// and runs until the corresponding response is received.
func (c *Client) RequestWorker(payload BaseSend, wg *sync.WaitGroup) {
	c.SendChannel <- payload

	for {
		opID := <-c.OperationDone
		if opID == payload.ID {
			wg.Done()
			return
		}
	}
}

// Description prints the string representation of a Client.
func (c *Client) Description() string {
	return fmt.Sprintf("%s (%s)", c.Host, c.User)
}

// TimeDifference returns the difference c.Timestamp - other.Timestamp.
func (c *Client) TimeDifference(other *Client) time.Duration {
	return c.Timestamp - other.Timestamp
}

// sendHandler is responsible for dispatching messages.
func (c *Client) sendHandler() {
	for {
		payload := <-c.SendChannel
		if err := c.Connection.WriteJSON(payload); err != nil {
			LogWarn(err)
		}

		// Add request to active operations
		c.ActiveOperations[payload.ID] = payload.Operation
	}
}

// readHandler is responsible for handling incoming messages.
func (c *Client) readHandler() {
	for {
		var result BaseRecv
		// var debug *json.RawMessage
		// if err := c.Connection.ReadJSON(&debug); err != nil {
		// 	LogWarn(err)
		// }
		// b, _ := debug.MarshalJSON()
		// if err := json.Unmarshal(b, &result); err != nil {
		// 	LogWarn(err)
		// }
		// LogInfo("debug", string(b))
		if err := c.Connection.ReadJSON(&result); err != nil {
			LogWarn(err)
		}

		if result.Error != nil {
			LogWarn(result.ToString())
		}

		if result.ID == 0 { // Notification always has ID 0
			c.handleNotification(result)
		} else { // Not 0 implies a unique ID that we generated, expecting a response
			c.handleResponse(result)
		}
	}
}

// handleNotification is responsible for handling notifications sent
// from clients.
func (c *Client) handleNotification(notif BaseRecv) {
	// Non-blocking notification send to client specific channel
	select {
	case c.Notification <- notif.Method:
	default:
	}

	switch notif.Method {
	case "Player.OnResume":
		c.State = 1
		LogInfof("Resumed %s\n", c.Description())
	case "Player.OnPause":
		c.State = 0
		LogInfof("Paused %s\n", c.Description())
	}

	// Don't send pool-wide notification if counter is over 0
	if c.IgnoreCount > 0 {
		c.IgnoreCount--
		return
	}

	// Non-blocking notification send to pool-wide channel
	select {
	case c.Pool.Notification <- notif.Method:
	default:
	}

}

// handleResponse is responsible for handling responses
// made to requests made in SendHandler.
func (c *Client) handleResponse(response BaseRecv) {
	operation := c.ActiveOperations[response.ID]

	switch operation {
	case PlayerGetPropertiesTime: // Set client time when this operation is done
		var pp PlayerProperties
		if err := json.Unmarshal(*response.Result, &pp); err != nil {
			LogWarn(err)
		}

		var pt PlayerTime
		if err := json.Unmarshal(*pp.Time, &pt); err != nil {
			LogWarn(err)
		}

		c.Timestamp = pt.ToDuration()
	case PlayerGetPropertiesSpeed: // Set client state (paused = 0, playing = 1)
		var pp PlayerProperties
		if err := json.Unmarshal(*response.Result, &pp); err != nil {
			LogWarn(err)
		}

		var speed int
		if err := json.Unmarshal(*pp.Speed, &speed); err != nil {
			LogWarn(err)
		}

		c.State = speed
	}

	// Delete the operation since we got a response
	delete(c.ActiveOperations, response.ID)

	// Notify through channel (non-blocking)
	// Default is necessary in cases where we don't care
	// about the response and ignore reading from c.OperationDone
	select {
	case c.OperationDone <- response.ID:
	default:
	}
}

// wsURI returns the websocket URI for a Client.
func (c *Client) wsURI() url.URL {
	return url.URL{Scheme: "ws", Host: c.Host, Path: "/jsonrpc"}
}

// addAuthHeader adds the Authorization header using the client
// credentials to a http header.
func (c *Client) addAuthHeader(h *http.Header) {
	bCreds := []byte(fmt.Sprintf("%s:%s", c.User, c.Password))
	sEnc := base64.StdEncoding.EncodeToString(bCreds)
	h.Add("Authorization", fmt.Sprintf("Basic %s", sEnc))
}
