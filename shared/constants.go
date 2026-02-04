package shared

import "errors"

// Networking constants
const (
	// NetworkPacketSizeBytes is the maximum size of
	// a UDP network packet
	NetworkPacketSizeBytes = 1400
	// ServerMessagePartsDelimiter is the delimiter between
	// different parts of a message on the server
	ServerMessagePartsDelimiter = ";"
	// ServerMessageItemDelimiter is the delimiter used within
	// individual parts to further break the part down
	ServerMessageItemDelimiter = ":"
	// ServerDiscoveryKeyword is the phrase used to distinguish
	// server discovery messages on the network
	ServerDiscoveryKeyword = "WHO_IS_MEDIA_SERVER"
	// ServerDiscoverResponse is the phrase used to distinguish
	// server discovery response messages on the network
	ServerDiscoveryResponse = "I_AM_MEDIA_SERVER"
	// ClientIdentificationKeyword is the phrase used to distinguish
	// client identification messages on the network
	ClientIdentificationKeyword = "I_AM_CLIENT"
	// ClientIdentificationResponse is the phrase used to distinguish
	// client identification response from the server
	ClientIdentificationResponse = "HI_CLIENT"
	// ClientIdentificationNameKey is the key for the 'name' item
	// within a client identification message
	ClientIdentificationNameKey = "NAME"
	// ClientIdentificationCapabilitiesKey is the key for the
	// 'capabilities' item within a client identification message
	ClientIdentificationCapabilitiesKey = "CAPABILITIES"
	// ClientAudioBytes is the audio bytes header
	ClientAudioBytes = "AUDIO"

	// ClientAudioBytesHeaderLen is the amount of bytes the audio client bytes header is
	// AUDIO (5) + ; (1) + UUID (36) + ; (1) = 43
	ClientAudioBytesHeaderLen = 43
)

var (
	// ErrNotClientIdentificationMessage is returned when you know
	ErrNotClientIdentificationMessage = errors.New("not identification message")
)

// ServerAction is the type of actions a client/server can take
type ServerAction string

const (
	// ServerActionUnknown is an unknown server action
	ServerActionUnknown ServerAction = "UNKNOWN"
	// ServerActionDiscover is the server action for discovering each other
	ServerActionDiscover ServerAction = "DISCOVER"
	// ServerActionIdentification is the server action for identifying a client
	ServerActionIdentification ServerAction = "IDENTIFICATION"
	// ServerActionExchangeAudio when audio is being exchanged
	ServerActionExchangeAudio ServerAction = "EXCHANGE_AUDIO"
)

// Audio device constants
const (
	NumInputChannels         = 2
	NumOutputChannels        = 2
	AudioSampleRate          = 48000
	SamplePeriodMilliseconds = 10
)

// MalgoCallback is the callback that gets passed to malgo
// to handle device data
type MalgoCallback = func(output []byte, input []byte, size uint32)
