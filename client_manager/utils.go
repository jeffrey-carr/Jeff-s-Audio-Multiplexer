package clientmanager

import (
	"net"
	"strings"
	"time"
)

// NewClient creates a new client
func NewClient(name string, addr net.Addr, capabilities []int) Client {
	addrParts := strings.Split(addr.String(), ":")
	// rejoin by : in case it was ipv6
	ip := strings.Join(addrParts[:len(addrParts)-1], ":")
	return Client{
		Name:         name,
		IP:           ip,
		Addr:         &addr,
		Capabilities: capabilities,
		Status:       ClientStatusConnected,
		LastSeen:     time.Now(),
	}
}
