package main

import (
	goconsistent "github.com/nobound/go-consistent"
)

const (
	replicationFactor = 100
)

type GoConsistentHasher struct {
	hasher *goconsistent.ConsistentHash
}

func NewGoConsistentHasher() *GoConsistentHasher {
	return &GoConsistentHasher{
		hasher: goconsistent.New(goconsistent.Config{
			ReplicationFactor: replicationFactor,
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

func (g *GoConsistentHasher) BucketCount() int {
	return len(g.hasher.GetNodeNames()) / replicationFactor
}
