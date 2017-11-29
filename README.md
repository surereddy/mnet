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

Mnet has the (mtcp)(./mtcp) package which implements a lightweight network and client structures suitable for fast communication and transfer of data in between, by combining zero data copying
with data buffering techniques, it ensures you transfer massive data within the nanosecond range.

- [Benchmarks](./mtcp/benchmark.txt)
Below is a benchmark of `mtcp` running with golang version `1.9.2` on OS X 10.12.6.

```bash
goos: darwin
goarch: amd64
pkg: github.com/influx6/mnet/mtcp
BenchmarkNonTLSNetworkWriteWithClient-4    	30000000	        56.0 ns/op	 142.83 MB/s	       0 B/op	       0 allocs/op
BenchmarkNonTLSNetworkWriteWithNetConn-4   	  200000	     10098 ns/op	   0.79 MB/s	     105 B/op	       3 allocs/op
PASS
ok  	github.com/influx6/mnet/mtcp	7.249s
````


### UDP

Mnet current has no implementation but is planned.

### Websockets

Mnet current has no implementation but is planned.



## Contributions

Contributors are welcome to file issue tickets and provide PRs in accordance to the following [Guidelines](./contrib.md)