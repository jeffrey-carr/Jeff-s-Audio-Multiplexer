package shared

import (
	"sync"
)

// We operate on 2 channels
// 4 bytes (float32) * 2 channels = 8 bytes
const frameSize = 8

// ThreadSafeBuffer is a buffer that is thread safe.
type ThreadSafeBuffer[T any] interface {
	// Add adds data to the buffer. If the buffer is full,
	// it overwrites the oldest data
	Add(newData ...T) error
	// Read reads a certain amount of data out of the buffer
	Read(amount int) []T
	// ReadInto reads the data into a slice
	ReadInto(target []T)
	// Size returns the current size of the buffer
	Size() int
}

type threadSafeBuffer[T any] struct {
	data       []T
	size       int
	head, tail int
	mutex      sync.Mutex
}

// NewThreadSafeBuffer creates a new threadsafe buffer
func NewThreadSafeBuffer[T any](maxSize int) ThreadSafeBuffer[T] {
	data := make([]T, maxSize)
	var zero T
	for i := range maxSize {
		data[i] = zero
	}
	return &threadSafeBuffer[T]{
		data:  data,
		size:  0,
		head:  0,
		tail:  0,
		mutex: sync.Mutex{},
	}
}

func (buf *threadSafeBuffer[T]) Add(newData ...T) error {
	buf.mutex.Lock()
	defer buf.mutex.Unlock()

	count := len(newData)
	freeSpace := cap(buf.data) - buf.size
	// If we need to overwrite old data to make room
	if count > freeSpace {
		needed := count - freeSpace
		overwriteCount := (needed + frameSize - 1) / frameSize * frameSize

		buf.head = (buf.head + overwriteCount) % cap(buf.data)
		buf.size -= overwriteCount
	}

	for _, d := range newData {
		buf.data[buf.tail] = d
		buf.tail = (buf.tail + 1) % cap(buf.data)
	}

	buf.size = min(buf.size+count, cap(buf.data))
	return nil
}

func (buf *threadSafeBuffer[T]) Read(amt int) []T {
	buf.mutex.Lock()
	defer buf.mutex.Unlock()

	returnSize := min(amt, buf.size)
	output := make([]T, returnSize)

	buf.readIntoUnsafe(output)
	return output
}

func (buf *threadSafeBuffer[T]) ReadInto(s []T) {
	buf.mutex.Lock()
	defer buf.mutex.Unlock()

	buf.readIntoUnsafe(s)
}

// readIntoUnsafe is a helper that doesn't grab the lock
func (buf *threadSafeBuffer[T]) readIntoUnsafe(s []T) {
	returnSize := min(len(s), buf.size)

	for i := range returnSize {
		s[i] = buf.data[buf.head]
		buf.data[buf.head] = *new(T) // free up memory for pointers
		buf.head = (buf.head + 1) % cap(buf.data)
		buf.size--
	}

	// As a courtesy, read in the rest as zeroes
	var zero T
	for i := returnSize; i < len(s); i++ {
		s[i] = zero
	}
}

func (buf *threadSafeBuffer[T]) Size() int {
	buf.mutex.Lock()
	defer buf.mutex.Unlock()
	return buf.size
}
