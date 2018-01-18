MSOCKS
-------
MSOCKS implements a websocket network server and client library for the `mnet` package to provide superfast reads and writes.

## Benchmarks

Below are recently runned benchmarks, see [BenchmarkTests](./benchmark.txt) for older runs.

```bash
goos: darwin
goarch: amd64
pkg: github.com/influx6/mnet/msocks
BenchmarkNoBytesMessages-4    	50000000	        26.3 ns/op	 265.70 MB/s	       0 B/op	       0 allocs/op
Benchmark2BytesMessages-4     	50000000	        26.5 ns/op	 340.20 MB/s	       0 B/op	       0 allocs/op
Benchmark4BytesMessages-4     	50000000	        25.7 ns/op	 427.40 MB/s	       0 B/op	       0 allocs/op
Benchmark8BytesMessages-4     	50000000	        26.9 ns/op	 558.56 MB/s	       0 B/op	       0 allocs/op
Benchmark16BytesMessages-4    	50000000	        26.3 ns/op	 873.85 MB/s	       0 B/op	       0 allocs/op
Benchmark32BytesMessages-4    	50000000	        25.9 ns/op	1503.12 MB/s	       0 B/op	       0 allocs/op
Benchmark64BytesMessages-4    	50000000	        25.9 ns/op	2744.21 MB/s	       0 B/op	       0 allocs/op
Benchmark128BytesMessages-4   	50000000	        26.0 ns/op	5194.63 MB/s	       0 B/op	       0 allocs/op
Benchmark256BytesMessages-4   	50000000	        25.5 ns/op	10314.74 MB/s	       0 B/op	       0 allocs/op
Benchmark1KMessages-4         	50000000	        25.1 ns/op	41009.28 MB/s	       0 B/op	       0 allocs/op
Benchmark4KMessages-4         	50000000	        26.3 ns/op	156042.32 MB/s	       0 B/op	       0 allocs/op
Benchmark8KMessages-4         	50000000	        27.0 ns/op	304163.43 MB/s	       0 B/op	       0 allocs/op
Benchmark16KMessages-4        	50000000	        25.7 ns/op	636983.55 MB/s	       0 B/op	       0 allocs/op
PASS
ok  	github.com/influx6/mnet/msocks	18.466s

```

## Examples

- MSOCKS Server
`msocks.Network` provides a websockets network server which readily through a handler function allows handling incoming client connections, as below: 

```go
var netw msocks.Network
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
`msocks.Connect` provides a websockets client which connects to a `msocks.Network` server, ready to allow user's communication at blazing speeds.

```go
client, err := msocks.Connect("localhost:4050")
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
