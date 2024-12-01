package main

import (
	"hash/fnv"

	anchorhash "github.com/wdamron/go-anchorhash"
)

type AnchorHasher struct {
	hasher  *anchorhash.Anchor
	buckets map[uint32]string
}

func NewAnchorHasher() *AnchorHasher {
	return &AnchorHasher{
		hasher:  anchorhash.NewAnchor(10000, 100),
		buckets: make(map[uint32]string),
	}
}

func (a *AnchorHasher) Add(bucket string) {
	bucketID := a.hasher.AddBucket()
	a.buckets[bucketID] = bucket
}

func (a *AnchorHasher) Remove(bucket string) {
	for id, b := range a.buckets {
		if b == bucket {
			a.hasher.RemoveBucket(id)
			delete(a.buckets, id)
			return
		}
	}
}

func (a *AnchorHasher) Get(key string) string {
	// hash key string to uint32 using fnv
	hasher := fnv.New32a()
	hasher.Write([]byte(key))
	hashKey := hasher.Sum32()

	bucketID := a.hasher.GetBucket(uint64(hashKey))
	return a.buckets[bucketID]
}

func (a *AnchorHasher) BucketCount() int {
	return len(a.buckets)
}
