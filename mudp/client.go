package mudp

import (
	"errors"
	"net"
	"sync/atomic"
	"time"

	"bufio"

	"sync"

	"github.com/influx6/faux/metrics"
	"github.com/influx6/faux/netutils"
	"github.com/influx6/mnet"
	"github.com/influx6/mnet/internal"
	uuid "github.com/satori/go.uuid"
)

// errors ..
var (
	ErrExpectedUDPConn = errors.New("net.Conn returned is not a *net.UDPConn")
)

// ConnectOptions defines a function type used to apply given
// changes to a *clientNetwork type
type ConnectOptions func(conn *clientConn)

// WriteInterval sets the clientNetwork to use the provided value
// as its write intervals for colasced/batch writing of send data.
func WriteInterval(dur time.Duration) ConnectOptions {
	return func(cm *clientConn) {
		cm.maxDeadline = dur
	}
}

// MaxBuffer sets the clientNetwork to use the provided value
// as its maximum buffer size for it's writer.
func MaxBuffer(buffer int) ConnectOptions {
	return func(cm *clientConn) {
		cm.maxWrite = buffer
	}
}

// Metrics sets the metrics instance to be used by the client for
// logging.
func Metrics(m metrics.Metrics) ConnectOptions {
	return func(cm *clientConn) {
		cm.metrics = m
	}
}

// KeepAliveTimeout sets the client to use given timeout for it's connection net.Dialer
// keepAliveTimeout.
func KeepAliveTimeout(dur time.Duration) ConnectOptions {
	return func(cm *clientConn) {
		cm.keepTimeout = dur
	}
}

// DialTimeout sets the client to use given timeout for it's connection net.Dialer
// dial timeout.
func DialTimeout(dur time.Duration) ConnectOptions {
	return func(cm *clientConn) {
		cm.dialTimeout = dur
	}
}

// NetworkID sets the id used by the client connection for identifying the
// associated network.
func NetworkID(id string) ConnectOptions {
	return func(cm *clientConn) {
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
	//host, _, _ := net.SplitHostPort(addr)

	network := new(clientConn)
	network.netClient = new(netClient)

	for _, op := range ops {
		op(network)
	}

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
		network.dialer = &net.Dialer{
			Timeout:   network.dialTimeout,
			KeepAlive: network.keepTimeout,
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

// clientConn implements the client side udp connection client.
// It embeds the netClient and adds extra methods to provide client
// side behaviours.
type clientConn struct {
	*netClient
	network     string
	dialer      *net.Dialer
	waiter      sync.WaitGroup
	keepTimeout time.Duration
	dialTimeout time.Duration
	started     int64
}

func (cn *clientConn) isStarted() bool {
	return atomic.LoadInt64(&cn.started) == 1
}

func (cn *clientConn) close(jn mnet.Client) error {
	if err := cn.isAlive(jn); err != nil {
		return mnet.ErrAlreadyClosed
	}

	err := cn.netClient.close(jn)
	cn.waiter.Wait()
	return err
}

func (cn *clientConn) reconnect(jn mnet.Client, addr string) error {
	if err := cn.isAlive(jn); err == nil && cn.isStarted() {
		return nil
	}

	defer atomic.StoreInt64(&cn.started, 1)

	cn.waiter.Wait()

	if cn.remoteAddr == nil || (cn.remoteAddr != nil && addr != "") {
		raddr, err := net.ResolveUDPAddr(cn.network, addr)
		if err != nil {
			return err
		}

		cn.remoteAddr = raddr
	}

	conn, err := cn.getConn(jn, cn.remoteAddr)
	if err != nil {
		return err
	}

	cn.localAddr = conn.LocalAddr()
	cn.mainAddr = cn.localAddr

	cn.cu.Lock()
	cn.conn = conn
	cn.cu.Unlock()

	cn.bu.Lock()
	cn.buffer = bufio.NewWriterSize(targetConn{
		conn:   conn,
		client: true,
		target: cn.remoteAddr,
	}, cn.maxWrite)
	cn.bu.Unlock()

	cn.waiter.Add(1)
	go cn.readLoop(conn, jn)

	return nil
}

func (cn *clientConn) readLoop(conn *net.UDPConn, jn mnet.Client) {
	defer cn.close(jn)
	defer cn.waiter.Done()

	incoming := make([]byte, mnet.MinBufferSize, mnet.MaxBufferSize)
	for {
		n, addr, err := conn.ReadFrom(incoming)
		if err != nil {
			cn.metrics.Send(metrics.Entry{
				ID:      cn.id,
				Message: "Connection failed to read: closing",
				Level:   metrics.ErrorLvl,
				Field: metrics.Field{
					"err": err,
				},
			})
			return
		}

		// if nothing was read, skip.
		if n == 0 && len(incoming) == 0 {
			continue
		}

		if err := cn.handleMessage(incoming[:n], addr); err != nil {
			cn.metrics.Send(metrics.Entry{
				ID:      cn.id,
				Message: "ParseError: failed to parse message",
				Level:   metrics.ErrorLvl,
				Field: metrics.Field{
					"err":  err,
					"data": string(incoming[:n]),
				},
			})
			return
		}

		atomic.AddInt64(&cn.totalRead, int64(n))

		// Lets shrink buffer abit within area.
		if n == len(incoming) && n < mnet.MaxBufferSize {
			incoming = incoming[0 : mnet.MinBufferSize*2]
		}

		if n < len(incoming)/2 && len(incoming) > mnet.MinBufferSize {
			incoming = incoming[0 : len(incoming)/2]
		}

		if n > len(incoming) && len(incoming) > mnet.MinBufferSize && n < mnet.MaxBufferSize {
			incoming = incoming[0 : mnet.MaxBufferSize/2]
		}
	}
}

// getConn returns net.Conn for giving addr.
func (cn *clientConn) getConn(_ mnet.Client, addr net.Addr) (*net.UDPConn, error) {
	lastSleep := mnet.MinTemporarySleep

	var err error
	var conn net.Conn
	for {
		conn, err = cn.dialer.Dial(cn.network, addr.String())
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

	if err != nil {
		return nil, err
	}

	if udpcon, ok := conn.(*net.UDPConn); ok {
		return udpcon, nil
	}

	return nil, ErrExpectedUDPConn
}
