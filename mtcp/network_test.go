package mtcp_test

import (
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/influx6/faux/context"
	"github.com/influx6/faux/metrics"
	"github.com/influx6/faux/metrics/custom"
	"github.com/influx6/faux/tests"
	"github.com/influx6/mnet"
	"github.com/influx6/mnet/certificates"
	"github.com/influx6/mnet/mtcp"
)

var (
	events metrics.Metrics
	dialer = &net.Dialer{Timeout: 2 * time.Second}
)

func initMetrics() {
	if testing.Verbose() {
		events = metrics.New(custom.StackDisplay(os.Stderr))
	}
}

func TestNonTLSNetworkWithNetConn(t *testing.T) {
	initMetrics()

	ctx := context.New()
	netw, err := createNewNetwork(ctx, "localhost:4050", nil)
	if err != nil {
		tests.FailedWithError(err, "Should have successfully create network")
	}
	tests.Passed("Should have successfully create network")

	conn, err := dialer.Dial("tcp", "localhost:4050")
	if err != nil {
		tests.FailedWithError(err, "Should have successfully connected to network")
	}
	tests.Passed("Should have successfully connected to network")

	payload := makeMessage([]byte("pub help"))
	if _, err := conn.Write(payload); err != nil {
		tests.FailedWithError(err, "Should have delivered message to network as client")
	}
	tests.Passed("Should have delivered message to network as client")

	expected := []byte("now publishing to [help]\r\n")
	received, err := readMessage(conn)
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
	ctx.Cancel()

	netw.Wait()
}
func TestTLSNetworkWithNetConn(t *testing.T) {
	initMetrics()

	_, server, client, err := createTLSCA()
	if err != nil {
		tests.FailedWithError(err, "Should have successfully created server and client certs")
	}
	tests.Passed("Should have successfully created server and client certs")

	serverTls, err := server.TLSServerConfig(true)
	if err != nil {
		tests.FailedWithError(err, "Should have successfully create sever's tls config")
	}
	tests.Passed("Should have successfully create sever's tls config")

	clientTls, err := client.TLSClientConfig()
	if err != nil {
		tests.FailedWithError(err, "Should have successfully create sever's tls config")
	}
	tests.Passed("Should have successfully create sever's tls config")

	clientTls.ServerName = "localhost"

	ctx := context.New()
	netw, err := createNewNetwork(ctx, "localhost:4050", serverTls)
	if err != nil {
		tests.FailedWithError(err, "Should have successfully create network")
	}
	tests.Passed("Should have successfully create network")

	conn, err := tls.DialWithDialer(dialer, "tcp", "localhost:4050", clientTls)
	if err != nil {
		tests.FailedWithError(err, "Should have successfully connected to network")
	}
	tests.Passed("Should have successfully connected to network")

	if err := conn.Handshake(); err != nil {
		tests.FailedWithError(err, "Should have successfully Handshaked tls connection")
	}
	tests.Passed("Should have successfully Handshaked tls connection")

	payload := makeMessage([]byte("pub help"))
	if _, err := conn.Write(payload); err != nil {
		tests.FailedWithError(err, "Should have delivered message to network as client")
	}
	tests.Passed("Should have delivered message to network as client")

	expected := []byte("now publishing to [help]\r\n")
	received, err := readMessage(conn)
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
	ctx.Cancel()

	netw.Wait()
}

func readMessage(conn net.Conn) ([]byte, error) {
	incoming := make([]byte, 2)
	_, err := conn.Read(incoming)
	if err != nil {
		return nil, err
	}

	expectedLen := binary.BigEndian.Uint16(incoming)
	data := make([]byte, expectedLen)
	_, err = conn.Read(data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func makeMessage(msg []byte) []byte {
	header := make([]byte, 2, len(msg)+2)
	binary.BigEndian.PutUint16(header, uint16(len(msg)))
	header = append(header, msg...)
	return header
}

func createTLSCA() (ca certificates.CertificateAuthority, server, client certificates.CertificateRequest, err error) {
	serials := certificates.SerialService{Length: 128}
	profile := certificates.CertificateProfile{
		CommonName:   "*",
		Local:        "Lagos",
		Organization: "DreamBench",
		Country:      "Nigeria",
		Province:     "South-West",
	}

	var service certificates.CertificateAuthorityService
	service.KeyStrength = 4096
	service.LifeTime = (time.Hour * 8760)
	service.Profile = profile
	service.Serials = serials
	service.Emails = append([]string{}, "alex.ewetumo@dreambench.io")

	var requestService certificates.CertificateRequestService
	requestService.Profile = profile
	requestService.KeyStrength = 2048

	ca, err = service.New()
	if err != nil {
		return
	}

	if server, err = requestService.New(); err == nil {
		if err = ca.ApproveServerCertificateSigningRequest(&server, serials, time.Hour*8760); err != nil {
			return
		}
	}

	if client, err = requestService.New(); err == nil {
		if err = ca.ApproveClientCertificateSigningRequest(&client, serials, time.Hour*8760); err != nil {
			return
		}
	}

	return
}

func createNewNetwork(ctx context.CancelContext, addr string, config *tls.Config) (*mtcp.Network, error) {
	var netw mtcp.Network
	netw.Addr = addr
	netw.Metrics = events
	netw.TLS = config
	netw.ClientMaxWriteDeadline = 1 * time.Second

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

			switch command {
			case "pub":
				client.Write([]byte(fmt.Sprintf("now publishing to %+s\r\n", rest)))
			case "sub":
				client.Write([]byte(fmt.Sprintf("subscribed to %+s\r\n", rest)))
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
