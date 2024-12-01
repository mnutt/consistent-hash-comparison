package main

import (
	"github.com/stathat/consistent"
)

type ConsistentHasher struct {
	hasher *consistent.Consistent
}

func NewConsistentHasher() *ConsistentHasher {
	hasher := consistent.New()
	hasher.NumberOfReplicas = 100
	return &ConsistentHasher{
		hasher: hasher,
	}
}

func (c *ConsistentHasher) Add(bucket string) {
	c.hasher.Add(bucket)
}

func (c *ConsistentHasher) Remove(bucket string) {
	c.hasher.Remove(bucket)
}

func (c *ConsistentHasher) Get(key string) string {
	bucket, _ := c.hasher.Get(key)
	return bucket
}

func (c *ConsistentHasher) BucketCount() int {
	return len(c.hasher.Members())
}
