MTCP
-------
MTCP implements a tcp network server and client library for the `mnet` package to provide superfast reads and writes. Mtcp guarantees double digits nanoseconds transfer regardless of data size.


## Implemented

Mtcp implements all methods required by the `mnet.Client` type except the following:

- `mnet.Client.WriteTo`
- `mnet.Client.ReadFrom`
- `mnet.Client.FlushAddr`

Hence when such methods are called, error will be returned.

## Benchmarks

Below are recently runned benchmarks, see [BenchmarkTests](./benchmark.txt) for older runs.

```bash
BenchmarkNoBytesMessages-4    	30000000	        47.1 ns/op	 127.41 MB/s	       0 B/op	       0 allocs/op
Benchmark2BytesMessages-4     	30000000	        42.8 ns/op	 186.91 MB/s	       0 B/op	       0 allocs/op
Benchmark4BytesMessages-4     	30000000	        42.7 ns/op	 234.43 MB/s	       0 B/op	       0 allocs/op
Benchmark8BytesMessages-4     	30000000	        43.1 ns/op	 324.56 MB/s	       0 B/op	       0 allocs/op
Benchmark16BytesMessages-4    	30000000	        42.4 ns/op	 519.47 MB/s	       0 B/op	       0 allocs/op
Benchmark32BytesMessages-4    	30000000	        41.7 ns/op	 911.93 MB/s	       0 B/op	       0 allocs/op
Benchmark64BytesMessages-4    	50000000	        42.2 ns/op	1659.15 MB/s	       0 B/op	       0 allocs/op
Benchmark128BytesMessages-4   	30000000	        41.2 ns/op	3252.38 MB/s	       0 B/op	       0 allocs/op
Benchmark256BytesMessages-4   	30000000	        44.2 ns/op	5927.46 MB/s	       0 B/op	       0 allocs/op
Benchmark1KMessages-4         	30000000	        51.3 ns/op	20085.31 MB/s	       0 B/op	       0 allocs/op
Benchmark4KMessages-4         	30000000	        47.1 ns/op	87023.23 MB/s	       0 B/op	       0 allocs/op
Benchmark8KMessages-4         	30000000	        45.7 ns/op	179226.61 MB/s	       0 B/op	       0 allocs/op
Benchmark16KMessages-4        	30000000	        45.7 ns/op	358461.13 MB/s	       0 B/op	       0 allocs/op

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

		// Get writer with 5byte space for response.
		if writer, err := client.Write(5); err == nil {
			writer.Write("welcome")
			writer.Close()
		}
		
		client.Flush()
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