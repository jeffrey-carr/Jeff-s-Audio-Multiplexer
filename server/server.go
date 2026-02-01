package server

import (
	"context"
	"errors"
	"fmt"
	"mediacenter/shared"
	"net"
	"slices"

	"github.com/gen2brain/malgo"
)

// MediaServer is the server for the media center
type MediaServer struct {
	serverPort    int
	discoveryPort int

	isRunning bool
}

// NewMediaServer creates a new MediaServer
func NewMediaServer(serverPort int, discoveryPort int) *MediaServer {
	return &MediaServer{
		serverPort:    serverPort,
		discoveryPort: discoveryPort,
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
	err = s.listenBroadcast(serverCtx)
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

func (s *MediaServer) launchServer(ctx context.Context, audioBuffer *shared.ThreadSafeBuffer[byte]) error {
	if s.isRunning {
		return errors.New("server is already running")
	}
	s.isRunning = true

	server, err := s.startUDP()
	if err != nil {
		return err
	}

	buffer := make([]byte, shared.NetworkPacketSizeBytes)
	go func() {
		for {
			if shared.ShouldKillCtx(ctx) {
				return
			}

			n, _, err := server.ReadFromUDP(buffer)
			if err != nil {
				fmt.Printf("Error reading: %s\n", err.Error())
				continue
			}

			audioBuffer.Add(buffer[:n]...)
		}
	}()

	fmt.Println("Started server")
	return nil
}

func (s *MediaServer) startUDP() (*net.UDPConn, error) {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", s.serverPort))
	if err != nil {
		return nil, err
	}

	return net.ListenUDP("udp", addr)
}

func (s *MediaServer) listenBroadcast(ctx context.Context) error {
	errChan := make(chan error, 1)
	go func() {
		defer close(errChan)

		listener, err := net.ListenPacket("udp", fmt.Sprintf(":%d", s.discoveryPort))
		if err != nil {
			errChan <- err
			return
		}
		defer listener.Close()

		// we've done all the scary work with starting the
		// listener server, so we can stop worrying
		// about errors (for now, we should log errors at some point)
		close(errChan)

		discoverResponse := shared.CraftServerDiscoveryResponse(s.serverPort)
		discoverMessage := []byte(shared.ServerDiscoveryKeyword)
		buffer := make([]byte, len(discoverMessage))
		for {
			if shared.ShouldKillCtx(ctx) {
				return
			}

			_, clientAddr, err := listener.ReadFrom(buffer)
			if err != nil {
				continue
			}

			fmt.Printf("Received message from %s: %s\n", clientAddr.String(), string(buffer))

			if slices.Equal(discoverMessage, buffer) {
				listener.WriteTo(discoverResponse, clientAddr)
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

func (s *MediaServer) handleAudio(audioBuffer *shared.ThreadSafeBuffer[byte]) shared.MalgoCallback {
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
