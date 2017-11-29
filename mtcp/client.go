package mtcp

import (
	"bytes"
	"crypto/tls"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/influx6/faux/metrics"
	"github.com/influx6/mnet"
	uuid "github.com/satori/go.uuid"
)

const (
	// DefaultDialTimeout sets the default maximum time in seconds allowed before
	// a net.Dialer exits attempt to dial a network.
	DefaultDialTimeout = 1 * time.Second

	// DefaultKeepAlive sets the default maximum time to keep alive a tcp connection
	// during no-use. It is used by net.Dialer.
	DefaultKeepAlive = 3 * time.Minute
)

// ConnectOptions defines a function type used to apply given
// changes to a *clientNetwork type
type ConnectOptions func(*clientNetwork)

// TLSConfig sets the giving tls.Config to be used by the returned
// client.
func TLSConfig(config *tls.Config) ConnectOptions {
	return func(cm *clientNetwork) {
		cm.secure = true
		cm.tls = config
	}
}

// SecureConnection sets the clientNetwork to use a tls.Connection
// regardless whether certificate is provided.
func SecureConnection() ConnectOptions {
	return func(cm *clientNetwork) {
		cm.secure = true
	}
}

// ClientWriteInterval sets the clientNetwork to use the provided value
// as its write intervals for colasced/batch writing of send data.
func ClientWriteInterval(dur time.Duration) ConnectOptions {
	return func(cm *clientNetwork) {
		cm.clientMaxWriteDeadline = dur
	}
}

// ClientMaxNetConnWriteBufferSize sets the clientNetwork to use the provided value
// as its maximum buffer size for it's writer.
func ClientMaxNetConnWriteBufferSize(buffer int) ConnectOptions {
	return func(cm *clientNetwork) {
		cm.clientMaxWriteSize = buffer
	}
}

// ClientInitialBuffer sets the clientNetwork to use the provided value
// as its initial buffer size for it's writer.
func ClientInitialBuffer(buffer int) ConnectOptions {
	return func(cm *clientNetwork) {
		cm.clientCollectBufferSize = buffer
	}
}

// Metrics sets the metrics instance to be used by the client for
// logging.
func Metrics(m metrics.Metrics) ConnectOptions {
	return func(cm *clientNetwork) {
		cm.metrics = m
	}
}

// KeepAliveTimeout sets the client to use given timeout for it's connection net.Dialer
// keepAliveTimeout.
func KeepAliveTimeout(dur time.Duration) ConnectOptions {
	return func(cm *clientNetwork) {
		cm.keepAliveTimeout = dur
	}
}

// DialTimeout sets the client to use given timeout for it's connection net.Dialer
// dial timeout.
func DialTimeout(dur time.Duration) ConnectOptions {
	return func(cm *clientNetwork) {
		cm.dialTimeout = dur
	}
}

