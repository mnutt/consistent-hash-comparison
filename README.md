# consistent-hash-comparison

Exploring properties of various consistent hash libraries.

I'm doing this for the purposes of selecting a consistent hash algorithm to use in a distributed write-through cache. The properties I'm looking for:

* Even distribution of keys among buckets: the cache nodes are evenly sized, and having one holding 50% more keys than the median is undesirable
* Good consistency in the face of node additions and removals: nodes come and go, and we want to cause as few cache misses as possible
* Reasonably low overhead lookups

## Running

```
go test -bench -v .
```

# Results

## Summary

| Metric                   | ns/op  | ns/op after full turnover | Coefficient of Variation | Adding 50 buckets (µs)   | Consistency afterwards   | Removing 50 buckets (µs) | Consistency afterwards   |
|--------------------------|--------|---------------------------|--------------------------|--------------------------|--------------------------|--------------------------|--------------------------|
| Consistent               | 193.9  | 194.9                     | 15.96%                   | 56,681                   | 69.88%                   | 56,780                   | 49.13%                   |
| DoubleJump - FNV         | 154.9  | 180.6                     | 0.35%                    | 16                       | 66.75%                   | 18                       | 41.42%                   |
| DoubleJump - Metro64     | 140.5  | 162.8                     | 0.32%                    | 16                       | 66.88%                   | 15                       | 44.71%                   |
| GoConsistent             | 1131.0 | 1180.0                    | 9.29%                    | 444,522                  | 66.80%                   | 443,878                  | 55.79%                   |
| GroupCache               | 212.5  | 156.5                     | 15.91%                   | 21,681                   | 69.79%                   | 1547                     | 0.74%                    |
| GroupCache - Prefix      | 145.7  | 122.9                     | 15.95%                   | 80,924                   | 69.89%                   | 37,476                   | 0.77%                    |
| HashRing                 | 523.4  | 524.2                     | 3.23%                    | 63,440                   | 67.15%                   | 63,470                   | 49.41%                   |

## Notes

