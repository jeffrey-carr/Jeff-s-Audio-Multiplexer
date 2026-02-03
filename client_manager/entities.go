package clientmanager

import (
	"mediacenter/shared"
	"net"
	"time"
)

// Client is the information we have about a client
type Client struct {
	Name           string `json:"name"`
	IP             string
	Addr           *net.Addr
	Status         ClientStatus
	DataBuffer     shared.ThreadSafeBuffer[byte]
	Capabilities   []int
	LastSeen       time.Time  `json:"lastSeen"`
	DisconnectedAt *time.Time `json:"disconnectedAt"`
}

// ClientStatus is the possible statuses for a client
type ClientStatus string

const (
	// ClientStatusConnected is the status for when a client is connected
	ClientStatusConnected ClientStatus = "connected"
	// ClientStatusDisconnected is the status for when a client is disconnected
	ClientStatusDisconnected ClientStatus = "disconnected"
)

// ClientCapability is the capabilities a client has for audio
type ClientCapability int

const (
	// ClientCapabilityRecord signals that a client can record audio
	ClientCapabilityRecord ClientCapability = 1
	// ClientCapabilityPlayback signals that a client can play audio
	ClientCapabilityPlayback ClientCapability = 2
)