// NetworkID sets the id used by the client connection for identifying the
// associated network.
func NetworkID(id string) ConnectOptions {
	return func(cm *clientNetwork) {
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

	network := new(clientNetwork)
	for _, op := range ops {
		op(network)
	}

	c.NID = network.nid

	if network.metrics == nil {
		network.metrics = metrics.New()
	}

	if network.dialTimeout <= 0 {
		network.dialTimeout = DefaultDialTimeout
	}

	if network.keepAliveTimeout <= 0 {
		network.keepAliveTimeout = DefaultKeepAlive
	}

	if network.clientCollectBufferSize <= 0 {
		network.clientCollectBufferSize = ClientCollectBufferSize
	}

	if network.clientMaxWriteDeadline <= 0 {
		network.clientMaxWriteDeadline = ClientWriteNetConnDeadline
	}

	if network.clientMaxWriteSize <= 0 {
		network.clientMaxWriteSize = ClientMaxNetConnWriteBuffer
	}

	if network.dialer == nil {
		network.dialer = &net.Dialer{
			Timeout:   network.dialTimeout,
			KeepAlive: network.keepAliveTimeout,
		}
	}

	network.id = c.ID
	network.addr = addr
	network.parser = mnet.NewSizedMessageParser()
	network.buffWriter = mnet.NewBufferedIntervalWriter(&network.scratch, network.clientMaxWriteSize, network.clientMaxWriteDeadline)
	network.bw = mnet.NewSizeAppenBuffereddWriter(network.buffWriter, network.clientCollectBufferSize)

	c.Metrics = network.metrics
	c.FlushFunc = network.flush
	c.ReaderFunc = network.read
	c.WriteFunc = network.write
	c.CloseFunc = network.close
	c.LocalAddrFunc = network.getLocalAddr
	c.RemoteAddrFunc = network.getRemoteAddr
	c.ReconnectionFunc = network.reconnect
	c.StatisticFunc = network.getStatistics

	if err := network.reconnect(c, addr); err != nil {
		return c, err
	}

	return c, nil
}

type clientNetwork struct {
	totalReadIn             int64
	totalWriteOut           int64
	totalWriteFlush         int64
	dialTimeout             time.Duration
	keepAliveTimeout        time.Duration
	clientMaxWriteSize      int
	clientCollectBufferSize int
	secure                  bool
	id                      string
	nid                     string
	addr                    string
	localAddr               net.Addr
	remoteAddr              net.Addr
	do                      sync.Once
	tls                     *tls.Config
	clientMaxWriteDeadline  time.Duration
	worker                  sync.WaitGroup
	metrics                 metrics.Metrics
	scratch                 bytes.Buffer
	dialer                  *net.Dialer
	parser                  *mnet.SizedMessageParser
	buffWriter              *mnet.BufferedIntervalWriter
	bw                      *mnet.SizeAppendBufferredWriter
	cu                      sync.RWMutex
	conn                    net.Conn
	clientErr               error
}

func (cn *clientNetwork) getStatistics(cm mnet.Client) (mnet.Statistics, error) {
	var stats mnet.Statistics
	stats.TotalReadInBytes = atomic.LoadInt64(&cn.totalReadIn)
	stats.TotalFlushedInBytes = atomic.LoadInt64(&cn.totalWriteFlush)
	stats.TotalWrittenInBytes = atomic.LoadInt64(&cn.totalWriteOut)
	return stats, nil
}

func (cn *clientNetwork) getRemoteAddr(cm mnet.Client) (net.Addr, error) {
	cn.cu.RLock()
	defer cn.cu.RUnlock()
	return cn.remoteAddr, nil
}

func (cn *clientNetwork) getLocalAddr(cm mnet.Client) (net.Addr, error) {
	cn.cu.RLock()
	defer cn.cu.RUnlock()
	return cn.localAddr, nil
}

func (cn *clientNetwork) flush(cm mnet.Client) error {
	err := cn.bw.Flush()
	atomic.StoreInt64(&cn.totalWriteFlush, int64(cn.bw.TotalFlushed()))
	if err != nil && err != io.ErrShortWrite {
		cn.metrics.Emit(
			metrics.Error(err),
			metrics.WithID(cn.id),
			metrics.With("network", cn.nid),
			metrics.Message("clientNetwork.flush"),
		)
		cn.close(cm)
	}
	return err
}

func (cn *clientNetwork) close(cm mnet.Client) error {
	cn.cu.RLock()
	if cn.clientErr != nil {
		cn.cu.RUnlock()
		return cn.clientErr
	}
	cn.cu.RUnlock()

	cn.metrics.Emit(
		metrics.WithID(cn.id),
		metrics.With("network", cn.nid),
		metrics.Message("clientNetwork.close"),
	)

	cn.cu.Lock()
	conn := cn.conn
	cn.conn = nil
	cn.cu.Unlock()

	cn.do.Do(func() {
		cn.buffWriter.StopTimer()

		conn.SetWriteDeadline(time.Now().Add(3 * time.Second))
		cn.buffWriter.Flush()
		conn.SetWriteDeadline(time.Time{})
		cn.buffWriter.Reset(&cn.scratch)

		conn.Close()
	})

	cn.cu.Lock()
	cn.clientErr = mnet.ErrAlreadyClosed
	cn.cu.Unlock()

	cn.worker.Wait()

	return nil
}

func (cn *clientNetwork) write(cm mnet.Client, d []byte) (int, error) {
	n, err := cn.bw.Write(d)
	atomic.AddInt64(&cn.totalWriteOut, int64(n))
	if err != nil && err != io.ErrShortWrite {
		cn.metrics.Emit(
			metrics.Error(err),
			metrics.WithID(cn.id),
			metrics.With("network", cn.nid),
			metrics.Message("clientNetwork.flush"),
		)
		cn.close(cm)
	}
	return n, err
}

func (cn *clientNetwork) read(cm mnet.Client) ([]byte, error) {
	return cn.parser.Next()
}

func (cn *clientNetwork) readLoop(cm mnet.Client, conn net.Conn) {
	// defer cn.close(cm)
	defer cn.worker.Done()

	incoming := make([]byte, MinBufferSize, MaxBufferSize)

	for {
		n, err := conn.Read(incoming)
		if err != nil {
			cn.metrics.Emit(
				metrics.Error(err),
				metrics.WithID(cn.id),
				metrics.With("network", cn.nid),
				metrics.Message("Connection failed to read: closing"),
			)
			break
		}

		// if nothing was read, skip.
		if n == 0 && len(incoming) == 0 {
			continue
		}

		// Send into go-routine (critical path)?
		cn.parser.Parse(incoming[:n])

		atomic.AddInt64(&cn.totalReadIn, int64(n))

		// Lets shrink buffer abit within area.
		if n == len(incoming) && n < MaxBufferSize {
			incoming = incoming[0 : MinBufferSize*2]
		}

		if n < len(incoming)/2 && len(incoming) > MinBufferSize {
			incoming = incoming[0 : len(incoming)/2]
		}

		if n > len(incoming) && len(incoming) > MinBufferSize && n < MaxBufferSize {
			incoming = incoming[0 : MaxBufferSize/2]
		}
	}
}

func (cn *clientNetwork) reconnect(cm mnet.Client, altAddr string) error {
	// if we are still connected, then we can't do anything here.
	cn.cu.RLock()
	if cn.conn != nil {
		cn.cu.RUnlock()
		return mnet.ErrStillConnected
	}
	cn.cu.RUnlock()

	// ensure we really have stopped loop.
	cn.worker.Wait()

	var err error
	var conn net.Conn

	// First we test out the alternate address, to see if we get a connection.
	// If we get no connection, then attempt to dial original address and finally
	// return error.
	if altAddr != "" {
		if conn, err = cn.getConn(cm, altAddr); err != nil {
			conn, err = cn.getConn(cm, cn.addr)
		}
	} else {
		conn, err = cn.getConn(cm, cn.addr)
	}

	// If failure was met, then return error and go-offline again.
	if err != nil {
		return err
	}

	// Ensure bufWriter timer is stopped immediately and scratch is released from
	// use use by writer as well.
	cn.buffWriter.Reset(nil)

	data := cn.scratch.Bytes()

	// if offline buffer has data written in, before new connection was started,
	// meaning we have offline data, then first flush this in, then set up new connection as
	// writer. But if error in write occurs like short write, then set back scratch as buff's
	// writer, write back data which was not written then return error.
	if len(data) != 0 {
		cn.scratch.Reset()

		n, err := conn.Write(data)
		if n < len(data) && err == nil {
			err = io.ErrShortWrite
		}

		if err != nil && err != io.ErrShortWrite {
			cn.scratch.Write(data[n:])
			cn.buffWriter.Reset(&cn.scratch)
			return err
		}

		if err != nil && err == io.ErrShortWrite {
			data = data[n:]
		}

		if err == nil && n == len(data) {
			data = data[:0]
		}
	}

	cn.cu.Lock()
	cn.do = sync.Once{}
	cn.conn = conn
	cn.buffWriter.Reset(conn)
	cn.localAddr = conn.LocalAddr()
	cn.remoteAddr = conn.RemoteAddr()
	cn.cu.Unlock()

	// Was it a short write?, then send rest back into buffer.
	if len(data) != 0 {
		cn.buffWriter.Write(data)
	}

	cn.worker.Add(1)
	go cn.readLoop(cm, conn)

	return nil
}

// getConn returns net.Conn for giving addr.
func (cn *clientNetwork) getConn(cm mnet.Client, addr string) (net.Conn, error) {
	lastSleep := MinTemporarySleep

	var err error
	var conn net.Conn
	for {
		conn, err = cn.dialer.Dial("tcp", addr)
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
				if lastSleep >= MaxTemporarySleep {
					return nil, err
				}

				time.Sleep(lastSleep)
				lastSleep *= 2
			}
			continue
		}
		break
	}

	if cn.secure && cn.tls != nil {
		tlsConn := tls.Client(conn, cn.tls)
		if err := tlsConn.Handshake(); err != nil {
			return nil, err
		}

		return tlsConn, nil
	}

	if cn.secure && cn.tls == nil {
		tlsConn := tls.Client(conn, &tls.Config{InsecureSkipVerify: true})
		if err := tlsConn.Handshake(); err != nil {
			return nil, err
		}

		return tlsConn, nil
	}

	return conn, nil
}
