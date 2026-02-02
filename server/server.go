package server

import (
	"context"
	"errors"
	"fmt"
	clientmanager "mediacenter/client_manager"
	"mediacenter/shared"
	"net"

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

	// audioBuffer is for holding the audio bytes for the speaker output
	audioBuffer := shared.NewThreadSafeBuffer[byte](AudioBufferSize)

	deviceCloser, err := shared.StartDevice(
		"", // Not passing in a device name plays out of the default device
		malgo.Playback,
		s.handleAudio(audioBuffer),
	)
	if err != nil {
		stopServer()
		return nil, err
	}
	err = s.launchServer(serverCtx, audioBuffer)
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

func (s *MediaServer) launchServer(ctx context.Context, audioBuffer shared.ThreadSafeBuffer[byte]) error {
	if s.isRunning {
		return errors.New("server is already running")
	}
	s.isRunning = true

	server, err := s.startUDP()
	if err != nil {
		return err
	}

	go func() {
		buffer := make([]byte, shared.NetworkPacketSizeBytes)
		for {
			if shared.ShouldKillCtx(ctx) {
				return
			}

			n, addr, err := server.ReadFromUDP(buffer)
			if err != nil {
				fmt.Printf("Error reading: %s\n", err.Error())
				continue
			}
			if addr != nil {
				s.clients.MarkClient(addr.IP.String())
			}

			audioBuffer.Add(buffer[:n]...)
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

func (s *MediaServer) handleAudio(audioBuffer shared.ThreadSafeBuffer[byte]) shared.MalgoCallback {
	isBuffering := true

	return func(pOutput, _ []byte, _ uint32) {
		currentSize := audioBuffer.Size()

		// try to keep buffer at least 3 packets to avoid crackling
		// 1 for playing, 1 for the buffer, and 1 "in-flight"
		// in-flight being part of the buffer to protect against dropped packets
		// or wifi latency
		if currentSize < shared.NetworkPacketSizeBytes*3 {
			isBuffering = true
		}

		if isBuffering {
			if currentSize < AudioBufferThreshold {
				// Fill the output slice with zeroes to prevent
				// garbage output from leftover data
				shared.ZeroSlice(pOutput)
				return
			}
			isBuffering = false
		}

		audioBuffer.ReadInto(pOutput)
	}
}
