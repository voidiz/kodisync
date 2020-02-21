package main

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Enum (starting at 1 to avoid zero value for int, 0)
// for client operations.
const (
	PlayerGetPropertiesTime = iota + 1
)

// BaseRecv is the base message received from Kodi.
type BaseRecv struct {
	JSONRPC string           `json:"jsonrpc"`
	Method  string           `json:"method"`
	Result  *json.RawMessage `json:"result"`
	Error   *json.RawMessage `json:"error"`
	ID      int              `json:"id"`
}

// BaseSend is the base message sent to Kodi.
type BaseSend struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
	ID      int         `json:"id,omitempty"`
}

// PlayerTime is the time property returned from
// Player.GetProperties.
type PlayerTime struct {
	Hours   int `json:"hours"`
	Minutes int `json:"minutes"`
	Seconds int `json:"seconds"`
}

// PlayerProperties contains properties returned by
// Player.GetProperties in the "result" key.
type PlayerProperties struct {
	Time *json.RawMessage `json:"time"`
}

// NewBaseSend creates a message containing a method and encoded
// params.
func NewBaseSend(method string, params interface{}) BaseSend {
	randID, _ := uuid.New().Time().UnixTime()
	return BaseSend{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      int(randID),
	}
}

// ToString prints a string representation of BaseRecv.
func (br *BaseRecv) ToString() string {
	return toString(br)
}

// ToString prints a string representation of BaseSend.
func (bs *BaseSend) ToString() string {
	return toString(bs)
}

// ToDuration creates a time.Duration from a PlayerTime.
func (pt *PlayerTime) ToDuration() time.Duration {
	hours := time.Duration(pt.Hours) * time.Hour
	minutes := time.Duration(pt.Minutes) * time.Minute
	seconds := time.Duration(pt.Seconds) * time.Second

	return hours + minutes + seconds
}

func toString(v interface{}) string {
	res, err := json.Marshal(&v)
	if err != nil {
		LogWarn(err)
		return ""
	}

	return string(res)
}