### Consistent
[github.com/stathat/consistent](https://github.com/stathat/consistent)
* The buckets were fairly unbalanced with this one. In this case a mean of 70k keys with a standard deviation of 11k keys. Not a deal breaker, but not great.
* Bucket additions and removals taking ~50ms was also ok-but-not-great.
* Consistency after adding/removing buckets was also middle of the pack. Pretty well describes this library overall.

### DoubleJump
[github.com/edwingeng/doublejump/v2](https://github.com/edwingeng/doublejump)
* In 2014, Google came out with a ["Jump Hash" paper] that described an algorithm improving on consistent hashing, in both simplicity and performance.
* The only problem is that it made these gains by discarding bucket removal. "Double Jump Hash" is an improvement that [adds back bucket removal](https://docs.google.com/presentation/d/e/2PACX-1vTHyFGUJ5CBYxZTzToc_VKxP_Za85AeZqQMNGLXFLP1tX0f9IF_z3ys9-pyKf-Jj3iWpm7dUDDaoFyb/pub?start=false&loop=false&delayms=3000#slide=id.g472441f6aa_0_202).
* Reading how they accomplished this, it's unsurprising that adding and removing nodes causes a dip in key->bucket lookup. But it's really not all that significant overall.
  * I ran some extra benchmarks to ensure that a single "full turnover" (removing every node and adding the same number of new ones) has the same performance whether it turns over once, or thousands of times.
* Jump Hash also assumes keys will be int64s, which requires a prehashing step from string to int64. Metro64 seems to be the fastest I could find.
* That said, adding and removing nodes is very, very fast compared to all of the other libraries
* Consistency after removing buckets is slightly less than the leaders for that metric.

### GoConsistent
[github.com/nobound/go-consistent](https://github.com/nobound/go-consistent)
* The key->bucket lookup time is the worst of all libraries tested
* Somewhat unbalanced buckets
* Adding and removing buckets also takes an order of magnitude longer than any other library
* Consistency is actually the best of all libraries tested, although not so much better to offset the other downsides
  * It is one of the few whose consistency doesn't trend linearly downwards for every bucket change operation
  * Not having read the code, my best guess here is that keys that were reassigned when buckets were removed are more likely to be assigned to the newly added buckets

### GroupCache
[github.com/golang/groupcache/consistenthash](https://github.com/golang/groupcache)
* Algorithm that is used in groupcache
* Also does not allow for bucket removals
  * In fact, groupcache itself does not allow additions _or_ removals. SetPeers() just recreates the hash
  * This destroys any consistency. I'm not totally sure if they benefit from a consistent hash at all, compared to just using regular hashing?
* There is a [PR](https://github.com/golang/groupcache/pull/103) that adds prefix match, which makes key lookup times very fast, but doesn't solve for the other drawbacks

### HashRing
[github.com/gobwas/hashring](github.com/gobwas/hashring)
* Second worst key lookup times
* Decent bucket distribution
* Average

## Detailed Benchmark Run

This includes tests with 10 and 1000 buckets in addition to 100.

```
❯ go test -v -bench .
goos: darwin
goarch: arm64
pkg: github.com/movableink/consistent-hash-comparison
cpu: Apple M1 Pro
BenchmarkConsistentHash
BenchmarkConsistentHash/ConsistentHasher-10-buckets
BenchmarkConsistentHash/ConsistentHasher-10-buckets-10     	 7111552	       168.2 ns/op
  distribution: cov: 0.1043, mean: 812165, stddev: 84748
	adding 5 / 15 buckets took 495.916µs
  15: 67583/100000 (67.58%) keys still map to the same bucket
	removing 5 / 10 buckets took 496.125µs
  10: 67593/100000 (67.59%) keys still map to the same bucket
BenchmarkConsistentHash/ConsistentHasher-10-buckets-pt-2
BenchmarkConsistentHash/ConsistentHasher-10-buckets-pt-2-10         	 7167723	       169.7 ns/op

BenchmarkConsistentHash/ConsistentHasher-100-buckets
BenchmarkConsistentHash/ConsistentHasher-100-buckets-10             	 6023625	       193.9 ns/op
  distribution: cov: 0.1596, mean: 70337, stddev: 11222
	adding 50 / 150 buckets took 56.681125ms
  150: 69883/100000 (69.88%) keys still map to the same bucket
	removing 50 / 100 buckets took 56.7805ms
  100: 49135/100000 (49.13%) keys still map to the same bucket
BenchmarkConsistentHash/ConsistentHasher-100-buckets-pt-2
BenchmarkConsistentHash/ConsistentHasher-100-buckets-pt-2-10        	 6160441	       194.9 ns/op

BenchmarkConsistentHash/ConsistentHasher-1000-buckets
BenchmarkConsistentHash/ConsistentHasher-1000-buckets-10            	 4749271	       245.5 ns/op
  distribution: cov: 0.2339, mean: 5759, stddev: 1347
	adding 500 / 1500 buckets took 6.852861708s
  1500: 66699/100000 (66.70%) keys still map to the same bucket
	removing 500 / 1000 buckets took 6.764329083s
  1000: 50406/100000 (50.41%) keys still map to the same bucket
BenchmarkConsistentHash/ConsistentHasher-1000-buckets-pt-2
BenchmarkConsistentHash/ConsistentHasher-1000-buckets-pt-2-10       	 4859574	       257.4 ns/op

BenchmarkConsistentHash/DoubleJumpFNVHasher-10-buckets
BenchmarkConsistentHash/DoubleJumpFNVHasher-10-buckets-10           	 8727534	       133.5 ns/op
  distribution: cov: 0.0010, mean: 973764, stddev: 997
	adding 5 / 15 buckets took 4.417µs
  15: 66797/100000 (66.80%) keys still map to the same bucket
	removing 5 / 10 buckets took 2.791µs
  10: 40414/100000 (40.41%) keys still map to the same bucket
BenchmarkConsistentHash/DoubleJumpFNVHasher-10-buckets-pt-2
BenchmarkConsistentHash/DoubleJumpFNVHasher-10-buckets-pt-2-10      	 8111766	       147.5 ns/op

BenchmarkConsistentHash/DoubleJumpFNVHasher-100-buckets
BenchmarkConsistentHash/DoubleJumpFNVHasher-100-buckets-10          	 7749918	       154.9 ns/op
  distribution: cov: 0.0035, mean: 87600, stddev: 307
	adding 50 / 150 buckets took 16.042µs
  150: 66752/100000 (66.75%) keys still map to the same bucket
	removing 50 / 100 buckets took 17.792µs
  100: 41420/100000 (41.42%) keys still map to the same bucket
BenchmarkConsistentHash/DoubleJumpFNVHasher-100-buckets-pt-2
BenchmarkConsistentHash/DoubleJumpFNVHasher-100-buckets-pt-2-10     	 6623568	       180.6 ns/op

BenchmarkConsistentHash/DoubleJumpFNVHasher-1000-buckets
BenchmarkConsistentHash/DoubleJumpFNVHasher-1000-buckets-10         	 6923325	       172.6 ns/op
  distribution: cov: 0.0112, mean: 7933, stddev: 89
	adding 500 / 1500 buckets took 95.667µs
  1500: 66455/100000 (66.45%) keys still map to the same bucket
	removing 500 / 1000 buckets took 164.25µs
  1000: 45115/100000 (45.12%) keys still map to the same bucket
BenchmarkConsistentHash/DoubleJumpFNVHasher-1000-buckets-pt-2
BenchmarkConsistentHash/DoubleJumpFNVHasher-1000-buckets-pt-2-10    	 5880048	       203.7 ns/op

BenchmarkConsistentHash/DoubleJumpMetroHasher-10-buckets
BenchmarkConsistentHash/DoubleJumpMetroHasher-10-buckets-10         	 9939873	       121.0 ns/op
  distribution: cov: 0.0009, mean: 1094997, stddev: 975
	adding 5 / 15 buckets took 5.667µs
  15: 66365/100000 (66.36%) keys still map to the same bucket
	removing 5 / 10 buckets took 3.292µs
  10: 54843/100000 (54.84%) keys still map to the same bucket
BenchmarkConsistentHash/DoubleJumpMetroHasher-10-buckets-pt-2
BenchmarkConsistentHash/DoubleJumpMetroHasher-10-buckets-pt-2-10    	 8569267	       137.9 ns/op

BenchmarkConsistentHash/DoubleJumpMetroHasher-100-buckets
BenchmarkConsistentHash/DoubleJumpMetroHasher-100-buckets-10        	 8529774	       140.5 ns/op
  distribution: cov: 0.0032, mean: 95399, stddev: 307
	adding 50 / 150 buckets took 16.667µs
  150: 66883/100000 (66.88%) keys still map to the same bucket
	removing 50 / 100 buckets took 14.5µs
  100: 44708/100000 (44.71%) keys still map to the same bucket
BenchmarkConsistentHash/DoubleJumpMetroHasher-100-buckets-pt-2
BenchmarkConsistentHash/DoubleJumpMetroHasher-100-buckets-pt-2-10   	 7322139	       162.8 ns/op

BenchmarkConsistentHash/DoubleJumpMetroHasher-1000-buckets
BenchmarkConsistentHash/DoubleJumpMetroHasher-1000-buckets-10       	 7496836	       161.1 ns/op
  distribution: cov: 0.0108, mean: 8507, stddev: 92
	adding 500 / 1500 buckets took 101.75µs
  1500: 66650/100000 (66.65%) keys still map to the same bucket
	removing 500 / 1000 buckets took 173.042µs
  1000: 43483/100000 (43.48%) keys still map to the same bucket
BenchmarkConsistentHash/DoubleJumpMetroHasher-1000-buckets-pt-2
BenchmarkConsistentHash/DoubleJumpMetroHasher-1000-buckets-pt-2-10  	 6080767	       194.5 ns/op

BenchmarkConsistentHash/DoubleJumpXXHashHasher-10-buckets
BenchmarkConsistentHash/DoubleJumpXXHashHasher-10-buckets-10        	 9285385	       129.3 ns/op
  distribution: cov: 0.0010, mean: 1029549, stddev: 1040
	adding 5 / 15 buckets took 5.208µs
  15: 66649/100000 (66.65%) keys still map to the same bucket
	removing 5 / 10 buckets took 4.5µs
  10: 40545/100000 (40.54%) keys still map to the same bucket
BenchmarkConsistentHash/DoubleJumpXXHashHasher-10-buckets-pt-2
BenchmarkConsistentHash/DoubleJumpXXHashHasher-10-buckets-pt-2-10   	 8329360	       142.1 ns/op

BenchmarkConsistentHash/DoubleJumpXXHashHasher-100-buckets
BenchmarkConsistentHash/DoubleJumpXXHashHasher-100-buckets-10       	 8013937	       151.1 ns/op
  distribution: cov: 0.0035, mean: 90240, stddev: 319
	adding 50 / 150 buckets took 15.542µs
  150: 66689/100000 (66.69%) keys still map to the same bucket
	removing 50 / 100 buckets took 22.167µs
  100: 44868/100000 (44.87%) keys still map to the same bucket
BenchmarkConsistentHash/DoubleJumpXXHashHasher-100-buckets-pt-2
BenchmarkConsistentHash/DoubleJumpXXHashHasher-100-buckets-pt-2-10  	 6771189	       173.7 ns/op

BenchmarkConsistentHash/DoubleJumpXXHashHasher-1000-buckets
BenchmarkConsistentHash/DoubleJumpXXHashHasher-1000-buckets-10      	 7104864	       168.7 ns/op
  distribution: cov: 0.0111, mean: 8115, stddev: 90
	adding 500 / 1500 buckets took 97.708µs
  1500: 66654/100000 (66.65%) keys still map to the same bucket
	removing 500 / 1000 buckets took 159.667µs
  1000: 45370/100000 (45.37%) keys still map to the same bucket
BenchmarkConsistentHash/DoubleJumpXXHashHasher-1000-buckets-pt-2
BenchmarkConsistentHash/DoubleJumpXXHashHasher-1000-buckets-pt-2-10 	 5984286	       200.2 ns/op

BenchmarkConsistentHash/GoConsistentHasher-10-buckets
BenchmarkConsistentHash/GoConsistentHasher-10-buckets-10            	 1266733	       945.6 ns/op
  distribution: cov: 0.1280, mean: 227683, stddev: 29143
	adding 5 / 15 buckets took 4.545833ms
  15: 67049/100000 (67.05%) keys still map to the same bucket
	removing 5 / 10 buckets took 4.297917ms
  10: 82818/100000 (82.82%) keys still map to the same bucket
BenchmarkConsistentHash/GoConsistentHasher-10-buckets-pt-2
BenchmarkConsistentHash/GoConsistentHasher-10-buckets-pt-2-10       	 1265634	       950.8 ns/op

BenchmarkConsistentHash/GoConsistentHasher-100-buckets
BenchmarkConsistentHash/GoConsistentHasher-100-buckets-10           	 1000000	      1131 ns/op
  distribution: cov: 0.0929, mean: 10101, stddev: 938
	adding 50 / 150 buckets took 444.522667ms
  150: 66795/100000 (66.80%) keys still map to the same bucket
	removing 50 / 100 buckets took 443.877958ms
  100: 55792/100000 (55.79%) keys still map to the same bucket
BenchmarkConsistentHash/GoConsistentHasher-100-buckets-pt-2
BenchmarkConsistentHash/GoConsistentHasher-100-buckets-pt-2-10      	 1000000	      1180 ns/op

BenchmarkConsistentHash/GoConsistentHasher-1000-buckets
BenchmarkConsistentHash/GoConsistentHasher-1000-buckets-10          	  573572	      2259 ns/op
  distribution: cov: 0.1069, mean: 584, stddev: 62
	adding 500 / 1500 buckets took 51.043394375s
  1500: 66717/100000 (66.72%) keys still map to the same bucket
	removing 500 / 1000 buckets took 51.122104792s
  1000: 48651/100000 (48.65%) keys still map to the same bucket
BenchmarkConsistentHash/GoConsistentHasher-1000-buckets-pt-2
BenchmarkConsistentHash/GoConsistentHasher-1000-buckets-pt-2-10     	  577809	      1930 ns/op

BenchmarkConsistentHash/GroupCacheHasher-10-buckets
BenchmarkConsistentHash/GroupCacheHasher-10-buckets-10              	 7239054	       162.9 ns/op
  distribution: cov: 0.1045, mean: 824916, stddev: 86172
	adding 5 / 15 buckets took 217.416µs
  15: 67570/100000 (67.57%) keys still map to the same bucket
	removing 5 / 10 buckets took 13.792µs
  10: 5296/100000 (5.30%) keys still map to the same bucket
BenchmarkConsistentHash/GroupCacheHasher-10-buckets-pt-2
BenchmarkConsistentHash/GroupCacheHasher-10-buckets-pt-2-10         	10030623	       119.5 ns/op

BenchmarkConsistentHash/GroupCacheHasher-100-buckets
BenchmarkConsistentHash/GroupCacheHasher-100-buckets-10             	 6215991	       212.5 ns/op
  distribution: cov: 0.1591, mean: 72261, stddev: 11500
	adding 50 / 150 buckets took 21.681458ms
  150: 69791/100000 (69.79%) keys still map to the same bucket
	removing 50 / 100 buckets took 1.546875ms
  100: 742/100000 (0.74%) keys still map to the same bucket
BenchmarkConsistentHash/GroupCacheHasher-100-buckets-pt-2
BenchmarkConsistentHash/GroupCacheHasher-100-buckets-pt-2-10        	 7860885	       156.5 ns/op

BenchmarkConsistentHash/GroupCacheHasher-1000-buckets
BenchmarkConsistentHash/GroupCacheHasher-1000-buckets-10            	 4746799	       236.2 ns/op
  distribution: cov: 0.2337, mean: 5757, stddev: 1345
	adding 500 / 1500 buckets took 2.830133333s
  1500: 66819/100000 (66.82%) keys still map to the same bucket
	removing 500 / 1000 buckets took 647.914583ms
  1000: 590/100000 (0.59%) keys still map to the same bucket
BenchmarkConsistentHash/GroupCacheHasher-1000-buckets-pt-2
BenchmarkConsistentHash/GroupCacheHasher-1000-buckets-pt-2-10       	 6962011	       174.8 ns/op

BenchmarkConsistentHash/GroupCachePrefixHasher-10-buckets
BenchmarkConsistentHash/GroupCachePrefixHasher-10-buckets-10         	 9337237	       125.1 ns/op
  distribution: cov: 0.1048, mean: 1034734, stddev: 108395
	adding 5 / 15 buckets took 763.916µs
  15: 67605/100000 (67.61%) keys still map to the same bucket
	removing 5 / 10 buckets took 67.583µs
  10: 6076/100000 (6.08%) keys still map to the same bucket
BenchmarkConsistentHash/GroupCachePrefixHasher-10-buckets-again
BenchmarkConsistentHash/GroupCachePrefixHasher-10-buckets-again-10   	10784682	       110.3 ns/op

BenchmarkConsistentHash/GroupCachePrefixHasher-100-buckets
BenchmarkConsistentHash/GroupCachePrefixHasher-100-buckets-10        	 8553729	       145.7 ns/op
  distribution: cov: 0.1595, mean: 95638, stddev: 15256
	adding 50 / 150 buckets took 80.924917ms
  150: 69888/100000 (69.89%) keys still map to the same bucket
	removing 50 / 100 buckets took 37.47825ms
  100: 765/100000 (0.77%) keys still map to the same bucket
BenchmarkConsistentHash/GroupCachePrefixHasher-100-buckets-again
BenchmarkConsistentHash/GroupCachePrefixHasher-100-buckets-again-10  	 9875851	       122.9 ns/op

BenchmarkConsistentHash/GroupCachePrefixHasher-1000-buckets
BenchmarkConsistentHash/GroupCachePrefixHasher-1000-buckets-10       	 5717476	       210.6 ns/op
  distribution: cov: 0.2341, mean: 6728, stddev: 1575
	adding 500 / 1500 buckets took 9.152211583s
  1500: 66884/100000 (66.88%) keys still map to the same bucket
	removing 500 / 1000 buckets took 31.68780025s
  1000: 575/100000 (0.57%) keys still map to the same bucket
BenchmarkConsistentHash/GroupCachePrefixHasher-1000-buckets-again
BenchmarkConsistentHash/GroupCachePrefixHasher-1000-buckets-again-10 	 9174766	       130.1 ns/op

BenchmarkConsistentHash/HashRingHasher-10-buckets
BenchmarkConsistentHash/HashRingHasher-10-buckets-10                	 3902038	       279.0 ns/op
  distribution: cov: 0.0237, mean: 491214, stddev: 11642
	adding 5 / 15 buckets took 3.977708ms
  15: 67316/100000 (67.32%) keys still map to the same bucket
	removing 5 / 10 buckets took 3.016666ms
  10: 33744/100000 (33.74%) keys still map to the same bucket
BenchmarkConsistentHash/HashRingHasher-10-buckets-pt-2
BenchmarkConsistentHash/HashRingHasher-10-buckets-pt-2-10           	 4231485	       283.1 ns/op

BenchmarkConsistentHash/HashRingHasher-100-buckets
BenchmarkConsistentHash/HashRingHasher-100-buckets-10               	 2416658	       523.4 ns/op
  distribution: cov: 0.0323, mean: 34268, stddev: 1106
	adding 50 / 150 buckets took 63.440792ms
  150: 67151/100000 (67.15%) keys still map to the same bucket
	removing 50 / 100 buckets took 63.47025ms
  100: 49407/100000 (49.41%) keys still map to the same bucket
BenchmarkConsistentHash/HashRingHasher-100-buckets-pt-2
BenchmarkConsistentHash/HashRingHasher-100-buckets-pt-2-10          	 2401882	       524.2 ns/op

BenchmarkConsistentHash/HashRingHasher-1000-buckets
BenchmarkConsistentHash/HashRingHasher-1000-buckets-10              	  825850	      1354 ns/op
  distribution: cov: 0.0456, mean: 836, stddev: 38
	adding 500 / 1500 buckets took 1.198535125s
  1500: 66470/100000 (66.47%) keys still map to the same bucket
	removing 500 / 1000 buckets took 1.135620208s
  1000: 46808/100000 (46.81%) keys still map to the same bucket
BenchmarkConsistentHash/HashRingHasher-1000-buckets-pt-2
BenchmarkConsistentHash/HashRingHasher-1000-buckets-pt-2-10         	  840664	      1303 ns/op

BenchmarkConsistentHash/MockHasher-10-buckets
BenchmarkConsistentHash/MockHasher-10-buckets-10                    	 9618852	       127.1 ns/op
  distribution: cov: 0.0009, mean: 1062895, stddev: 1007
	adding 5 / 15 buckets took 2.084µs
  15: 33127/100000 (33.13%) keys still map to the same bucket
	removing 5 / 10 buckets took 2.375µs
  10: 20015/100000 (20.02%) keys still map to the same bucket
BenchmarkConsistentHash/MockHasher-10-buckets-pt-2
BenchmarkConsistentHash/MockHasher-10-buckets-pt-2-10               	 9413370	       125.2 ns/op

BenchmarkConsistentHash/MockHasher-100-buckets
BenchmarkConsistentHash/MockHasher-100-buckets-10                   	 9434217	       129.1 ns/op
  distribution: cov: 0.0029, mean: 104443, stddev: 301
	adding 50 / 150 buckets took 6.458µs
  150: 33445/100000 (33.45%) keys still map to the same bucket
	removing 50 / 100 buckets took 9.375µs
  100: 4016/100000 (4.02%) keys still map to the same bucket
BenchmarkConsistentHash/MockHasher-100-buckets-pt-2
BenchmarkConsistentHash/MockHasher-100-buckets-pt-2-10              	 9344893	       129.2 ns/op

BenchmarkConsistentHash/MockHasher-1000-buckets
BenchmarkConsistentHash/MockHasher-1000-buckets-10                  	 9046532	       132.0 ns/op
  distribution: cov: 0.0099, mean: 10057, stddev: 100
	adding 500 / 1500 buckets took 36.458µs
  1500: 33416/100000 (33.42%) keys still map to the same bucket
	removing 500 / 1000 buckets took 530µs
  1000: 500/100000 (0.50%) keys still map to the same bucket
BenchmarkConsistentHash/MockHasher-1000-buckets-pt-2
BenchmarkConsistentHash/MockHasher-1000-buckets-pt-2-10             	 9025242	       132.5 ns/op

PASS
ok  	github.com/movableink/consistent-hash-comparison	343.782s
```
