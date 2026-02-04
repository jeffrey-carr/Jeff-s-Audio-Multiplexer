package server

import (
	"context"
	"fmt"
	clientmanager "mediacenter/client_manager"
	"mediacenter/shared"
	"net"
	"strings"
)

// ListenerServer listens for new connections
type ListenerServer struct {
	port            int
	mainServicePort int

	clients clientmanager.ClientManager
}

// NewListenerServer starts a new listener server
func NewListenerServer(port int, mainServicePort int, clientManager clientmanager.ClientManager) *ListenerServer {
	return &ListenerServer{
		port:            port,
		mainServicePort: mainServicePort,
		clients:         clientManager,
	}
}

// Start starts the listener server
func (server *ListenerServer) Start(ctx context.Context) error {
	errChan := make(chan error, 1)
	go func() {
		defer close(errChan)

		listener, err := net.ListenPacket("udp4", fmt.Sprintf(":%d", server.port))
		if err != nil {
			errChan <- err
			return
		}
		defer listener.Close()

		// we've done all the scary work with starting the
		// listener server, so we can stop worrying
		// about errors (for now, we should log errors at some point)
		close(errChan)

		buffer := make([]byte, 2048)
		for {
			if shared.ShouldKillCtx(ctx) {
				return
			}

			_, clientAddr, err := listener.ReadFrom(buffer)
			if err != nil {
				continue
			}

			bufferContent := strings.TrimRight(string(buffer), "\x00")
			shared.ZeroSlice(buffer)

			fmt.Printf("Received message from %s: %s\n", clientAddr.String(), bufferContent)

			actionRequest := shared.IdentifyServerAction(bufferContent)
			switch actionRequest {
			case shared.ServerActionDiscover:
				err = server.handleDiscoveryRequest(bufferContent, listener, clientAddr)
			case shared.ServerActionIdentification:
				err = server.handleClientIdentificationRequest(bufferContent, listener, clientAddr)
			default:
				continue
			}
			if err != nil {
				fmt.Printf("Error communicating with client: %s\n", err.Error())
			}
		}
	}()

	for err := range errChan {
		if err != nil {
			return err
		}
	}

	fmt.Println("Started listener server")
	return nil
}

// handleDiscoverRequest takes in discovery requests and returns the port of the main server to the client.
// Since we don't know if this client will ever be anything, we don't need to store anything from this interaction
// until the client introduces themselves
func (server *ListenerServer) handleDiscoveryRequest(message string, conn net.PacketConn, dst net.Addr) error {
	if message != shared.ServerDiscoveryKeyword {
		return nil
	}

	_, err := conn.WriteTo(shared.CraftServerDiscoveryResponse(server.mainServicePort), dst)
	return err
}

// handleClientIdentificationRequest handles incoming client identification requests
func (server *ListenerServer) handleClientIdentificationRequest(message string, conn net.PacketConn, dst net.Addr) error {
	isIdentificationMessage, name, capabilities, err := shared.ReadClientIdentificationMessage(message)
	if err != nil {
		conn.WriteTo(shared.CraftClientIdentificationResponse(false, ""), dst)
		return err
	}
	if !isIdentificationMessage {
		return nil
	}

	client, err := server.clients.AddClient(name, dst, capabilities)
	if err != nil {
		conn.WriteTo(shared.CraftClientIdentificationResponse(false, ""), dst)
		return err
	}

	server.clients.PrintStatuses()

	_, err = conn.WriteTo(shared.CraftClientIdentificationResponse(true, client.SessionToken), dst)
	return err
}
