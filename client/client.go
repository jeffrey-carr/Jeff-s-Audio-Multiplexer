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

	connection, sessionToken, err := client.startUDP(clientCtx)
	if err != nil {
		return nil, err
	}

	deviceCloser, err := shared.StartDevice("blackhole", malgo.Capture, func(_, pInput []byte, _ uint32) {
		for packet := range shared.StreamSlice(pInput, shared.NetworkPacketSizeBytes) {
			connection.Write(shared.CreateClientBytesRequest(sessionToken, packet))
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

func (client *MediaClient) discoverServer(ctx context.Context) (*net.UDPAddr, string, error) {
	// set up a listener for server responses
	listener, err := net.ListenPacket("udp", ":0")
	if err != nil {
		return nil, "", err
	}

	dst, err := net.ResolveUDPAddr("udp", fmt.Sprintf("255.255.255.255:%d", client.serverPort))
	if err != nil {
		return nil, "", err
	}

	attempts := 0
	for {
		if shared.ShouldKillCtx(ctx) {
			return nil, "", nil
		}

		attempts++
		fmt.Printf("Sending message to server: %s\n", shared.ServerDiscoveryKeyword)
		listener.WriteTo([]byte(shared.ServerDiscoveryKeyword), dst)
		listener.SetDeadline(time.Now().Add(ServerDiscoveryTimeout))
		buffer := make([]byte, 1024)
		var serverPort int

		for {
			if shared.ShouldKillCtx(ctx) {
				return nil, "", nil
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
				ok, sessionToken, err := client.handleIdentificationResponse(bufferContents)
				if err != nil {
					serverPort = 0
					break
				}
				if ok {
					peerUDPAddr.Port = serverPort
					return peerUDPAddr, sessionToken, nil
				}
			default:
				fmt.Println("unknown action")
			}
		}

		if attempts >= ServerDiscoveryAttempts {
			return nil, "", errors.New("could not find server")
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

func (client *MediaClient) handleIdentificationResponse(message string) (bool, string, error) {
	ok, sessionToken, err := shared.ReadClientIdentificationResponse(message)
	if err == shared.ErrNotClientIdentificationMessage {
		return false, "", nil
	}
	if err != nil {
		return false, "", err
	}
	if !ok {
		return false, "", errors.New("connection error")
	}

	return true, sessionToken, nil
}

func (client *MediaClient) startUDP(ctx context.Context) (*net.UDPConn, string, error) {
	serverAddr, sessionToken, err := client.discoverServer(ctx)
	if err != nil {
		return nil, "", err
	}

	conn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		return nil, "", err
	}

	return conn, sessionToken, nil
}
