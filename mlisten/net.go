// Package mlisten implements function composition for net.Listeners and net.Conns providing functional compositions
// where behaviours are hidden within functions and reducing alot of deadlocks.
package mlisten

import (
	"crypto/tls"
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"sync/atomic"

	"github.com/influx6/faux/netutils"
)

const (
	// TLSHandshakeTTL defines time to set Read deadline for TLsHandshake.
	TLSHandshakeTTL = 700 * time.Millisecond

	// MaxTLSHandshakeTTLWait defines time to wait before we end connection because
	// tls handshake failed to happen in expected time.
	MaxTLSHandshakeTTLWait = 2 * time.Second
)

// errors ...
var (
	ErrListenerClosed = errors.New("listener has being closed")
)

// ConnReader defines reader for net.Conn type.
type ConnReadWriteCloser interface {
	io.Closer
	Addr() net.Addr
	WriteConn(net.Conn) error
	ReadConn() (net.Conn, error)
}

// Listen returns a Readr and Writer for reading net.Conns from a underline net.Listener.
func Listen(protocol string, addr string, config *tls.Config) (ConnReadWriteCloser, error) {
	lt, err := netutils.MakeListener(protocol, addr, config)
	if err != nil {
		return nil, err
	}

	if tlt, ok := lt.(*net.TCPListener); ok {
		lt = netutils.NewKeepAliveListener(tlt)
	}

	readWriter := new(connReadWriter)
	readWriter.addr = lt.Addr()
	readWriter.tls = config
	readWriter.l = lt

	return readWriter, nil
}

type connReadWriter struct {
	addr net.Addr
	ml   sync.Mutex
	l    net.Listener
	tls  *tls.Config
}

func (cs *connReadWriter) Addr() net.Addr {
	return cs.addr
}

// WriteConn receives the provided net.Conn and closes the connection, this
// assumes all operation with the net.Conn has being complete and the resource
// and connection should end here.
func (cs *connReadWriter) WriteConn(conn net.Conn) error {
	return conn.Close()
}

// Close closes the underneath net.Listener, ending all
func (cs *connReadWriter) Close() error {
	cs.ml.Lock()
	if cs.l == nil {
		cs.ml.Unlock()
		return ErrListenerClosed
	}
	listener := cs.l
	cs.ml.Unlock()

	err := listener.Close()
	cs.ml.Lock()
	cs.l = nil
	cs.ml.Unlock()

	return err
}

// ReadConn returns a new net.Conn from the underying listener.
func (cs *connReadWriter) ReadConn() (net.Conn, error) {
	cs.ml.Lock()
	if cs.l == nil {
		cs.ml.Unlock()
		return nil, ErrListenerClosed
	}
	listener := cs.l
	cs.ml.Unlock()

	newConn, err := listener.Accept()
	if err != nil {
		if ne, ok := err.(*net.OpError); ok && !ne.Temporary() {
			cs.Close()
			return nil, ErrListenerClosed
		}
		return nil, err
	}

	// if we are not using tls then continue.
	if cs.tls == nil {
		return newConn, nil
	}

	tlsConn, ok := newConn.(*tls.Conn)
	if !ok {
		tlsConn = tls.Server(newConn, cs.tls)
	}

	var tlsHandshaked int64

	// If we pass over this time and we have not being sorted then kill connection.
	time.AfterFunc(MaxTLSHandshakeTTLWait, func() {
		if atomic.LoadInt64(&tlsHandshaked) != 1 {
			tlsConn.SetReadDeadline(time.Time{})
			tlsConn.Close()
		}
	})

	tlsConn.SetReadDeadline(time.Now().Add(TLSHandshakeTTL))
	if err := tlsConn.Handshake(); err != nil {
		atomic.StoreInt64(&tlsHandshaked, 1)
		tlsConn.Close()
		return nil, err
	}

	atomic.StoreInt64(&tlsHandshaked, 1)
	tlsConn.SetReadDeadline(time.Time{})

	return tlsConn, nil
}
