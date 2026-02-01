package client

import "time"

const (
	// ServerDiscoveryTimeout is the amount of time we wait
	// for a server response before giving up
	ServerDiscoveryTimeout = time.Second * 2
	// ServerDiscoveryAttempts is the number of time we'll try
	// contacting the server before giving up
	ServerDiscoveryAttempts = 3
)
