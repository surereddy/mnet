MTCP
-------
MTCP implements a network server and client library for the `mnet` package to provide superfast reads and writes.

## Examples

- MTCP Server
`mtcp.Network` provides a tcp network server which readily receives connections and data within `[Size Header][Data]` format. It can be created as below:

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
`mtcp.Connect` provides a tcp client which connects to a `mtcp.Network` which readily receives data within `[Size Header][Data]` format and expects written data to follow such format as well. It can be created as below:

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