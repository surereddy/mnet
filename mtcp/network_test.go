package mtcp_test

import (
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/influx6/faux/context"
	"github.com/influx6/faux/metrics"
	"github.com/influx6/faux/metrics/custom"
	"github.com/influx6/faux/tests"
	"github.com/influx6/mnet"
	"github.com/influx6/mnet/certificates"
	"github.com/influx6/mnet/mocks"
	"github.com/influx6/mnet/mtcp"
)

var (
	events metrics.Metrics
	space  = regexp.MustCompile("\\s")
)

func initMetrics() {
	if testing.Verbose() {
		events = metrics.New(custom.StackDisplay(os.Stderr))
	}
}

func TestNonTLSNetworkWithNetConn(t *testing.T) {
	initMetrics()

	ctx := context.New()
	netw, err := createNewNetwork(ctx, ":4050", nil)
	if err != nil {
		tests.FailedWithError(err, "Should have successfully create network")
	}
	tests.Passed("Should have successfully create network")

	conn, err := net.DialTimeout("tcp", ":4050", 2*time.Second)
	if err != nil {
		tests.FailedWithError(err, "Should have successfully connected to network")
	}
	tests.Passed("Should have successfully connected to network")

	payload := []byte("pub help")
	if err := writeMessage(conn, payload); err != nil {
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

func writeMessage(w io.Writer, msg []byte) error {
	header := make([]byte, 2, len(msg)+2)
	binary.BigEndian.PutUint16(header, uint16(len(msg)))
	header = append(header, msg...)
	_, err := w.Write(header)
	return err
}

func createTLSCA() (ca certificates.CertificateAuthority, server, client certificates.CertificateRequest, err error) {
	var store mocks.PersistenceStoreMock
	store.GetFunc, store.PersistFunc = mocks.MapStore(make(map[string][]byte))

	serials := certificates.SerialService{Length: 128}
	profile := certificates.CertificateProfile{
		Local:        "Lagos",
		Organization: "DreamBench",
		CommonName:   "DreamBench Inc",
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
