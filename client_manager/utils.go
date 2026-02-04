package clientmanager

import (
	"mediacenter/shared"
	"net"
	"time"

	"github.com/google/uuid"
)

// NewClient creates a new client
func NewClient(name string, addr net.Addr, capabilities []int, sessionToken string) Client {
	return Client{
		Name:         name,
		SessionToken: sessionToken,
		Addr:         &addr,
		Capabilities: capabilities,
		DataBuffer:   shared.NewThreadSafeBuffer[byte](48000),
		Status:       ClientStatusConnected,
		LastSeen:     time.Now(),
	}
}

// GenerateUUID generates a new UUID
func GenerateUUID() string {
	return uuid.NewString()
}
