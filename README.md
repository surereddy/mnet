MNet
------
Mnet is a collection of superfast networking packages with implementations from ontop of `tcp`, `udp`, and others as planned. It exists to provide a lightweight foundation where other higher level APIs can 
be built on.

## Install

```
go get -v github.com/influx6/mnet/...
```

## Protocols Implemented

### TCP

Mnet has the (mtcp)(./mtcp) package which implements a lightweight network and client structures suitable for fast communication and transfer of data in between, by combining minimal data copy
with data buffering techniques, it ensures you transfer massive data within great nanosecond range.

- [Benchmarks](./mtcp/benchmark.txt)
Below is a benchmark of `mtcp` running with golang version `1.9.2` on OS X 10.12.6.

```bash
BenchmarkNonTLSNetworkWriteWithClient-4    	30000000	        58.5 ns/op	 136.79 MB/s	
BenchmarkNonTLSNetworkWriteWithNetConn-4   	  200000	     10016 ns/op	   0.80 MB/s
BenchmarkNoBytesMessages-4                 	30000000	        55.0 ns/op	  72.70 MB/s
Benchmark2BytesMessages-4                  	20000000	        58.4 ns/op	 102.83 MB/s
Benchmark8BytesMessages-4                  	20000000	        66.6 ns/op	 180.09 MB/s
Benchmark16BytesMessages-4                 	20000000	        73.7 ns/op	 271.25 MB/s
Benchmark32BytesMessages-4                 	20000000	        89.3 ns/op	 403.21 MB/s
Benchmark64BytesMessages-4                 	20000000	       122 ns/op	 553.03 MB/s
Benchmark128BytesMessages-4                	10000000	       224 ns/op	 587.28 MB/s
Benchmark256BytesMessages-4                	 3000000	       461 ns/op	 563.32 MB/s   1 B/op
Benchmark1KMessages-4                      	 1000000	      1840 ns/op	 558.48 MB/s   2 B/op
Benchmark4KMessages-4                      	  200000	      6533 ns/op	 627.51 MB/s   13 B/op
Benchmark8KMessages-4                      	  100000	     14562 ns/op	 562.81 MB/s   26 B/op
````


### UDP

Mnet current has no implementation but is planned.

### Websockets

Mnet current has no implementation but is planned.



## Contributions

Contributors are welcome to file issue tickets and provide PRs in accordance to the following [Guidelines](./contrib.md)