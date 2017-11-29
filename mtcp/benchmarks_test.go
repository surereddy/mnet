package mtcp_test

import (
	"math/rand"
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

func BenchmarkNoBytesMessages(b *testing.B) {
	b.StopTimer()
	payload := sizedPayload(0)
	benchThis(b, payload)
}

func Benchmark2BytesMessages(b *testing.B) {
	b.StopTimer()
	payload := sizedPayload(2)
	benchThis(b, payload)
}

func Benchmark8BytesMessages(b *testing.B) {
	b.StopTimer()
	payload := sizedPayload(8)
	benchThis(b, payload)
}

func Benchmark16BytesMessages(b *testing.B) {
	b.StopTimer()
	payload := sizedPayload(16)
	benchThis(b, payload)
}

func Benchmark32BytesMessages(b *testing.B) {
	b.StopTimer()
	payload := sizedPayload(32)
	benchThis(b, payload)
}

func Benchmark64BytesMessages(b *testing.B) {
	b.StopTimer()
	payload := sizedPayload(64)
	benchThis(b, payload)
}

func Benchmark128BytesMessages(b *testing.B) {
	b.StopTimer()
	payload := sizedPayload(128)
	benchThis(b, payload)
}

func Benchmark256BytesMessages(b *testing.B) {
	b.StopTimer()
	payload := sizedPayload(256)
	benchThis(b, payload)
}

func Benchmark1KMessages(b *testing.B) {
	b.StopTimer()
	payload := sizedPayload(1024)
	benchThis(b, payload)
}

func Benchmark4KMessages(b *testing.B) {
	b.StopTimer()
	payload := sizedPayload(4 * 1024)
	benchThis(b, payload)
}

func Benchmark8KMessages(b *testing.B) {
	b.StopTimer()
	payload := sizedPayload(8 * 1024)
	benchThis(b, payload)
}

func benchThis(b *testing.B, payload []byte) {
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

var pub = []byte("pub")
var space = []byte(" ")
var ch = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@$#%^&*()")

func sizedPayloadString(sz int) string {
	return string(sizedPayload(sz))
}

func sizedPayload(sz int) []byte {
	payload := make(sz + len(pub) + 1)
	copy(payload, pub)
	copy(payload, space)
	copy(payload, sizedBytes(sz))
	return payload
}

func sizedBytes(sz int) []byte {
	b := make([]byte, sz)
	for i := range b {
		b[i] = ch[rand.Intn(len(ch))]
	}
	return b
}

func sizedString(sz int) string {
	return string(sizedBytes(sz))
}
