package server

import (
	"context"
	"errors"
	"fmt"
	clientmanager "mediacenter/client_manager"
	"mediacenter/shared"
	"net"
	"time"

	"github.com/gen2brain/malgo"
)

// MediaServer is the server for the media center
type MediaServer struct {
	serverPort    int
	discoveryPort int
	clients       clientmanager.ClientManager
	listener      *ListenerServer

	isRunning bool
}

// NewMediaServer creates a new MediaServer
func NewMediaServer(serverPort int, discoveryPort int, clientManager clientmanager.ClientManager) *MediaServer {
	listenerServer := NewListenerServer(discoveryPort, serverPort, clientManager)
	return &MediaServer{
		serverPort:    serverPort,
		discoveryPort: discoveryPort,
		clients:       clientManager,
		listener:      listenerServer,
	}
}

// Start starts the server
func (s *MediaServer) Start() (func() error, error) {
	bgCtx := context.Background()
	serverCtx, stopServer := context.WithCancel(bgCtx)

	deviceCloser, err := shared.StartDevice(
		"", // Not passing in a device name plays out of the default device
		malgo.Playback,
		s.handleAudio(),
	)
	if err != nil {
		stopServer()
		return nil, err
	}
	err = s.launchServer(serverCtx)
	if err != nil {
		stopServer()
		return nil, err
	}
	err = s.listener.Start(serverCtx)
	if err != nil {
		stopServer()
		return nil, err
	}

	closer := func() error {
		if stopServer != nil {
			stopServer()
			return nil
		}

		var err error
		if deviceCloser != nil {
			err = deviceCloser()
		}

		fmt.Println("Stopped server.")
		return err
	}

	return closer, nil
}

func (s *MediaServer) launchServer(ctx context.Context) error {
	if s.isRunning {
		return errors.New("server is already running")
	}
	s.isRunning = true

	server, err := s.startUDP()
	if err != nil {
		return err
	}

	go func() {
		buffer := make([]byte, shared.NetworkPacketSizeBytes+shared.ClientAudioBytesHeaderLen)
		for {
			if shared.ShouldKillCtx(ctx) {
				return
			}

			bytesReceived, _, err := server.ReadFromUDP(buffer)
			if err != nil {
				fmt.Printf("Error reading: %s\n", err.Error())
				continue
			}

			header := buffer[:shared.ClientAudioBytesHeaderLen]

			ok, sessionToken, err := shared.ReadClientBytesRequest(string(header))
			if err != nil {
				fmt.Printf("error reading message: %s\n", err.Error())
				continue
			}
			if !ok {
				fmt.Println("header not ok")
				continue
			}

			client, found := s.clients.GetClientBySessionToken(sessionToken)
			if !found {
				fmt.Printf("could not find client with session token %s\n", sessionToken)
				continue
			}
			client.LastSeen = time.Now()
			client.DataBuffer.Add(buffer[shared.ClientAudioBytesHeaderLen:bytesReceived]...)
			s.clients.SetClient(client)
		}
	}()

	fmt.Println("Started server")
	return nil
}

func (s *MediaServer) startUDP() (*net.UDPConn, error) {
	addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf(":%d", s.serverPort))
	if err != nil {
		return nil, err
	}

	return net.ListenUDP("udp4", addr)
}

func (s *MediaServer) handleAudio() shared.MalgoCallback {
	return func(pOutput, _ []byte, _ uint32) {
		bytesNeeded := len(pOutput)
		connectedClients := s.clients.ConnectedClients()

		// we need to wait for at least 1 client to be ready to send
		// audio
		if len(connectedClients) == 0 {
			shared.ZeroSlice(pOutput)
			return
		}

		readyClients := shared.FilterSlice(connectedClients, func(client clientmanager.Client) bool {
			return client.DataBuffer.Size() >= shared.NetworkPacketSizeBytes*3
		})
		if len(readyClients) < 1 {
			shared.ZeroSlice(pOutput)
			return
		}

		audioBuffers := make([][]byte, len(readyClients))
		for i, client := range readyClients {
			audioBuffers[i] = client.DataBuffer.Read(bytesNeeded)
		}

		mixed := MixInputs(audioBuffers)
		copy(pOutput, mixed)
	}
}
