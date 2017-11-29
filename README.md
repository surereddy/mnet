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
BenchmarkNonTLSNetworkWriteWithClient-4    	30000000	        59.6 ns/op	 134.24 MB/s	
BenchmarkNonTLSNetworkWriteWithNetConn-4   	  200000	      8926 ns/op	   0.90 MB/s	 
BenchmarkNoBytesMessages-4                 	30000000	        50.2 ns/op	  79.67 MB/s	
Benchmark2BytesMessages-4                  	30000000	        54.4 ns/op	 110.38 MB/s	
Benchmark8BytesMessages-4                  	20000000	        61.1 ns/op	 196.44 MB/s	
Benchmark16BytesMessages-4                 	20000000	        68.7 ns/op	 290.99 MB/s	
Benchmark32BytesMessages-4                 	20000000	        88.8 ns/op	 405.52 MB/s	
Benchmark64BytesMessages-4                 	20000000	        99.7 ns/op	 682.32 MB/s	
Benchmark128BytesMessages-4                	10000000	       189 ns/op	 698.30 MB/s	
Benchmark256BytesMessages-4                	 5000000	       362 ns/op	 717.96 MB/s	
Benchmark1KMessages-4                      	 1000000	      1426 ns/op	 720.77 MB/s	
Benchmark4KMessages-4                      	  300000	      5663 ns/op	 723.95 MB/s	
Benchmark8KMessages-4                      	  200000	     11352 ns/op	 721.96 MB/s	1 B/op
````


### UDP

Mnet current has no implementation but is planned.

### Websockets

Mnet current has no implementation but is planned.



## Contributions

Contributors are welcome to file issue tickets and provide PRs in accordance to the following [Guidelines](./contrib.md)