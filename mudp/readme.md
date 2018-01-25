MUDP
-------
MUDP implements a udp network server and client library for the `mnet` package to provide superfast reads and writes.

## Benchmarks

Below are recently runned benchmarks, see [BenchmarkTests](./benchmark.txt) for older runs.

```bash
goos: darwin
goarch: amd64
pkg: github.com/influx6/mnet/mudp
BenchmarkNoBytesMessages-4    	 3000000	       433 ns/op	  13.85 MB/s	      16 B/op	       1 allocs/op
Benchmark2BytesMessages-4     	 3000000	       435 ns/op	  18.38 MB/s	      16 B/op	       1 allocs/op
Benchmark4BytesMessages-4     	 3000000	       442 ns/op	  22.59 MB/s	      16 B/op	       1 allocs/op
Benchmark8BytesMessages-4     	 3000000	       450 ns/op	  31.05 MB/s	      16 B/op	       1 allocs/op
Benchmark16BytesMessages-4    	 3000000	       429 ns/op	  51.22 MB/s	      16 B/op	       1 allocs/op
Benchmark32BytesMessages-4    	 3000000	       446 ns/op	  85.15 MB/s	      16 B/op	       1 allocs/op
Benchmark64BytesMessages-4    	 3000000	       448 ns/op	 156.16 MB/s	      16 B/op	       1 allocs/op
Benchmark128BytesMessages-4   	 3000000	       445 ns/op	 300.74 MB/s	      16 B/op	       1 allocs/op
Benchmark256BytesMessages-4   	 3000000	       435 ns/op	 601.86 MB/s	      16 B/op	       1 allocs/op
Benchmark1KMessages-4         	 3000000	       439 ns/op	2345.05 MB/s	      16 B/op	       1 allocs/op
Benchmark4KMessages-4         	 3000000	       455 ns/op	9001.28 MB/s	      16 B/op	       1 allocs/op
Benchmark8KMessages-4         	 3000000	       466 ns/op	17580.83 MB/s	      16 B/op	       1 allocs/op
Benchmark16KMessages-4        	 3000000	       460 ns/op	35576.56 MB/s	      16 B/op	       1 allocs/op
PASS
ok  	github.com/influx6/mnet/mudp	23.583
```

## Examples

#### MUDP Server
`mudp.Network` provides a udp network server which readily through a handler function allows handling incoming client connections, as below: 

```go
var netw mudp.Network
netw.Network = "udp"
netw.Addr = "localhost:5050"
netw.Handler = func(client mnet.Client) error {
    // Flush all incoming data out
    for {
        _, err := client.Read()
        if err != nil {
            if err == mnet.ErrNoDataYet {
                time.Sleep(300 * time.Millisecond)
                continue
            }

            return err
        }

        writer.Write("welcome")
        writer.Flush()
    }
}

```

#### Client

`mudp.Connect` provides a udp client which connects to a `mudp.Network` server, ready to allow user's communication at blazing speeds.

```go
client, err := mudp.Connect("localhost:4050")
if err != nil {
    log.Fatalf(err)
    return
}

// create writer by telling client size of data
// to be written.
writer, err := client.Write(10)
if err != nil {
    log.Fatalf(err)
    return
}

writer.Write([]byte("pub help"))
if err := writer.Close(); err != nil {
    log.Fatalf(err)
    return
}

if _, err := client.Flush(); err != nil {
    log.Fatalf(err)
    return
}

for {
    res, readErr := client.Read()
    if readErr != nil && readErr == mnet.ErrNoDataYet {
        continue
    }

    // do stuff with data
}
```

