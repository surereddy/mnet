MUDP
-------
MUDP implements a udp network server and client library for the `mnet` package to provide superfast reads and writes.

## Benchmarks

Below are recently runned benchmarks, see [BenchmarkTests](./benchmark.txt) for older runs.

```bash

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
        _, addr, err := client.ReadFrom()
        if err != nil {
            if err == mnet.ErrNoDataYet {
                time.Sleep(300 * time.Millisecond)
                continue
            }

            return err
        }

        writer.WriteTo("welcome", addr)
        writer.FlushAddr(addr)
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

- Target server using `RemoteAddress`

```go
client, err := mudp.Connect("localhost:4050")
if err != nil {
    log.Fatalf(err)
    return
}

// Get target/server address.
target, err := client.RemoteAddr()
if err != nil {
    log.Fatalf(err)
    return
}

	
// create writer by telling client size of data
// to be written and the address we wish to write to.
writer, err := client.WriteTo(10, target)
if err != nil {
    log.Fatalf(err)
    return
}

// write data desired for size.
writer.Write([]byte("pub help"))
if err := writer.Close(); err != nil {
    log.Fatalf(err)
    return
}

// flush all communication to address.
if _, err := client.FlushAddr(target); err != nil {
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
