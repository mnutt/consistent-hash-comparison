package main

import (
	"hash/fnv"

	"github.com/cespare/xxhash"
	"github.com/dgryski/go-metro"
	"github.com/edwingeng/doublejump/v2"
)

// NewDoubleJumpFNVHasher implementation using DoubleJumpFNV library
type DoubleJumpFNVHasher struct {
	hasher *doublejump.Hash[string]
}

func NewDoubleJumpFNVHasher() *DoubleJumpFNVHasher {
	return &DoubleJumpFNVHasher{
		hasher: doublejump.NewHash[string](),
	}
}

func (d *DoubleJumpFNVHasher) Add(bucket string) {
	d.hasher.Add(bucket)
}

func (d *DoubleJumpFNVHasher) Remove(bucket string) {
	d.hasher.Remove(bucket)
}

func (d *DoubleJumpFNVHasher) Get(key string) string {
	hasher := fnv.New64a()
	hasher.Write([]byte(key))
	hashKey := hasher.Sum64()
	bucket, _ := d.hasher.Get(hashKey)
	return bucket
}

func (d *DoubleJumpFNVHasher) BucketCount() int {
	return d.hasher.Len()
}

type DoubleJumpXXHashHasher struct {
	hasher *doublejump.Hash[string]
}

func NewDoubleJumpXXHashHasher() *DoubleJumpXXHashHasher {
	return &DoubleJumpXXHashHasher{
		hasher: doublejump.NewHash[string](),
	}
}

func (d *DoubleJumpXXHashHasher) Add(bucket string) {
	d.hasher.Add(bucket)
}

func (d *DoubleJumpXXHashHasher) Remove(bucket string) {
	d.hasher.Remove(bucket)
}

func (d *DoubleJumpXXHashHasher) Get(key string) string {
	hasher := xxhash.New()
	hasher.Write([]byte(key))
	hashKey := hasher.Sum64()
	bucket, _ := d.hasher.Get(hashKey)
	return bucket
}

func (d *DoubleJumpXXHashHasher) BucketCount() int {
	return d.hasher.Len()
}

type DoubleJumpMetroHasher struct {
	hasher *doublejump.Hash[string]
}

func NewDoubleJumpMetroHasher() *DoubleJumpMetroHasher {
	return &DoubleJumpMetroHasher{
		hasher: doublejump.NewHash[string](),
	}
}

func (d *DoubleJumpMetroHasher) Add(bucket string) {
	d.hasher.Add(bucket)
}

func (d *DoubleJumpMetroHasher) Remove(bucket string) {
	d.hasher.Remove(bucket)
}

func (d *DoubleJumpMetroHasher) Get(key string) string {
	hashKey := metro.Hash64([]byte(key), 0)
	bucket, _ := d.hasher.Get(hashKey)
	return bucket
}

func (d *DoubleJumpMetroHasher) BucketCount() int {
	return d.hasher.Len()
}
