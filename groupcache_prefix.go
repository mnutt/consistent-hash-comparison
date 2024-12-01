package main

import "github.com/bobhansen/groupcache/consistenthash"

type GroupCachePrefixHasher struct {
	hasher *consistenthash.Map
	keys   []string
}

func NewGroupCachePrefixHasher() *GroupCachePrefixHasher {
	return &GroupCachePrefixHasher{
		hasher: consistenthash.New(100, nil),
	}
}

func (g *GroupCachePrefixHasher) Add(bucket string) {
	g.keys = append(g.keys, bucket)
	g.hasher.Add(bucket)
}

func (g *GroupCachePrefixHasher) Remove(bucket string) {
	// Groupcache does not actually support removing a bucket
	// from the hash ring. Instead, we create a new hash ring
	// without the bucket to be removed.
	newHash := consistenthash.New(1, nil)
	for _, key := range g.keys {
		if key != bucket {
			newHash.Add(key)
		}
	}
	// remove key from g.keys
	for i, key := range g.keys {
		if key == bucket {
			g.keys = append(g.keys[:i], g.keys[i+1:]...)
			break
		}
	}
	g.hasher = newHash
}

func (g *GroupCachePrefixHasher) Get(key string) string {
	return g.hasher.Get(key)
}

func (g *GroupCachePrefixHasher) BucketCount() int {
	return len(g.keys)
}
