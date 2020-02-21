package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
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
	Connection  *websocket.Conn // WS connection
	SendChannel chan BaseSend   // Channel used to send messages over WS

	// ActiveOperations is a map of SendRecv.ID: <operation type constant>
	// used to keep track of responses to requests (recv to send)
	ActiveOperations map[int]int

	// OperationDone is a channel used to notify listeners of completed operations.
	// The integer contains the SendRecv ID associated with the operation.
	OperationDone chan int
}

// ByTimestamp implements sort.Interface for []Client
// based on the Timestamp field.
type ByTimestamp []Client

func (bt ByTimestamp) Len() int           { return len(bt) }
func (bt ByTimestamp) Swap(i, j int)      { bt[i], bt[j] = bt[j], bt[i] }
func (bt ByTimestamp) Less(i, j int) bool { return bt[i].Timestamp < bt[j].Timestamp }

// InitializeClients creates and connects clients from client
// identifiers written in a text file located at path.
// Format: hostname:port,username,password
func InitializeClients(path string) []Client {
	file, err := os.Open(path)
	if err != nil {
		LogFatal(err)
	}

	// Read line by line
	var clients []Client
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

		newClient := Client{
			Host:             identifier[0],
			User:             identifier[1],
			Password:         identifier[2],
			SendChannel:      make(chan BaseSend),
			ActiveOperations: map[int]int{},
			OperationDone:    make(chan int),
		}

		if err := newClient.Connect(); err != nil {
			LogWarn(err)
		}
		LogInfof("Connected to %s\n", newClient.Description())

		clients = append(clients, newClient)
	}

	return clients
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
		select {
		case opID := <-c.OperationDone:
			if opID == payload.ID {
				LogInfof("Worker %d finished\n", payload.ID)
				wg.Done()
				return
			}
		}

	}
}

// Description prints the string representation of a Client.
func (c Client) Description() string {
	return fmt.Sprintf("%s (%s)", c.Host, c.User)
}

// sendHandler is responsible for dispatching messages.
func (c *Client) sendHandler() {
	for {
		select {
		case payload := <-c.SendChannel:
			if err := c.Connection.WriteJSON(payload); err != nil {
				LogWarn(err)
			}
		}
	}
}

// readHandler is responsible for handling incoming messages.
func (c *Client) readHandler() {
	for {
		var result BaseRecv
		if err := c.Connection.ReadJSON(&result); err != nil {
			LogWarn(err)
		}

		if result.Error != nil {
			LogWarn(result.ToString())
		}

		// Notification
		c.handleNotification(result)

		// Response (to request made in SendHandler)
		c.handleResponse(result)
	}
}

// handleNotification is responsible for handling notifications sent
// from clients.
func (c *Client) handleNotification(notif BaseRecv) {
	switch notif.Method {
	case "Player.OnResume":
		LogInfof("Resumed %s\n", c.Description())
	case "Player.OnPause":
		LogInfof("Paused %s\n", c.Description())
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
		LogInfo(c.Timestamp)
	}

	// Delete the operation since we got a response
	delete(c.ActiveOperations, response.ID)

	// Notify through channel
	c.OperationDone <- response.ID
}

// wsURI returns the websocket URI for a Client.
func (c Client) wsURI() url.URL {
	return url.URL{Scheme: "ws", Host: c.Host, Path: "/jsonrpc"}
}

// addAuthHeader adds the Authorization header using the client
// credentials to a http header.
func (c Client) addAuthHeader(h *http.Header) {
	bCreds := []byte(fmt.Sprintf("%s:%s", c.User, c.Password))
	sEnc := base64.StdEncoding.EncodeToString(bCreds)
	h.Add("Authorization", fmt.Sprintf("Basic %s", sEnc))
}
