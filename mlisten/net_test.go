package mlisten_test

import (
	"crypto/tls"
	"net"
	"testing"
	"time"

	"github.com/influx6/faux/tests"
	"github.com/influx6/mnet/mlisten"
)

var rsaCertPEM = `-----BEGIN CERTIFICATE-----
MIIB0zCCAX2gAwIBAgIJAI/M7BYjwB+uMA0GCSqGSIb3DQEBBQUAMEUxCzAJBgNV
BAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEwHwYDVQQKDBhJbnRlcm5ldCBX
aWRnaXRzIFB0eSBMdGQwHhcNMTIwOTEyMjE1MjAyWhcNMTUwOTEyMjE1MjAyWjBF
MQswCQYDVQQGEwJBVTETMBEGA1UECAwKU29tZS1TdGF0ZTEhMB8GA1UECgwYSW50
ZXJuZXQgV2lkZ2l0cyBQdHkgTHRkMFwwDQYJKoZIhvcNAQEBBQADSwAwSAJBANLJ
hPHhITqQbPklG3ibCVxwGMRfp/v4XqhfdQHdcVfHap6NQ5Wok/4xIA+ui35/MmNa
rtNuC+BdZ1tMuVCPFZcCAwEAAaNQME4wHQYDVR0OBBYEFJvKs8RfJaXTH08W+SGv
zQyKn0H8MB8GA1UdIwQYMBaAFJvKs8RfJaXTH08W+SGvzQyKn0H8MAwGA1UdEwQF
MAMBAf8wDQYJKoZIhvcNAQEFBQADQQBJlffJHybjDGxRMqaRmDhX0+6v02TUKZsW
r5QuVbpQhH6u+0UgcW0jp9QwpxoPTLTWGXEWBBBurxFwiCBhkQ+V
-----END CERTIFICATE-----
`

var rsaKeyPEM = `-----BEGIN RSA PRIVATE KEY-----
MIIBOwIBAAJBANLJhPHhITqQbPklG3ibCVxwGMRfp/v4XqhfdQHdcVfHap6NQ5Wo
k/4xIA+ui35/MmNartNuC+BdZ1tMuVCPFZcCAwEAAQJAEJ2N+zsR0Xn8/Q6twa4G
6OB1M1WO+k+ztnX/1SvNeWu8D6GImtupLTYgjZcHufykj09jiHmjHx8u8ZZB/o1N
MQIhAPW+eyZo7ay3lMz1V01WVjNKK9QSn1MJlb06h/LuYv9FAiEA25WPedKgVyCW
SmUwbPw8fnTcpqDWE3yTO3vKcebqMSsCIBF3UmVue8YU3jybC3NxuXq3wNm34R8T
xVLHwDXh/6NJAiEAl2oHGGLz64BuAfjKrqwz7qMYr9HCLIe/YsoWq/olzScCIQDi
D2lWusoe2/nEqfDVVWGWlyJ7yOmqaVm/iNUN9B2N2g==
-----END RSA PRIVATE KEY-----
`

func TestListen(t *testing.T) {
	reader, err := mlisten.Listen("tcp", ":4050", nil)
	if err != nil {
		tests.FailedWithError(err, "Should have successfully created reader and writer")
	}
	tests.Passed("Should have successfully created reader and writer")

	go makeConn(":4050")
	conn, err := reader.ReadConn()
	if err != nil {
		tests.FailedWithError(err, "Should have sucessfully read connection")
	}
	tests.Passed("Should have sucessfully read connection")

	if err := reader.WriteConn(conn); err != nil {
		tests.FailedWithError(err, "Should have sucessfully written connection")
	}
	tests.Passed("Should have sucessfully written connection")

	if cerr := reader.Close(); cerr != nil {
		tests.ErroredWithError(err, "Should have sucessfully closed reader")
	}

	_, err = reader.ReadConn()
	if err == nil {
		tests.FailedWithError(err, "Should have sucessfully failed to read connection")
	}
	tests.Passed("Should have sucessfully failed to read connection")

	if err != mlisten.ErrListenerClosed {
		tests.FailedWithError(err, "Should have successfully matched close error")
	}
	tests.Passed("Should have successfully matched close error")
}

func makeConn(addr string) {
	conn, err := net.DialTimeout("tcp", addr, 30*time.Second)
	if err != nil {
		return
	}

	conn.Write([]byte("Hello"))
}

func TestListenWithTLS(t *testing.T) {
	cert, err := tls.X509KeyPair([]byte(rsaCertPEM), []byte(rsaKeyPEM))
	if err != nil {
		tests.ErroredWithError(err, "Should have successfully loaded rsa cert and key pair")
	}
	tests.Passed("Should have successfully loaded rsa cert and key pair")

	var config tls.Config
	config.MinVersion = tls.VersionTLS12
	config.InsecureSkipVerify = true
	config.Certificates = append(config.Certificates, cert)

	reader, err := mlisten.Listen("tcp", ":4050", &config)
	if err != nil {
		tests.FailedWithError(err, "Should have successfully created reader and writer")
	}
	tests.Passed("Should have successfully created reader and writer")

	go makeTLSConn(":4050", &config)
	conn, err := reader.ReadConn()
	if err != nil {
		tests.FailedWithError(err, "Should have sucessfully read connection")
	}
	tests.Passed("Should have sucessfully read connection")

	if err := reader.WriteConn(conn); err != nil {
		tests.FailedWithError(err, "Should have sucessfully written connection")
	}
	tests.Passed("Should have sucessfully written connection")

	if cerr := reader.Close(); cerr != nil {
		tests.ErroredWithError(err, "Should have sucessfully closed reader")
	}
}

func makeTLSConn(addr string, config *tls.Config) {
	conn, err := tls.Dial("tcp", addr, config)
	if err != nil {
		return
	}

	conn.Write([]byte("Hello"))
}
