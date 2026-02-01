package shared

import (
	"errors"
	"strings"

	"github.com/gen2brain/malgo"
)

// MalgoConfig creates the Malgo configuration
func MalgoConfig(deviceID *malgo.DeviceID, deviceType malgo.DeviceType) malgo.DeviceConfig {
	deviceConfig := malgo.DefaultDeviceConfig(malgo.Duplex)
	deviceConfig.Capture.Format = malgo.FormatF32
	deviceConfig.Capture.Channels = NumInputChannels
	deviceConfig.Playback.Format = malgo.FormatF32
	deviceConfig.Playback.Channels = NumOutputChannels
	deviceConfig.SampleRate = AudioSampleRate
	deviceConfig.Alsa.NoMMap = 1
	deviceConfig.PeriodSizeInMilliseconds = SamplePeriodMilliseconds

	if deviceID != nil {
		switch deviceType {
		case malgo.Capture:
			deviceConfig.Capture.DeviceID = deviceID.Pointer()
		case malgo.Playback:
			deviceConfig.Playback.DeviceID = deviceID.Pointer()
		}
	}

	return deviceConfig
}

// FindDeviceID finds the ID of the specified device, or not
func FindDeviceID(
	ctx *malgo.AllocatedContext,
	deviceType malgo.DeviceType,
	nameSnippet string,
) (malgo.DeviceID, error) {
	infos, err := ctx.Devices(deviceType)
	if err != nil {
		return malgo.DeviceID{}, err
	}

	for _, info := range infos {
		deviceName := strings.ToLower(strings.TrimRight(info.Name(), "\x00"))
		if strings.Contains(deviceName, strings.ToLower(nameSnippet)) {
			return info.ID, nil
		}
	}

	return malgo.DeviceID{}, errors.New("device not found")
}

// BuildDeviceKiller builds the closer for the client
func BuildDeviceKiller(ctx *malgo.AllocatedContext, device *malgo.Device) func() error {
	return func() error {
		if device != nil {
			device.Uninit()
		}

		if ctx != nil {
			err := ctx.Uninit()
			if err != nil {
				return err
			}
			ctx.Free()
		}

		return nil
	}
}

// StartDevice starts an audio device
func StartDevice(
	deviceName string,
	deviceType malgo.DeviceType,
	callback func([]byte, []byte, uint32),
) (func() error, error) {
	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		return nil, err
	}

	var deviceIDPtr *malgo.DeviceID
	if deviceName != "" {
		deviceID, err := FindDeviceID(ctx, deviceType, deviceName)
		if err != nil {
			return nil, err
		}
		deviceIDPtr = &deviceID
	}

	deviceConfig := MalgoConfig(deviceIDPtr, deviceType)
	deviceCallbacks := malgo.DeviceCallbacks{
		Data: callback,
	}

	device, err := malgo.InitDevice(ctx.Context, deviceConfig, deviceCallbacks)
	if err != nil {
		return nil, err
	}

	err = device.Start()
	if err != nil {
		return nil, err
	}

	return BuildDeviceKiller(ctx, device), nil
}
