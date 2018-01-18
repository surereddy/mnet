package msocks_test

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/influx6/mnet"
	"github.com/influx6/mnet/msocks"
)

var (
	defaultClientSize = 32768
)

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

func Benchmark4BytesMessages(b *testing.B) {
	b.StopTimer()
	payload := sizedPayload(4)
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

func Benchmark16KMessages(b *testing.B) {
	b.StopTimer()
	payload := sizedPayload(16 * 1024)
	benchThis(b, payload)
}

func benchThis(b *testing.B, payload []byte) {
	b.StopTimer()
	b.ReportAllocs()

	payloadLen := len(payload)
	ctx, cancel := context.WithCancel(context.Background())
	netw, err := createBenchmarkNetwork(ctx, "localhost:5050")
	if err != nil {
		b.Fatalf("Failed to create network: %+q", err)
		return
	}

	netw.MaxWriteSize = defaultClientSize

	client, err := msocks.Connect("localhost:5050", msocks.Metrics(events), msocks.MaxBuffer(defaultClientSize))
	if err != nil {
		b.Fatalf("Failed to dial network %+q", err)
		return
	}

	b.SetBytes(int64(payloadLen))
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		if w, err := client.Write(payloadLen); err == nil {
			w.Write(payload)
			w.Close()
		}
	}

	client.Flush()

	b.StopTimer()
	client.Close()
	cancel()
	netw.Wait()
}

func createBenchmarkNetwork(ctx context.Context, addr string) (*msocks.Network, error) {
	var netw msocks.Network
	netw.Addr = addr
	netw.Metrics = events
	netw.MaxDeadline = 3 * time.Second
	netw.Handler = func(client mnet.Client) error {
		for {
			if _, err := client.Read(); err != nil {
				if err == mnet.ErrNoDataYet {
					continue
				}
				return err
			}
		}
	}

	return &netw, netw.Start(ctx)
}

var pub = []byte("pub  ")
var ch = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@$#%^&*()")

func sizedPayloadString(sz int) string {
	return string(sizedPayload(sz))
}

func sizedPayload(sz int) []byte {
	payload := make([]byte, len(pub)+2+sz)
	n := copy(payload, pub)
	n += copy(payload[n:], sizedBytes(sz))
	n += copy(payload, []byte("\r\n"))
	return payload[:n]
}

func sizedBytes(sz int) []byte {
	if sz <= 0 {
		return []byte("")
	}

	b := make([]byte, sz)
	for i := range b {
		b[i] = ch[rand.Intn(len(ch))]
	}
	return b
}

func sizedString(sz int) string {
	return string(sizedBytes(sz))
}
