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
BenchmarkNonTLSNetworkWriteWithClient-4    	20000000	        55.5 ns/op	 144.01 MB/s	       
BenchmarkNonTLSNetworkWriteWithNetConn-4   	  200000	      9195 ns/op	   0.87 MB/s	      72 B/op	       2 allocs/op
```

```bash
BenchmarkNoBytesMessages-4                 	30000000	        52.5 ns/op	  76.15 MB/s	       
Benchmark2BytesMessages-4                  	30000000	        59.2 ns/op	 101.39 MB/s	       
Benchmark8BytesMessages-4                  	20000000	        67.2 ns/op	 178.47 MB/s	       
Benchmark16BytesMessages-4                 	20000000	        71.4 ns/op	 279.97 MB/s	       
Benchmark32BytesMessages-4                 	20000000	        91.3 ns/op	 394.12 MB/s	       
Benchmark64BytesMessages-4                 	10000000	       129 ns/op	 523.85 MB/s	       
Benchmark128BytesMessages-4                	10000000	       237 ns/op	 554.98 MB/s	       
Benchmark256BytesMessages-4                	 3000000	       416 ns/op	 624.44 MB/s	       
Benchmark1KMessages-4                      	 1000000	      1761 ns/op	 583.56 MB/s	       
Benchmark4KMessages-4                      	  300000	      4974 ns/op	 824.28 MB/s	       
Benchmark8KMessages-4                      	  300000	      4673 ns/op	1753.75 MB/s	       
````


### UDP

Mnet current has no implementation but is planned.

### Websockets

Mnet current has no implementation but is planned.



## Contributions

Contributors are welcome to file issue tickets and provide PRs in accordance to the following [Guidelines](./contrib.md)