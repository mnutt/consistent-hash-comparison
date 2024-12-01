package main

import (
	"hash"
	"sync"

	"github.com/cespare/xxhash"
)

// MockHasher implementation for testing purposes
type MockHasher struct {
	buckets []string
	mutex   sync.Mutex
	hasher  hash.Hash64
}

func NewMockHasher() *MockHasher {
	return &MockHasher{
		hasher: xxhash.New(),
	}
}

func (m *MockHasher) Add(bucket string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.buckets = append(m.buckets, bucket)
}

func (m *MockHasher) Remove(bucket string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	for i, b := range m.buckets {
		if b == bucket {
			m.buckets = append(m.buckets[:i], m.buckets[i+1:]...)
			return
		}
	}
}

func (m *MockHasher) Get(key string) string {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if len(m.buckets) == 0 {
		return ""
	}
	m.hasher.Reset()
	m.hasher.Write([]byte(key))
	idx := m.hasher.Sum64() % uint64(len(m.buckets))
	return m.buckets[idx]
}

func (m *MockHasher) BucketCount() int {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return len(m.buckets)
}
