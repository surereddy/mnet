package mtcp

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"encoding/binary"

	"github.com/influx6/faux/metrics"
	"github.com/influx6/faux/netutils"
	"github.com/influx6/mnet"
	uuid "github.com/satori/go.uuid"
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

// WriteInterval sets the clientNetwork to use the provided value
// as its write intervals for colasced/batch writing of send data.
func WriteInterval(dur time.Duration) ConnectOptions {
	return func(cm *clientNetwork) {
		cm.clientMaxWriteDeadline = dur
	}
}

// MaxBuffer sets the clientNetwork to use the provided value
// as its maximum buffer size for it's writer.
func MaxBuffer(buffer int) ConnectOptions {
	return func(cm *clientNetwork) {
		cm.clientMaxWriteSize = buffer
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

	addr = netutils.GetAddr(addr)
	host, _, _ := net.SplitHostPort(addr)

	network := new(clientNetwork)
	for _, op := range ops {
		op(network)
	}

	c.NID = network.nid

	network.totalReconnects = -1
	if network.tls != nil && !network.tls.InsecureSkipVerify {
		network.tls.ServerName = host
	}

	if network.metrics == nil {
		network.metrics = metrics.New()
	}

	if network.dialTimeout <= 0 {
		network.dialTimeout = DefaultDialTimeout
	}

	if network.keepAliveTimeout <= 0 {
		network.keepAliveTimeout = DefaultKeepAlive
	}

	if network.clientMaxWriteSize <= 0 {
		network.clientMaxWriteDeadline = MaxBufferSize
	}

	if network.clientMaxWriteDeadline <= 0 {
		network.clientMaxWriteDeadline = ClientWriteNetConnDeadline
	}

	if network.dialer == nil {
		network.dialer = &net.Dialer{
			Timeout:   network.dialTimeout,
			KeepAlive: network.keepAliveTimeout,
		}
	}

	network.id = c.ID
	network.addr = addr
	network.parser = new(tagMessages)
	network.scratch = bytes.NewBuffer(make([]byte, 0, DefaultReconnectBufferSize))
	network.buffWriter = bufio.NewWriterSize(network.scratch, network.clientMaxWriteSize)

	c.Metrics = network.metrics
	c.LiveFunc = network.isLive
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
	totalRead       int64
	totalWritten    int64
	totalFlushed    int64
	MessageRead     int64
	MessageWritten  int64
	totalReconnects int64
	totalMisses     int64

	dialTimeout      time.Duration
	keepAliveTimeout time.Duration

	clientMaxWriteSize     int
	clientMaxWriteDeadline time.Duration

	parser     *tagMessages
	scratch    *bytes.Buffer
	bu         sync.Mutex
	buffWriter *bufio.Writer

	secure     bool
	id         string
	nid        string
	addr       string
	localAddr  net.Addr
	remoteAddr net.Addr
	do         sync.Once
	tls        *tls.Config
	worker     sync.WaitGroup
	metrics    metrics.Metrics
	dialer     *net.Dialer

	cu        sync.RWMutex
	conn      net.Conn
	clientErr error
}

func (cn *clientNetwork) getStatistics(cm mnet.Client) (mnet.ClientStatistic, error) {
	var stats mnet.ClientStatistic
	stats.ID = cn.id
	stats.Local = cn.localAddr
	stats.Remote = cn.remoteAddr
	stats.BytesRead = atomic.LoadInt64(&cn.totalRead)
	stats.MessagesRead = atomic.LoadInt64(&cn.MessageRead)
	stats.BytesWritten = atomic.LoadInt64(&cn.totalWritten)
	stats.BytesFlushed = atomic.LoadInt64(&cn.totalFlushed)
	stats.MessagesWritten = atomic.LoadInt64(&cn.MessageWritten)
	stats.Reconnects = atomic.LoadInt64(&cn.totalReconnects)
	return stats, nil
}

// isLive returns error if clientNetwork is not connected to remote network.
func (cn *clientNetwork) isLive(cm mnet.Client) error {
	cn.cu.RLock()
	if cn.clientErr != nil {
		cn.cu.RUnlock()
		return cn.clientErr
	}
	cn.cu.RUnlock()
	return nil
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
	cn.cu.RLock()
	if cn.clientErr != nil {
		cn.cu.RUnlock()
		return cn.clientErr
	}
	cn.cu.RUnlock()

	cn.bu.Lock()
	defer cn.bu.Unlock()
	if cn.buffWriter == nil {
		return mnet.ErrAlreadyClosed
	}

	available := cn.buffWriter.Buffered()
	atomic.StoreInt64(&cn.totalFlushed, int64(available))

	err := cn.buffWriter.Flush()
	if err != nil {
		cn.metrics.Emit(
			metrics.Error(err),
			metrics.WithID(cn.id),
			metrics.With("network", cn.nid),
			metrics.Message("clientNetwork.flush"),
		)
		return err
	}

	return err
}

func (cn *clientNetwork) write(cm mnet.Client, inSize int) (io.WriteCloser, error) {
	cn.cu.RLock()
	if cn.clientErr != nil {
		cn.cu.RUnlock()
		return nil, cn.clientErr
	}
	cn.cu.RUnlock()

	return bufferPool.Get(inSize, func(incoming int, w io.WriterTo) error {
		atomic.AddInt64(&cn.MessageWritten, 1)
		atomic.AddInt64(&cn.totalWritten, int64(incoming))

		cn.bu.Lock()
		defer cn.bu.Unlock()

		if cn.buffWriter == nil {
			return mnet.ErrAlreadyClosed
		}

		buffered := cn.buffWriter.Buffered()
		atomic.AddInt64(&cn.totalFlushed, int64(buffered))

		available := cn.buffWriter.Available()

		// size of next write.
		toWrite := available + incoming

		// add size header
		toWrite += 4

		if toWrite >= cn.clientMaxWriteSize {
			if err := cn.buffWriter.Flush(); err != nil {
				return err
			}
		}

		// write length header first.
		header := make([]byte, headerLength)
		binary.BigEndian.PutUint32(header, uint32(incoming))
		cn.buffWriter.Write(header)

		// then flush data alongside header.
		_, err := w.WriteTo(cn.buffWriter)
		return err
	}), nil
}

func (cn *clientNetwork) read(cm mnet.Client) ([]byte, error) {
	cn.cu.RLock()
	if cn.clientErr != nil {
		cn.cu.RUnlock()
		return nil, cn.clientErr
	}
	cn.cu.RUnlock()
	return cn.parser.Next()
}

func (cn *clientNetwork) readLoop(cm mnet.Client, conn net.Conn) {
	defer cn.close(cm)
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
			return
		}

		// if nothing was read, skip.
		if n == 0 && len(incoming) == 0 {
			continue
		}

		// if we fail to parse the data then we error out.
		if err := cn.parser.Parse(incoming[:n]); err != nil {
			return
		}

		atomic.AddInt64(&cn.totalRead, int64(n))

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
		conn.SetWriteDeadline(time.Now().Add(MaxFlushDeadline))
		cn.buffWriter.Flush()
		conn.SetWriteDeadline(time.Time{})
		cn.buffWriter.Reset(cn.scratch)

		conn.Close()
	})

	cn.cu.Lock()
	cn.clientErr = mnet.ErrAlreadyClosed
	cn.cu.Unlock()

	cn.worker.Wait()

	return nil
}

