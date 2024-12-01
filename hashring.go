package main

import (
	"io"

	"github.com/gobwas/hashring"
)

type HashRingHasher struct {
	hasher *hashring.Ring
	count  int
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
	h.count++
}

func (h *HashRingHasher) Remove(bucket string) {
	h.hasher.Delete(HashRingStringItem(bucket))
	h.count--
}

func (h *HashRingHasher) Get(key string) string {
	item := h.hasher.Get(HashRingStringItem(key))
	return string(item.(HashRingStringItem))
}

func (h *HashRingHasher) BucketCount() int {
	return h.count
}
