package mtcp_test

import (
	"net"
	"testing"
	"time"

	"github.com/influx6/faux/context"
	"github.com/influx6/mnet/mtcp"
)

func BenchmarkNonTLSNetworkWriteWithClient(b *testing.B) {
	b.StopTimer()

	ctx := context.New()
	netw, err := createNewNetwork(ctx, ":5050", nil)
	if err != nil {
		b.Fatalf("Failed to create network: %+q", err)
		return
	}

	client, err := mtcp.Connect(":5050", mtcp.Metrics(events), mtcp.ClientWriteInterval(1*time.Second))
	if err != nil {
		b.Fatalf("Failed to dial network %+q", err)
		return
	}

	payload := []byte("pub help")

	b.StartTimer()
	b.SetBytes(int64(len(payload)))
	for i := 0; i < b.N; i++ {
		client.Write(payload)
	}

	client.Flush()
	b.StopTimer()
	client.Close()
	ctx.Cancel()
	netw.Wait()

}

func BenchmarkNonTLSNetworkWriteWithNetConn(b *testing.B) {
	b.StopTimer()

	ctx := context.New()
	netw, err := createNewNetwork(ctx, ":5050", nil)
	if err != nil {
		b.Fatalf("Failed to create network: %+q", err)
		return
	}

	conn, err := net.DialTimeout("tcp", ":5050", 2*time.Second)
	if err != nil {
		b.Fatalf("Failed to dial network: %+q", err)
		return
	}

	payload := []byte("pub help")

	b.StartTimer()
	b.SetBytes(int64(len(payload)))
	for i := 0; i < b.N; i++ {
		writeMessage(conn, payload)
		if i%100 == 0 {
			time.Sleep(1 * time.Nanosecond)
		}
	}

	b.StopTimer()
	conn.Close()
	ctx.Cancel()
	netw.Wait()

}
