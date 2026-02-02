package shared

import (
	"errors"
	"iter"
	"maps"
	"sync"
)

// ErrMapFull is returned when the map is full
var ErrMapFull = errors.New("map is at capacity")

// ThreadSafeMap is a map that is thread safe
type ThreadSafeMap[T comparable, K any] interface {
	// Snapshot returns a copy of the data. It returns
	// a copy to prevent accidental thread-unsafe modifications
	Snapshot() map[T]K
	// Set sets a key's value
	Set(key T, value K) error
	// Get gets an item from the map and returns if it was found
	Get(key T) (K, bool)
	// Remove removes an item from the map
	Remove(key T)
	// Size returns the current size of the map
	Size() int
}

type threadSafeMap[T comparable, K any] struct {
	data    map[T]K
	dataCap int

	mu sync.Mutex
}

// NewThreadSafeMap creates a new thread safe map. If the provided data cap is <= 0,
// the map will not cap the data
func NewThreadSafeMap[T comparable, K any](dataCap int) ThreadSafeMap[T, K] {
	var data map[T]K
	if dataCap > 0 {
		data = make(map[T]K, dataCap)
	}

	return &threadSafeMap[T, K]{
		data:    data,
		dataCap: dataCap,
		mu:      sync.Mutex{},
	}
}

func (m *threadSafeMap[T, K]) Snapshot() map[T]K {
	m.mu.Lock()
	defer m.mu.Unlock()

	snap := make(map[T]K, len(m.data))
	maps.Copy(snap, m.data)
	return snap
}

func (m *threadSafeMap[T, K]) Keys() iter.Seq[T] {
	m.mu.Lock()
	defer m.mu.Unlock()
	return maps.Keys(m.data)
}

func (m *threadSafeMap[T, K]) Set(key T, value K) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, exists := m.data[key]
	if !exists && m.dataCap > 0 && len(m.data) == m.dataCap {
		return ErrMapFull
	}

	m.data[key] = value
	return nil
}

func (m *threadSafeMap[T, K]) Get(key T) (K, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	item, ok := m.data[key]
	return item, ok
}

func (m *threadSafeMap[T, K]) Remove(key T) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.data, key)
}

func (m *threadSafeMap[T, K]) Size() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return len(m.data)
}
