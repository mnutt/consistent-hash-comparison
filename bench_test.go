package main

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"testing"
	"time"

	"gonum.org/v1/gonum/stat"
)

const CHANGE_FACTOR = 0.5

// Hasher interface that allows multiple consistent hash libraries to be plugged in.
type Hasher interface {
	Add(bucket string)
	Remove(bucket string)
	Get(key string) string
	BucketCount() int
}

// Benchmark function to test consistent hash implementations
func BenchmarkConsistentHash(b *testing.B) {
	bucketCounts := []int{10, 100, 1000}

	libraries := map[string]func() Hasher{
		"MockHasher":             func() Hasher { return NewMockHasher() },
		"DoubleJumpFNVHasher":    func() Hasher { return NewDoubleJumpFNVHasher() },
		"DoubleJumpXXHashHasher": func() Hasher { return NewDoubleJumpXXHashHasher() },
		"DoubleJumpMetroHasher":  func() Hasher { return NewDoubleJumpMetroHasher() },
		"GoConsistentHasher":     func() Hasher { return NewGoConsistentHasher() },
		"HashRingHasher":         func() Hasher { return NewHashRingHasher() },
		"GroupCacheHasher":       func() Hasher { return NewGroupCacheHasher() },
		"GroupCachePrefixHasher": func() Hasher { return NewGroupCachePrefixHasher() },
		"ConsistentHasher":       func() Hasher { return NewConsistentHasher() },
		"AnchorHasher":           func() Hasher { return NewAnchorHasher() },
	}
	libraryKeys := make([]string, 0, len(libraries))
	for libName := range libraries {
		libraryKeys = append(libraryKeys, libName)
	}
	sort.Strings(libraryKeys)

	for _, libName := range libraryKeys {
		libFn := libraries[libName]

		for _, numBuckets := range bucketCounts {
			lib := libFn()

			serverIPs := make([]string, numBuckets)
			serverIPCounts := make(map[string]int)
			for i := 0; i < numBuckets; i++ {
				serverIPs[i] = fmt.Sprintf("192.168.0.%d", i)
				serverIPCounts[serverIPs[i]] = 0
			}

			random := rand.New(rand.NewSource(time.Now().UnixNano()))

			for _, ip := range serverIPs {
				lib.Add(ip)
			}

			sampleUuid := fmt.Sprintf("%x", random.Uint64())
			sampleBucket := lib.Get(sampleUuid)
			for i := 0; i < 100; i++ {
				if lib.Get(sampleUuid) != sampleBucket {
					fmt.Printf("  %s: Inconsistent hash\n", libName)
					break
				}
			}

			// Run the benchmark
			b.Run(fmt.Sprintf("%s-%d-buckets", libName, numBuckets), func(b *testing.B) {
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					uuid := fmt.Sprintf("%x", random.Uint64())
					bucket := lib.Get(uuid)
					serverIPCounts[bucket]++
				}
			})

			// Find the distribution of uuids across the buckets
			var counts []float64
			for _, count := range serverIPCounts {
				counts = append(counts, float64(count))
			}
			stdDev := stat.StdDev(counts, nil)
			mean := stat.Mean(counts, nil)
			cov := stdDev / mean
			fmt.Printf("  distribution: cov: %.4f, mean: %.0f, stddev: %.0f\n", cov, mean, stdDev)

			mappings := make(map[string]string)
			for i := 0; i < 100000; i++ {
				uuid := fmt.Sprintf("%x", random.Uint64())
				bucket := lib.Get(uuid)
				mappings[uuid] = bucket
			}

			changeCount := int(math.Round(float64(numBuckets) * CHANGE_FACTOR))

			// Add 20% more buckets and check consistency of mappings
			start := time.Now()
			for i := 0; i < changeCount; i++ {
				serverIP := fmt.Sprintf("192.168.0.%d", numBuckets+i)
				lib.Add(serverIP)
				serverIPs = append(serverIPs, serverIP)
			}
			fmt.Printf("	adding %d / %d buckets took %s\n", changeCount, lib.BucketCount(), time.Since(start))

			addSameBucketCount := 0
			addTotalCount := len(mappings)
			for uuid, bucket := range mappings {
				if lib.Get(uuid) == bucket {
					addSameBucketCount++
				}
			}
			fmt.Printf("  %d: %d/%d (%.2f%%) keys still map to the same bucket\n", len(serverIPs), addSameBucketCount, addTotalCount, float64(addSameBucketCount)/float64(addTotalCount)*100)

			// Remove some buckets and check consistency of mappings
			start = time.Now()
			for i := 0; i < changeCount; i++ {
				idx := random.Intn(len(serverIPs))
				lib.Remove(serverIPs[idx])
				serverIPs = append(serverIPs[:idx], serverIPs[idx+1:]...)
			}
			fmt.Printf("	removing %d / %d buckets took %s\n", changeCount, lib.BucketCount(), time.Since(start))

			removeSameBucketCount := 0
			removeTotalCount := len(mappings)
			for uuid, bucket := range mappings {
				if lib.Get(uuid) == bucket {
					removeSameBucketCount++
				}
			}
			fmt.Printf("  %d: %d/%d (%.2f%%) keys still map to the same bucket\n", len(serverIPs), removeSameBucketCount, removeTotalCount, float64(removeSameBucketCount)/float64(removeTotalCount)*100)

			// remove every node and add the same number of new ones
			serverIPCount := len(serverIPs)
			for i := 0; i < len(serverIPs); i++ {
				lib.Remove(serverIPs[i])
			}
			// add the same number of new ones, with different values
			for i := 0; i < serverIPCount; i++ {
				serverIP := fmt.Sprintf("192.168.1.%d", i)
				lib.Add(serverIP)
			}

			// Run the benchmark again, in case bucket turnover makes a difference in performance
			b.Run(fmt.Sprintf("%s-%d-buckets-again", libName, numBuckets), func(b *testing.B) {
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					uuid := fmt.Sprintf("%x", random.Uint64())
					bucket := lib.Get(uuid)
					serverIPCounts[bucket]++
				}
			})

			fmt.Printf("\n")
		}
	}
}
