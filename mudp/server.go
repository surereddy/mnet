package mudp

import (
	"context"
	"crypto/tls"
	"net"
	"time"

	"github.com/apex/apex/metrics"
	"github.com/influx6/mnet"
)

// Network defines a network which runs ontop of provided mnet.ConnHandler.
type Network struct {
	ID         string
	Addr       string
	ServerName string
	TLS        *tls.Config
	Handler    mnet.ConnHandler
	Metrics    metrics.Metrics

	// MaxWriterSize sets the maximum size of bufio.Writer for
	// network connection.
	MaxWriterSize int

	// MaxWriterDeadline sets deadline to be enforced when writing
	// to network.
	MaxWriteDeadline time.Duration

	totalClients int64
	totalClosed  int64
	totalActive  int64
	totalOpened  int64

	conn *net.UDPConn
	addr *net.UDPAddr
	rung sync.WorkGroup
}

// Start boots up the server and initializes all internals to make
// itself ready for servicing requests.
func (n *Network) Start(ctx context.Context) error {
	return nil
}

func (n *Network) handleConnections(ctx context.Context) {

}

func (n *Network) handleCloseConnections(ctx context.Context) {

}
