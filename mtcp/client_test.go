package mtcp_test

import (
	"bytes"
	"testing"

	"github.com/influx6/faux/context"
	"github.com/influx6/faux/tests"
	"github.com/influx6/mnet"
	"github.com/influx6/mnet/mtcp"
)

func TestNonTLSNetworkWithClient(t *testing.T) {
	initMetrics()

	ctx := context.New()
	netw, err := createNewNetwork(ctx, "localhost:4050", nil)
	if err != nil {
		tests.FailedWithError(err, "Should have successfully create network")
	}
	tests.Passed("Should have successfully create network")

	client, err := mtcp.Connect("localhost:4050", mtcp.Metrics(events))
	if err != nil {
		tests.FailedWithError(err, "Should have successfully connected to network")
	}
	tests.Passed("Should have successfully connected to network")

	payload := []byte("pub help")
	if _, err := client.Write(payload); err != nil {
		tests.FailedWithError(err, "Should have successfully written payload to client")
	}
	tests.Passed("Should have successfully written payload to client")

	if ferr := client.Flush(false); ferr != nil {
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

	ctx.Cancel()
	netw.Wait()
}

func TestTLSNetworkWithClient(t *testing.T) {
	initMetrics()

	_, serverca, clientca, err := createTLSCA()
	if err != nil {
		tests.FailedWithError(err, "Should have successfully created server and client certs")
	}
	tests.Passed("Should have successfully created server and client certs")

	serverTls, err := serverca.TLSServerConfig(true)
	if err != nil {
		tests.FailedWithError(err, "Should have successfully create sever's tls config")
	}
	tests.Passed("Should have successfully create sever's tls config")

	clientTls, err := clientca.TLSClientConfig()
	if err != nil {
		tests.FailedWithError(err, "Should have successfully create sever's tls config")
	}
	tests.Passed("Should have successfully create sever's tls config")

	ctx := context.New()
	netw, err := createNewNetwork(ctx, "localhost:4050", serverTls)
	if err != nil {
		tests.FailedWithError(err, "Should have successfully create network")
	}
	tests.Passed("Should have successfully create network")

	client, err := mtcp.Connect("localhost:4050", mtcp.Metrics(events), mtcp.TLSConfig(clientTls))
	if err != nil {
		tests.FailedWithError(err, "Should have successfully connected to network")
	}
	tests.Passed("Should have successfully connected to network")

	payload := []byte("pub help")
	if _, err := client.Write(payload); err != nil {
		tests.FailedWithError(err, "Should have successfully written payload to client")
	}
	tests.Passed("Should have successfully written payload to client")

	if ferr := client.Flush(false); ferr != nil {
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

	ctx.Cancel()
	netw.Wait()
}
