package server

import (
	"fmt"
	"mediacenter/shared"
	"unsafe"
)

// MixInputs mixes a group of inputs to a single output stream
func MixInputs(ins [][]byte) []byte {
	// To mix inputs, we'll need to do some basic addition, so
	// we need to convert our bytes to their native float32 format
	floats := shared.Map(ins, func(in []byte) []float32 {
		return BytesToFloats(in)
	})

	if len(floats) == 0 {
		return nil
	}

	// normalize all slices to the same size
	longest := 0
	for _, in := range floats {
		if len(in) > longest {
			longest = len(in)
		}
	}
	zeroes := make([]float32, longest)
	for i, in := range floats {
		difference := longest - len(in)
		if difference > 0 {
			floats[i] = append(in, zeroes[:difference]...)
		}
	}

	// To mix audio, you just add/subtract them
	// To avoid clipping, we need to cap this between
	// -1.0 and 1.0
	mixedFloats := make([]float32, longest)
	for i := range longest {
		var total float32
		for _, in := range floats {
			total = shared.ClampFloat(total+in[i], -1, 1)
		}
		mixedFloats[i] = total
	}

	var inAvg float32
	for _, n := range floats[0] {
		inAvg += n
	}
	inAvg /= float32(len(floats[0]))
	var outAvg float32
	for _, n := range mixedFloats {
		outAvg += n
	}
	outAvg /= float32(len(mixedFloats))

	fmt.Printf("In average: %f, out average: %f\n", inAvg, outAvg)

	// Now convert the float back to bytes and we're golden
	return FloatsToBytes(mixedFloats)
}

// BytesToFloats converts a byte slice to a float32 slice without copying.
func BytesToFloats(b []byte) []float32 {
	if len(b) == 0 {
		return nil
	}
	// Divide length by 4 because float32 is 4 bytes
	return unsafe.Slice((*float32)(unsafe.Pointer(&b[0])), len(b)/4)
}

// FloatsToBytes converts a float32 slice to a byte slice without copying.
func FloatsToBytes(f []float32) []byte {
	if len(f) == 0 {
		return nil
	}
	// Multiply length by 4 because each float32 is 4 bytes
	return unsafe.Slice((*byte)(unsafe.Pointer(&f[0])), len(f)*4)
}
