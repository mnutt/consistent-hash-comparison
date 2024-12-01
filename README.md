# consistent-hash-comparison

Exploring properties of various consistent hash libraries.

I'm doing this for the purposes of selecting a consistent hash algorithm to use in a distributed write-through cache. The properties I'm looking for:

* Even distribution of keys among buckets: the cache nodes are evenly sized, and having one holding 50% more keys than the median is undesirable
* Good consistency in the face of node additions and removals: nodes come and go, and we want to cause as few cache misses as possible
* Reasonably low overhead lookups

## Running

```
go test -bench -v .
