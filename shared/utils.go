package shared

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"strconv"
	"strings"
)

// ShouldKillCtx easily tells you if your context has been canceled
func ShouldKillCtx(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

// Map runs a function over a slice and returns the output
func Map[T any, K any](s []T, f func(T) K) []K {
	out := make([]K, len(s))
	for i, item := range s {
		out[i] = f(item)
	}
	return out
}

// StreamSlice allows streaming a slice in certain lengths
func StreamSlice[T any](s []T, length int) iter.Seq[[]T] {
	return func(yield func([]T) bool) {
		if length <= 0 {
			yield(s)
			return
		}

		head := 0
		for head < len(s) {
			tail := min(head+length, len(s))
			if !yield(s[head:tail]) {
				return
			}
			head = tail
		}
	}
}

// ZeroSlice fills a slice with some capacity to zero values
func ZeroSlice[T any](s []T) {
	var zero T
	for i := range s {
		s[i] = zero
	}
}

// IdentifyServerAction identifies the server action from the message
func IdentifyServerAction(message string) ServerAction {
	actionStr := strings.Split(message, ServerMessagePartsDelimiter)[0]
	switch actionStr {
	case ServerDiscoveryKeyword:
		fallthrough
	case ServerDiscoveryResponse:
		return ServerActionDiscover
	case ClientIdentificationKeyword:
		fallthrough
	case ClientIdentificationResponse:
		return ServerActionIdentification
	default:
		return ServerActionUnknown
	}
}

// CraftServerDiscoveryResponse creates the server positive response
// including the port number
func CraftServerDiscoveryResponse(port int) []byte {
	return []byte(joinParts(ServerDiscoveryResponse, strconv.FormatInt(int64(port), 10)))
}

// ReadServerDiscoveryResponse reads the response and returns a
// bool telling you if that is the correct message, and the port
// that is included in the message
func ReadServerDiscoveryResponse(resp string) (bool, int, error) {
	parts := strings.Split(resp, ServerMessagePartsDelimiter)
	if len(parts) != 2 {
		return false, 0, nil
	}

	if parts[0] != ServerDiscoveryResponse {
		return false, 0, nil
	}

	port, err := strconv.Atoi(parts[1])
	return true, port, err
}

// CraftClientIdentificationMessage puts together a client identification message
func CraftClientIdentificationMessage(name string, capabilities []int) []byte {
	capabilitiesStr := strings.Join(Map(capabilities, func(capability int) string {
		return strconv.FormatInt(int64(capability), 10)
	}), ",")

	// Non-AI proof. Only a human could make code so disgusting
	return []byte(joinParts(
		ClientIdentificationKeyword,
		joinItems(ClientIdentificationNameKey, name),
		joinItems(ClientIdentificationCapabilitiesKey, capabilitiesStr),
	))
}

// CraftClientIdentificationResponse puts together a client identification response message
func CraftClientIdentificationResponse(ok bool, sessionToken string) []byte {
	return []byte(
		joinParts(
			ClientIdentificationResponse,
			fmt.Sprintf("%t", ok),
			sessionToken,
		),
	)
}

// ReadClientIdentificationMessage reads a client identification message and
// returns the individual parts, and a boolean flag if this was indeed a client identification message
func ReadClientIdentificationMessage(message string) (bool, string, []int, error) {
	parts := strings.Split(message, ServerMessagePartsDelimiter)
	if len(parts) != 3 {
		return false, "", nil, nil
	}

	if parts[0] != ClientIdentificationKeyword {
		return false, "", nil, nil
	}

	nameIdentificationParts := strings.Split(parts[1], ServerMessageItemDelimiter)
	if len(nameIdentificationParts) != 2 ||
		nameIdentificationParts[0] != ClientIdentificationNameKey {
		return true, "", nil, errors.New("client name not provided")
	}
	name := nameIdentificationParts[1]

	capabilitiesParts := strings.Split(parts[2], ServerMessageItemDelimiter)
	if len(capabilitiesParts) != 2 ||
		capabilitiesParts[0] != ClientIdentificationCapabilitiesKey {
		return true, "", nil, errors.New("client capabilities not provided")
	}
	capabilities := Map(strings.Split(capabilitiesParts[1], ","), func(capabilityStr string) int {
		// ehhh it'll be easy to figure out if the capability isn't being sent correctly
		i, _ := strconv.Atoi(capabilityStr)
		return i
	})
	capabilities = FilterSlice(capabilities, func(capability int) bool { return capability > 0 })

	return true, name, capabilities, nil
}

// ReadClientIdentificationResponse returns whether the connection was ok
// based on the server response
func ReadClientIdentificationResponse(message string) (bool, string, error) {
	parts := strings.Split(message, ServerMessagePartsDelimiter)
	if len(parts) != 3 || parts[0] != ClientIdentificationResponse {
		return false, "", ErrNotClientIdentificationMessage
	}

	ok, err := strconv.ParseBool(parts[1])
	if err != nil {
		return false, "", err
	}

	return ok, parts[2], nil
}

// CreateClientBytesRequest creates a client bytes request
func CreateClientBytesRequest(sessionToken string, audio []byte) []byte {
	header := []byte(joinParts(ClientAudioBytes, sessionToken) + ";")
	return append(header, audio...)
}

// ReadClientBytesRequest reads in a clients bytes request
func ReadClientBytesRequest(message string) (bool, string, error) {
	parts := strings.Split(message, ServerMessagePartsDelimiter)
	if len(parts) != 3 || parts[0] != ClientAudioBytes {
		fmt.Printf("%s not okay, len %d and first thing is %s\n", message, len(parts), parts[0])
		return false, "", nil
	}

	return true, parts[1], nil
}

func joinParts(parts ...string) string {
	return strings.Join(parts, ServerMessagePartsDelimiter)
}

func joinItems(items ...string) string {
	return strings.Join(items, ServerMessageItemDelimiter)
}

// FilterSlice filters items out of a slice
func FilterSlice[T any](s []T, f func(T) bool) []T {
	var out []T
	for _, item := range s {
		if f(item) {
			out = append(out, item)
		}
	}

	return out
}

// CountInSlice counts the number of items in s that meet the criteria
// of the provided function
func CountInSlice[T any](s []T, shouldCount func(T) bool) int {
	validItems := FilterSlice(s, shouldCount)
	return len(validItems)
}

// ClampFloat clamps a float between the boundaries. If only one number
// is provided, it is assumed to be the minimum. If two numbers are provided,
// it is assumed to be [min, max). Any boundaries above 3 will be ignored.
func ClampFloat(in float32, boundaries ...float32) float32 {
	if len(boundaries) == 0 {
		return in
	}
	if len(boundaries) >= 1 {
		in = max(boundaries[0], in)
	}
	if len(boundaries) >= 2 {
		in = min(boundaries[1], in)
	}

	return in
}
