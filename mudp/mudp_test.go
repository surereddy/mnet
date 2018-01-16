package mudp_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/influx6/faux/metrics"
	"github.com/influx6/faux/metrics/custom"
	"github.com/influx6/faux/tests"
	"github.com/influx6/mnet"
	"github.com/influx6/mnet/mudp"
)

var (
	events = metrics.New()
	dialer = &net.Dialer{Timeout: 2 * time.Second}
)

func init() {
	if testing.Verbose() {
		events = metrics.New(custom.StackDisplay(os.Stderr))
	}
}

func TestUDPServerWithNetConn(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	netw, err := createNewNetwork(ctx, "localhost:4050")
	if err != nil {
		tests.FailedWithError(err, "Should have successfully create network")
	}
	tests.Passed("Should have successfully create network")

	conn, err := dialer.Dial("udp", "localhost:4050")
	if err != nil {
		tests.FailedWithError(err, "Should have successfully connected to network")
	}
	tests.Passed("Should have successfully connected to network")

	udpconn, ok := conn.(*net.UDPConn)
	if !ok {
		tests.Failed("Should have gotten net.UDPConn type from dialer")
	}
	tests.Passed("Should have gotten net.UDPConn type from dialer")

	payload := makeMessage([]byte("pub help"))
	if _, err := conn.Write(payload); err != nil {
		tests.FailedWithError(err, "Should have delivered message to network as client")
	}
	tests.Passed("Should have delivered message to network as client")

	expected := []byte("now publishing to [help]\r\n")
	received, _, err := readMessage(udpconn)
	if err != nil {
		tests.FailedWithError(err, "Should have successfully read message from network")
	}
	tests.Passed("Should have successfully read message from network")

	if !bytes.Equal(received, expected) {
		tests.Info("Received: %+q", received)
		tests.Info("Expected: %+q", expected)
		tests.FailedWithError(err, "Should have successfully matched expected data with received from network")
	}
	tests.Passed("Should have successfully matched expected data with received from network")

	conn.Close()
	cancel()

	netw.Wait()
}

func TestUDPServerWithMUDPClient(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	netw, err := createNewNetwork(ctx, "localhost:4050")
	if err != nil {
		tests.FailedWithError(err, "Should have successfully create network")
	}
	tests.Passed("Should have successfully create network")

	client, err := mudp.Connect("localhost:4050", mudp.Metrics(events))
	if err != nil {
		tests.FailedWithError(err, "Should have successfully connected to network")
	}
	tests.Passed("Should have successfully connected to network")

	payload := []byte("pub help")
	cw, err := client.Write(len(payload))
	if err != nil {
		tests.FailedWithError(err, "Should have successfully created new writer")
	}
	tests.Passed("Should have successfully created new writer")

	cw.Write(payload)
	if err := cw.Close(); err != nil {
		tests.FailedWithError(err, "Should have successfully written payload to client")
	}
	tests.Passed("Should have successfully written payload to client")

	if ferr := client.Flush(); ferr != nil {
		tests.FailedWithError(ferr, "Should have successfully flush data to network")
	}
	tests.Passed("Should have successfully flush data to network")

	var res []byte
	var readErr error
	for {
		res, readErr = client.Read()
		if readErr != nil && readErr == mnet.ErrNoDataYet {
			continue
		}

		break
	}

	if readErr != nil {
		tests.FailedWithError(readErr, "Should have successfully read reply from network")
	}
	tests.Passed("Should have successfully read reply from network")

	expected := []byte("now publishing to [help]\r\n")
	if !bytes.Equal(res, expected) {
		tests.Info("Received: %+q", res)
		tests.Info("Expected: %+q", expected)
		tests.FailedWithError(err, "Should have successfully matched expected data with received from network")
	}
	tests.Passed("Should have successfully matched expected data with received from network")

	if cerr := client.Close(); cerr != nil {
		tests.FailedWithError(cerr, "Should have successfully closed client connection")
	}
	tests.Passed("Should have successfully closed client connection")

	cancel()
	netw.Wait()
}

func createNewNetwork(ctx context.Context, addr string) (*mudp.Network, error) {
	var netw mudp.Network
	netw.Addr = addr
	netw.Network = "udp"
	netw.Metrics = events
	netw.MaxWriteDeadline = 3 * time.Second

	netw.Handler = func(client mnet.Client) error {
		for {
			message, err := client.Read()
			if err != nil {
				if err == mnet.ErrNoDataYet {
					time.Sleep(300 * time.Millisecond)
					continue
				}

				return err
			}

			messages := strings.Split(string(message), " ")
			if len(messages) == 0 {
				continue
			}

			command := messages[0]
			rest := messages[1:]
			tests.Info("UDP Server received: %q -> %+q", command, rest)

			switch command {
			case "pub":
				res := []byte(fmt.Sprintf("now publishing to %+s\r\n", rest))
				w, err := client.Write(len(res))
				if err != nil {
					return err
				}

				w.Write(res)
				w.Close()
			case "sub":
				res := []byte(fmt.Sprintf("subscribed to %+s\r\n", rest))
				w, err := client.Write(len(res))
				if err != nil {
					return err
				}

				w.Write(res)
				w.Close()
			}

			if err := client.Flush(); err != nil {
				if err == io.ErrShortWrite {
					continue
				}

				return err
			}
		}
	}

	return &netw, netw.Start(ctx)
}

func makeMessage(msg []byte) []byte {
	header := make([]byte, 4, len(msg)+4)
	binary.BigEndian.PutUint32(header, uint32(len(msg)))
	header = append(header, msg...)
	return header
}

func readMessage(conn *net.UDPConn) ([]byte, net.Addr, error) {
	incoming := make([]byte, 128)
	n, addr, err := conn.ReadFrom(incoming)
	if err != nil {
		return nil, nil, err
	}

	incoming = incoming[:n]

	expectedLen := int(binary.BigEndian.Uint32(incoming[:mnet.HeaderLength]))
	data := incoming[mnet.HeaderLength:]
	if len(data) != expectedLen {
		return nil, addr, errors.New("expected length unmarched")
	}

	return data, addr, nil
}
