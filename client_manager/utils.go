package clientmanager

import (
	"mediacenter/shared"
	"net"
	"time"
)

// NewClient creates a new client
func NewClient(name string, addr net.Addr, capabilities []int) Client {
	return Client{
		Name:         name,
		Addr:         &addr,
		Capabilities: capabilities,
		DataBuffer:   shared.NewThreadSafeBuffer[byte](48000),
		Status:       ClientStatusConnected,
		LastSeen:     time.Now(),
	}
}
