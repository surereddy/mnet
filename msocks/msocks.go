package msocks

import (
	"crypto/tls"
	"errors"
	"net"
	"strings"
	"sync/atomic"
	"time"

	"context"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/influx6/faux/metrics"
	"github.com/influx6/faux/netutils"
	"github.com/influx6/mnet"
	"github.com/influx6/mnet/internal"
	uuid "github.com/satori/go.uuid"
)

var (
	wsClientState = ws.StateClientSide
	wsReadBuffer  = 1024
	wsWriteBuffer = 1024
)

// errors ...
var (
	ErrNoTLSConfig = errors.New("no tls.Config provided")
)

// ConnectOptions defines a function type used to apply given
// changes to a *clientNetwork type
type ConnectOptions func(conn *socketClient)

// WriteInterval sets the clientNetwork to use the provided value
// as its write intervals for colasced/batch writing of send data.
func WriteInterval(dur time.Duration) ConnectOptions {
	return func(cm *socketClient) {
		cm.maxDeadline = dur
	}
}

// MaxBuffer sets the clientNetwork to use the provided value
// as its maximum buffer size for it's writer.
func MaxBuffer(buffer int) ConnectOptions {
	return func(cm *socketClient) {
		cm.maxWrite = buffer
	}
}

// Metrics sets the metrics instance to be used by the client for
// logging.
func Metrics(m metrics.Metrics) ConnectOptions {
	return func(cm *socketClient) {
		cm.metrics = m
	}
}

// TLSConfig sets the giving tls.Config to be used by the returned
// client.
func TLSConfig(config *tls.Config) ConnectOptions {
	return func(cm *socketClient) {
		cm.secure = true
		cm.tls = config
	}
}

// KeepAliveTimeout sets the client to use given timeout for it's connection net.Dialer
// keepAliveTimeout.
func KeepAliveTimeout(dur time.Duration) ConnectOptions {
	return func(cm *socketClient) {
		cm.keepTimeout = dur
	}
}

// Dialer sets the ws.Dialer to used creating a connection
// to the server.
func Dialer(dialer *ws.Dialer) ConnectOptions {
	return func(cm *socketClient) {
		cm.dialer = dialer
	}
}

// DialTimeout sets the client to use given timeout for it's connection net.Dialer
// dial timeout.
func DialTimeout(dur time.Duration) ConnectOptions {
	return func(cm *socketClient) {
		cm.dialTimeout = dur
	}
}

// NetworkID sets the id used by the client connection for identifying the
// associated network.
func NetworkID(id string) ConnectOptions {
	return func(cm *socketClient) {
		cm.nid = id
	}
}

// Connect is used to implement the client connection to connect to a
// mtcp.Network. It implements all the method functions required
// by the Client to communicate with the server. It understands
// the message length header sent along by every message and follows
// suite when sending to server.
func Connect(addr string, ops ...ConnectOptions) (mnet.Client, error) {
	var c mnet.Client
	c.ID = uuid.NewV4().String()

	addr = netutils.GetAddr(addr)
	host, _, _ := net.SplitHostPort(addr)

	network := new(socketClient)
	network.socketConn = new(socketConn)

	for _, op := range ops {
		op(network)
	}

	network.id = c.ID
	network.addr = addr
	network.hostname = host
	network.header = make([]byte, mnet.HeaderLength)
	if network.network == "" {
		network.network = "udp"
	}

	if network.metrics == nil {
		network.metrics = metrics.New()
	}

	if network.maxWrite <= 0 {
		network.maxWrite = mnet.MaxBufferSize
	}

	if network.maxDeadline <= 0 {
		network.maxDeadline = mnet.MaxFlushDeadline
	}

	if network.dialer == nil {
		network.dialer = &ws.Dialer{
			Timeout:         network.dialTimeout,
			ReadBufferSize:  wsReadBuffer,
			WriteBufferSize: wsWriteBuffer,
		}
	}

	network.parser = new(internal.TaggedMessages)

	c.NID = network.nid
	c.Metrics = network.metrics
	c.CloseFunc = network.close
	c.WriteFunc = network.write
	c.WriteFunc = network.write
	c.ReaderFunc = network.read
	c.FlushFunc = network.flush
	c.LiveFunc = network.isAlive
	c.StatisticFunc = network.getStatistics
	c.LiveFunc = network.isAlive
	c.LocalAddrFunc = network.getLocalAddr
	c.RemoteAddrFunc = network.getRemoteAddr
	c.ReconnectionFunc = network.reconnect

	if err := network.reconnect(c, addr); err != nil {
		return c, err
	}

	return c, nil
}

type socketClient struct {
	*socketConn
	addr        string
	hostname    string
	secure      bool
	tls         *tls.Config
	network     string
	dialer      *ws.Dialer
	keepTimeout time.Duration
	dialTimeout time.Duration
	started     int64
}

func (cn *socketClient) isStarted() bool {
	return atomic.LoadInt64(&cn.started) == 1
}

func (cn *socketClient) close(jn mnet.Client) error {
	if err := cn.isAlive(jn); err != nil {
		return mnet.ErrAlreadyClosed
	}

	err := cn.socketConn.close(jn)
	cn.waiter.Wait()
	return err
}

func (cn *socketClient) reconnect(jn mnet.Client, addr string) error {
	if err := cn.isAlive(jn); err == nil && cn.isStarted() {
		return nil
	}

	if !strings.HasPrefix(addr, "ws://") && !strings.HasPrefix(addr, "wss://") {
		addr = "ws://" + addr
	}

	if strings.HasPrefix(addr, "wss://") && cn.tls == nil {
		return ErrNoTLSConfig
	}

	defer atomic.StoreInt64(&cn.started, 1)

	cn.waiter.Wait()

	var conn net.Conn
	var err error

	if addr != "" {
		if conn, err = cn.getConn(addr); err != nil {
			conn, err = cn.getConn(cn.addr)
		}
	} else {
		conn, err = cn.getConn(cn.addr)
	}

	if err != nil {
		return err
	}

	cn.localAddr = conn.LocalAddr()
	cn.remoteAddr = conn.RemoteAddr()

	reader := wsutil.NewReader(conn, wsClientState)
	writer := wsutil.NewWriter(conn, wsClientState, ws.OpBinary)

	cn.cu.Lock()
	cn.conn = conn
	cn.cu.Unlock()

	cn.bu.Lock()
	cn.wsReader = reader
	cn.wsWriter = writer
	cn.bu.Unlock()

	cn.waiter.Add(1)
	go cn.readLoop(conn, reader)

	return nil
}

// getConn returns net.Conn for giving addr.
func (cn *socketClient) getConn(addr string) (net.Conn, error) {
	lastSleep := mnet.MinTemporarySleep

	var err error
	var conn net.Conn
	var hs ws.Handshake
	for {
		conn, _, hs, err = cn.dialer.Dial(context.Background(), addr)
		if err != nil {
			cn.metrics.Emit(
				metrics.Error(err),
				metrics.WithID(cn.id),
				metrics.With("type", "tcp"),
				metrics.With("addr", addr),
				metrics.With("network", cn.nid),
				metrics.Message("Connection: failed to connect"),
			)
			if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
				if lastSleep >= mnet.MaxTemporarySleep {
					return nil, err
				}

				time.Sleep(lastSleep)
				lastSleep *= 2
			}
			continue
		}
		break
	}

	cn.handshake = hs
	return conn, err
}
