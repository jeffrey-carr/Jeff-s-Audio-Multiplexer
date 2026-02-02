package clientmanager

import (
	"context"
	"fmt"
	"maps"
	"mediacenter/shared"
	"net"
	"slices"
	"time"
)

// ClientManager managers client connections
type ClientManager interface {
	AddClient(
		name string,
		clientAddr net.Addr,
		capabilities []int,
	) error
	// MarkClient marks a client as seen
	MarkClient(ip string)
	PrintStatuses()
}

type clientManager struct {
	// clients is a map of client name to data
	clients    shared.ThreadSafeMap[string, Client]
	cleanIters int
}

// NewClientManager creates a new client manager
func NewClientManager(ctx context.Context) ClientManager {
	manager := &clientManager{
		clients: shared.NewThreadSafeMap[string, Client](MaxConnections),
	}
	manager.startCleaner(ctx)
	return manager
}

func (cm *clientManager) AddClient(
	name string,
	clientAddr net.Addr,
	capabilities []int,
) error {
	// For right now, there isn't really any data that we need to carry
	// over between client connections, so there isn't any reason to check
	// if the client already exists. Even if we have an already-connected
	// client, it's more likely the client lost connection and is re-joining.
	// So we just create a whole new client every time and save it
	client := NewClient(name, clientAddr, capabilities)

	err := cm.clients.Set(name, client)
	if err == shared.ErrMapFull {
		// If our connection map is full, try forcing cleaning out any
		// disconnected clients
		cm.cleanConnections(true)
		err = cm.clients.Set(name, client)
	}

	return err
}

func (cm *clientManager) MarkClient(ip string) {
	go func() {
		snap := cm.clients.Snapshot()
		var client Client
		var found bool
		for _, potentialClient := range snap {
			if potentialClient.IP == ip {
				client = potentialClient
				found = true
				break
			}
		}

		if !found {
			return
		}

		client.LastSeen = time.Now()
		cm.clients.Set(client.Name, client)
	}()
}

func (cm *clientManager) Message(name, msg string) error {
	// TODO
	return nil
}

func (cm *clientManager) PrintStatuses() {
	clients := cm.clients.Snapshot()

	fmt.Println("Client name - Client status - Last seen")
	for _, client := range clients {
		fmt.Printf("\t%s - %s - %s\n", client.Name, client.Status, client.LastSeen.String())
	}
}

func (cm *clientManager) startCleaner(ctx context.Context) {
	go func() {
		for {
			if shared.ShouldKillCtx(ctx) {
				return
			}

			cm.cleanIters++
			cm.cleanConnections(false)
			if cm.cleanIters == 15 {
				cm.PrintStatuses()
				cm.cleanIters = 0
			}
			time.Sleep(time.Second)
		}
	}()
}

func (cm *clientManager) cleanConnections(forceClean bool) {
	allClients := cm.clients.Snapshot()
	if len(allClients) == 0 {
		return
	}

	clients := slices.Collect(maps.Values(allClients))
	for _, client := range clients {
		if client.Status == ClientStatusConnected && time.Since(client.LastSeen) > ConnectionTimeout {
			now := time.Now()
			client.DisconnectedAt = &now
			client.Status = ClientStatusDisconnected
			cm.clients.Set(client.Name, client)
		}
		if forceClean && client.Status == ClientStatusDisconnected {
			cm.clients.Remove(client.Name)
		}
	}
}
