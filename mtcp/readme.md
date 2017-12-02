MTCP
-------
MTCP implements a network server and client library for the `mnet` package to provide superfast reads and writes.

## Restrictions
Since `MTCP` expects data to be read or written to be in a format of `[DATASIZE][DATA]`, it is required that the user provides all writes within such a format, has the client and the server will 
processed this. This is the case to allow colasing multiple writes into a single write where each 
has a size header that can be read of to know it's individual message length.

*`MNet` provides the `SizeAppendWriter` built into the server and client connections which will append appropriate size header within unto provided data. Note, that such writers do have max sizes and must be aware of when using to collect data, has they do push once size is beyond max. This is configurable


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

client.Write([]byte("pub help"))
client.Flush()

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