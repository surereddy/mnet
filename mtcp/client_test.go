package mtcp_test

import (
	"bytes"
	"testing"

	"github.com/influx6/faux/context"
	"github.com/influx6/faux/tests"
	"github.com/influx6/mnet"
	"github.com/influx6/mnet/mtcp"
)

func TestClientConnectFunction(t *testing.T) {
	initMetrics()

	ctx := context.New()
	netw, err := createNewNetwork(ctx, ":4050", nil)
	if err != nil {
		tests.FailedWithError(err, "Should have successfully create network")
	}
	tests.Passed("Should have successfully create network")

	client, err := mtcp.Connect(":4050", mtcp.Metrics(events))
	if err != nil {
		tests.FailedWithError(err, "Should have successfully connected to network")
	}
	tests.Passed("Should have successfully connected to network")

	payload := []byte("pub help")
	if _, err := client.Write(payload); err != nil {
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
	if !bytes.Equal(received, expected) {
		tests.Info("Received: %+q", received)
		tests.Info("Expected: %+q", expected)
		tests.FailedWithError(err, "Should have successfully matched expected data with received from network")
	}
	tests.Passed("Should have successfully matched expected data with received from network")

	if cerr := client.Close(); cerr != nil {
		tests.FailedWithError(cerr, "Should have successfully closed client connection")
	}
	tests.Passed("Should have successfully closed client connection")
}
