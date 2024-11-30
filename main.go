package main

import (
	"fmt"
	"hash"
	"hash/fnv"
	"io"
	"math"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cespare/xxhash"
	"github.com/edwingeng/doublejump/v2"
	"github.com/gobwas/hashring"
	goconsistent "github.com/nobound/go-consistent"
)

// Hasher interface that allows multiple consistent hash libraries to be plugged in.
type Hasher interface {
	Add(bucket string)
	Remove(bucket string)
	Get(key string) string
}

// Mock implementation for the consistent hash library.
// Replace this with actual libraries you want to compare.
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
	// Simplified, just truncating from list; real logic could be more complex
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

// DoubleJumpHasher implementation using doublejump library
type DoubleJumpHasher struct {
	hasher *doublejump.Hash[string]
}

func NewDoubleJumpHasher() *DoubleJumpHasher {
	return &DoubleJumpHasher{
		hasher: doublejump.NewHash[string](),
	}
}

func (d *DoubleJumpHasher) Add(bucket string) {
	d.hasher.Add(bucket)
}

func (d *DoubleJumpHasher) Remove(bucket string) {
	d.hasher.Remove(bucket)
}

func (d *DoubleJumpHasher) Get(key string) string {
	hasher := fnv.New64a()
	hasher.Write([]byte(key))
	hashKey := hasher.Sum64()
	bucket, _ := d.hasher.Get(hashKey)
	return bucket
}

type GoConsistentHasher struct {
	hasher *goconsistent.ConsistentHash
}

func NewGoConsistentHasher() *GoConsistentHasher {
	return &GoConsistentHasher{
		hasher: goconsistent.New(goconsistent.Config{
			ReplicationFactor: 1,
		}),
	}
}

func (g *GoConsistentHasher) Add(bucket string) {
	g.hasher.Add(bucket)
}

func (g *GoConsistentHasher) Remove(bucket string) {
	g.hasher.Remove(bucket)
}

func (g *GoConsistentHasher) Get(key string) string {
	return g.hasher.Get(key)
}

type HashRingHasher struct {
	hasher *hashring.Ring
}

type HashRingStringItem string

func (s HashRingStringItem) WriteTo(w io.Writer) (int64, error) {
	n, err := io.WriteString(w, string(s))
	return int64(n), err
}

func NewHashRingHasher() *HashRingHasher {
	return &HashRingHasher{
		hasher: &hashring.Ring{},
	}
}

func (h *HashRingHasher) Add(bucket string) {
	h.hasher.Insert(HashRingStringItem(bucket), 1)
}

func (h *HashRingHasher) Remove(bucket string) {
	h.hasher.Delete(HashRingStringItem(bucket))
}

func (h *HashRingHasher) Get(key string) string {
	item := h.hasher.Get(HashRingStringItem(key))
	return string(item.(HashRingStringItem))
}

func main() {
	// Configure parameters
	numBuckets := 50
	testDuration := 5 * time.Second
	requestRate := 100000 // Requests per second
	numWorkers := 8

	// Set up consistent hash libraries
	libraries := map[string]Hasher{
		"MockHasher":         NewMockHasher(),
		"DoubleJumpHasher":   NewDoubleJumpHasher(),
		"GoConsistentHasher": NewGoConsistentHasher(),
		"HashRingHasher":     NewHashRingHasher(),
		// Add more implementations here
	}

	serverIPs := make([]string, numBuckets)
	for i := 0; i < numBuckets; i++ {
		serverIPs[i] = fmt.Sprintf("192.168.0.%d", i)
	}

	random := rand.New(rand.NewSource(time.Now().UnixNano()))

	for libName, lib := range libraries {
		// Add all buckets to the library
		for _, ip := range serverIPs {
			lib.Add(ip)
		}

		runtime.GC()

		fmt.Printf("Starting test for library: %s\n", libName)

		testValue := "test"
		initialBucket := lib.Get(testValue)
		invalid := false
		for i := 0; i < 1000; i++ {
			bucket := lib.Get(testValue)
			if bucket != initialBucket {
				fmt.Printf("Error: initial bucket %s does not match bucket %s\n", initialBucket, bucket)
				invalid = true
				break
			}
		}
		if invalid {
			continue
		}

		var requestCount int64 = 0
		distribution := sync.Map{}
		var wg sync.WaitGroup
		wg.Add(numWorkers)
		start := time.Now()
		elapsed := int64(0)

		// Start workers
		for i := 0; i < numWorkers; i++ {
			go func() {
				defer wg.Done()
				for time.Since(start) < testDuration {
					uuid := fmt.Sprintf("%x", random.Uint64())
					before := time.Now()
					bucket := lib.Get(uuid)
					atomic.AddInt64(&elapsed, time.Since(before).Nanoseconds())
					distributionValue, _ := distribution.LoadOrStore(bucket, new(int64))
					atomic.AddInt64(distributionValue.(*int64), 1)
					atomic.AddInt64(&requestCount, 1)
					time.Sleep(time.Second / time.Duration(requestRate))
				}
			}()
		}

		// Wait for workers to finish
		wg.Wait()

		// Measure results
		fmt.Printf("Library: %s\n", libName)
		fmt.Printf("Total Requests: %d\n", requestCount)
		nsPerOp := float64(elapsed) / float64(requestCount)
		fmt.Printf("%f ns/op\n", nsPerOp)

		// Measure distribution
		var counts []int64
		distribution.Range(func(key, value interface{}) bool {
			counts = append(counts, atomic.LoadInt64(value.(*int64)))
			return true
		})

		if len(counts) > 0 {
			var sum, sumSquares int64
			min := counts[0]
			max := counts[0]
			for _, count := range counts {
				sum += count
				sumSquares += count * count
				if count < min {
					min = count
				}
				if count > max {
					max = count
				}
			}
			mean := float64(sum) / float64(len(counts))
			variance := float64(sumSquares)/float64(len(counts)) - mean*mean
			stdDev := math.Sqrt(variance)

			fmt.Printf("Bucket Distribution Statistics:\n")
			fmt.Printf("Min: %d\n", min)
			fmt.Printf("Max: %d\n", max)
			fmt.Printf("Standard Deviation: %f\n", stdDev)
		}
		fmt.Println()
	}
}
