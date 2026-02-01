package shared

import (
	"context"
	"fmt"
	"iter"
	"strconv"
	"strings"
	"unsafe"
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

// BytesToFloats takes a slice of raw bytes and converts
// them to a slice of float32
func BytesToFloats(b []byte) []float32 {
	if len(b) == 0 {
		return nil
	}

	return unsafe.Slice((*float32)(unsafe.Pointer(&b[0])), len(b)/4)
}

// FloatsToBytes takes a slice of floats and converts
// it to raw bytes
func FloatsToBytes(f []float32) []byte {
	if len(f) == 0 {
		return nil
	}

	return unsafe.Slice((*byte)(unsafe.Pointer(&f[0])), len(f)*4)
}

// CraftServerDiscoveryResponse creates the server positive response
// including the port number
func CraftServerDiscoveryResponse(port int) []byte {
	return []byte(fmt.Sprintf("%s:%d", ServerDiscoveryResponse, port))
}

// ReadServerDiscoveryResponse reads the response and returns a
// bool telling you if that is the correct message, and the port
// that is included in the message
func ReadServerDiscoveryResponse(resp string) (bool, int, error) {
	parts := strings.Split(resp, ServerDiscoveryDelimiter)
	if len(parts) != 2 {
		return false, 0, nil
	}

	if parts[0] != ServerDiscoveryResponse {
		return false, 0, nil
	}

	port, err := strconv.Atoi(parts[1])
	return true, port, err
}
