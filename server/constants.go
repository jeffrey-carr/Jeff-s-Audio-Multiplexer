package server

import "mediacenter/shared"

const (
	// AudioBufferSize is the size of the audio buffer
	AudioBufferSize = 48000
	// AudioBufferThreshold is the threshold of content
	// the audio buffer must reach to start playing audio
	AudioBufferThreshold = shared.NumOutputChannels * shared.AudioSampleRate / 4
)
