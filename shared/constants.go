package shared

// Networking constants
const (
	// NetworkPacketSizeBytes is the maximum size of
	// a UDP network packet
	NetworkPacketSizeBytes = 1400
	// ServerDiscoveryKeyword is the phrase used
	// to distinguish server discovery messages on
	// the network
	ServerDiscoveryKeyword   = "WHO_IS_MEDIA_SERVER"
	ServerDiscoveryResponse  = "I_AM_MEDIA_SERVER"
	ServerDiscoveryDelimiter = ":"
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