func (cn *clientNetwork) reconnect(cm mnet.Client, altAddr string) error {
	// if we are still connected, then we can't do anything here.
	cn.cu.RLock()
	if cn.conn != nil {
		cn.cu.RUnlock()
		return mnet.ErrStillConnected
	}
	cn.cu.RUnlock()

	atomic.AddInt64(&cn.totalReconnects, 1)

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
		atomic.AddInt64(&cn.totalReconnects, 1)
		return err
	}

	// Ensure bufWriter timer is stopped immediately and scratch is released from
	// use use by writer as well.
	cn.buffWriter.Reset(nil)

	// if offline buffer has data written in, before new connection was started,
	// meaning we have offline data, then first flush this in, then set up new connection as
	// writer. But if error in write occurs like short write, then set back scratch as buff's
	// writer, write back data which was not written then return error.
	if cn.scratch.Len() != 0 {
		if _, werr := cn.scratch.WriteTo(conn); werr != nil {
			cn.buffWriter.Reset(cn.scratch)
			return werr
		}
	}

	cn.cu.Lock()
	cn.clientErr = nil
	cn.do = sync.Once{}
	cn.conn = conn
	cn.buffWriter.Reset(conn)
	cn.localAddr = conn.LocalAddr()
	cn.remoteAddr = conn.RemoteAddr()
	cn.cu.Unlock()

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
