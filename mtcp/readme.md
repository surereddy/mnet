MTCP
-------
MTCP implements a tcp network server and client library for the `mnet` package to provide superfast reads and writes.

## Benchmarks

Below are recently runned benchmarks, see [BenchmarkTests](./benchmark.txt) for older runs.

```bash
BenchmarkNoBytesMessages-4    	30000000	        40.5 ns/op	 148.28 MB/s	       0 B/op	       0 allocs/op
Benchmark2BytesMessages-4     	30000000	        41.4 ns/op	 193.24 MB/s	       0 B/op	       0 allocs/op
Benchmark4BytesMessages-4     	30000000	        40.6 ns/op	 246.18 MB/s	       0 B/op	       0 allocs/op
Benchmark8BytesMessages-4     	30000000	        41.2 ns/op	 340.04 MB/s	       0 B/op	       0 allocs/op
Benchmark16BytesMessages-4    	50000000	        40.8 ns/op	 539.56 MB/s	       0 B/op	       0 allocs/op
Benchmark32BytesMessages-4    	30000000	        41.8 ns/op	 909.35 MB/s	       0 B/op	       0 allocs/op
Benchmark64BytesMessages-4    	30000000	        40.6 ns/op	1725.10 MB/s	       0 B/op	       0 allocs/op
Benchmark128BytesMessages-4   	50000000	        40.9 ns/op	3273.82 MB/s	       0 B/op	       0 allocs/op
Benchmark256BytesMessages-4   	30000000	        40.5 ns/op	6474.85 MB/s	       0 B/op	       0 allocs/op
Benchmark1KMessages-4         	50000000	        41.1 ns/op	25060.10 MB/s	       0 B/op	       0 allocs/op
Benchmark4KMessages-4         	30000000	        42.0 ns/op	97627.75 MB/s	       0 B/op	       0 allocs/op
Benchmark8KMessages-4         	30000000	        41.3 ns/op	198691.27 MB/s	       0 B/op	       0 allocs/op

```

## Examples

- MTCP Server
`mtcp.Network` provides a tcp network server which readily through a handler function allows handling incoming client connections, as below: 

```go
var netw mtcp.Network
netw.TLS = config
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

- Client
`mtcp.Connect` provides a tcp client which connects to a `mtcp.Network` server, ready to allow user's communication at blazing speeds.

```go
client, err := mtcp.Connect("localhost:4050")
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