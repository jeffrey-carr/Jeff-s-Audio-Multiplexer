package clientmanager

import (
	"errors"
	"time"
)

const (
	// MaxConnections is the maximum number of connections
	// allowed on the server
	MaxConnections = 150
	// ConnectionTimeout is length of time with no message
	// until a client is considered "disconnected"
	ConnectionTimeout = time.Duration(30) * time.Second
)

var (
	// ErrMaxConnections is thrown when the maximum number of clients are
	// already connected to the server
	ErrMaxConnections = errors.New("maximum number of connections reached")
)
