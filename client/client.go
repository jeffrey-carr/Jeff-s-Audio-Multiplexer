package client

import (
	"context"
	"errors"
	"fmt"
	"mediacenter/shared"
	"net"
	"strings"
	"time"

	"github.com/gen2brain/malgo"
	"go.uber.org/multierr"
)

// MediaClient is the media client
type MediaClient struct {
	serverPort int

	name         string
	capabilities []int
}

// NewMediaClient creates a new media client
func NewMediaClient(serverPort int, clientName string) *MediaClient {
	// TODO - capabilities
	return &MediaClient{
		serverPort: serverPort,
		name:       clientName,
	}
}

// Start starts listening for audio and sending it to the server
func (client *MediaClient) Start() (func() error, error) {
	rootCtx := context.Background()
	clientCtx, stopClient := context.WithCancel(rootCtx)
	// the context is really only used during boot, to be able
	// to return from server discovery early. So we can save ourselves
	// some stopClient() calls by just deferring it here
	defer stopClient()

	connection, err := client.startUDP(clientCtx)
	if err != nil {
		return nil, err
	}

	deviceCloser, err := shared.StartDevice("blackhole", malgo.Capture, func(_, pInput []byte, _ uint32) {
		// Typical MTU on networks is 1400 bytes, so we need to split our message into smaller packets
		for packet := range shared.StreamSlice(pInput, shared.NetworkPacketSizeBytes) {
			connection.Write(packet)
		}
	})
	if err != nil {
		return nil, err
	}

	closer := func() error {
		deviceErr := deviceCloser()
		connErr := connection.Close()
		return multierr.Append(deviceErr, connErr)
	}

	return closer, nil
}

func (client *MediaClient) discoverServer(ctx context.Context) (*net.UDPAddr, error) {
	// set up a listener for server responses
	listener, err := net.ListenPacket("udp", ":0")
	if err != nil {
		return nil, err
	}

	dst, err := net.ResolveUDPAddr("udp", fmt.Sprintf("255.255.255.255:%d", client.serverPort))
	if err != nil {
		return nil, err
	}

	attempts := 0
	for {
		if shared.ShouldKillCtx(ctx) {
			return nil, nil
		}

		attempts++
		fmt.Printf("Sending message to server: %s\n", shared.ServerDiscoveryKeyword)
		listener.WriteTo([]byte(shared.ServerDiscoveryKeyword), dst)
		listener.SetDeadline(time.Now().Add(ServerDiscoveryTimeout))
		// craft a sample response to determine the size of the buffer
		sampleResponse := shared.CraftServerDiscoveryResponse(9999)
		buffer := make([]byte, len(sampleResponse))
		var serverPort int

		for {
			if shared.ShouldKillCtx(ctx) {
				return nil, nil
			}

			_, peerAddr, err := listener.ReadFrom(buffer)
			if err != nil {
				break
			}

			bufferContents := strings.TrimRight(string(buffer), "\x00")
			shared.ZeroSlice(buffer)

			peerUDPAddr, ok := peerAddr.(*net.UDPAddr)
			if !ok {
				continue
			}

			fmt.Printf("received response from %s: %s\n", peerAddr.String(), bufferContents)
			actionType := shared.IdentifyServerAction(bufferContents)

			switch actionType {
			case shared.ServerActionDiscover:
				var ok bool
				ok, serverPort, err = client.handleDiscoveryResponse(bufferContents, peerUDPAddr, listener)
				if !ok || err != nil {
					continue
				}
			case shared.ServerActionIdentification:
				ok, err := client.handleIdentificationResponse(bufferContents)
				if err != nil {
					serverPort = 0
					break
				}
				if ok {
					peerUDPAddr.Port = serverPort
					return peerUDPAddr, nil
				}
			default:
				fmt.Println("unknown action")
			}
		}

		if attempts >= ServerDiscoveryAttempts {
			return nil, errors.New("could not find server")
		}
	}
}

func (client *MediaClient) handleDiscoveryResponse(message string, dst *net.UDPAddr, conn net.PacketConn) (bool, int, error) {
	isServer, serverPort, err := shared.ReadServerDiscoveryResponse(message)
	if err != nil {
		return false, 0, err
	}

	if !isServer {
		return false, 0, nil
	}

	_, err = conn.WriteTo(shared.CraftClientIdentificationMessage(client.name, []int{1}), dst)
	if err != nil {
		return false, 0, err
	}

	return true, serverPort, nil
}

func (client *MediaClient) handleIdentificationResponse(message string) (bool, error) {
	ok, err := shared.ReadClientIdentificationResponse(message)
	if err == shared.ErrNotClientIdentificationMessage {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !ok {
		return false, errors.New("connection error")
	}

	return true, nil
}

func (client *MediaClient) startUDP(ctx context.Context) (*net.UDPConn, error) {
	serverAddr, err := client.discoverServer(ctx)
	if err != nil {
		return nil, err
	}

	return net.DialUDP("udp", nil, serverAddr)
}
