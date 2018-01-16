MUDP
-------
MUDP implements a udp network server and client library for the `mnet` package to provide superfast reads and writes.

## Benchmarks

Below are recently runned benchmarks, see [BenchmarkTests](./benchmark.txt) for older runs.

```bash
goos: darwin
goarch: amd64
pkg: github.com/influx6/mnet/mudp
BenchmarkNoBytesMessages-4    	 3000000	       555 ns/op	  10.79 MB/s	      80 B/op	       2 allocs/op
Benchmark2BytesMessages-4     	 3000000	       565 ns/op	  14.15 MB/s	      80 B/op	       2 allocs/op
Benchmark4BytesMessages-4     	 3000000	       552 ns/op	  18.11 MB/s	      80 B/op	       2 allocs/op
Benchmark8BytesMessages-4     	 3000000	       566 ns/op	  24.70 MB/s	      80 B/op	       2 allocs/op
Benchmark16BytesMessages-4    	 3000000	       576 ns/op	  38.17 MB/s	      80 B/op	       2 allocs/op
Benchmark32BytesMessages-4    	 2000000	       542 ns/op	  70.03 MB/s	      80 B/op	       2 allocs/op
Benchmark64BytesMessages-4    	 3000000	       582 ns/op	 120.22 MB/s	      80 B/op	       2 allocs/op
Benchmark128BytesMessages-4   	 3000000	       544 ns/op	 246.13 MB/s	      80 B/op	       2 allocs/op
Benchmark256BytesMessages-4   	 3000000	       597 ns/op	 438.74 MB/s	      80 B/op	       2 allocs/op
Benchmark1KMessages-4         	 2000000	       589 ns/op	1746.58 MB/s	      80 B/op	       2 allocs/op
Benchmark4KMessages-4         	 2000000	       751 ns/op	5456.55 MB/s	      80 B/op	       2 allocs/op
Benchmark8KMessages-4         	 2000000	       809 ns/op	10129.31 MB/s	      80 B/op	       2 allocs/op
Benchmark16KMessages-4        	 1000000	      1415 ns/op	11582.20 MB/s	      80 B/op	       2 allocs/op
PASS
ok  	github.com/influx6/mnet/mudp	28.212s

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

