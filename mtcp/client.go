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
	"github.com/influx6/mnet/internal"
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
		network.dialTimeout = mnet.DefaultDialTimeout
	}

	if network.keepAliveTimeout <= 0 {
		network.keepAliveTimeout = mnet.DefaultKeepAlive
	}

	if network.clientMaxWriteSize <= 0 {
		network.clientMaxWriteSize = mnet.MaxBufferSize
	}

	if network.clientMaxWriteDeadline <= 0 {
		network.clientMaxWriteDeadline = mnet.MaxFlushDeadline
	}

	if network.dialer == nil {
		network.dialer = &net.Dialer{
			Timeout:   network.dialTimeout,
			KeepAlive: network.keepAliveTimeout,
		}
	}

	network.id = c.ID
	network.addr = addr
	network.parser = new(internal.TaggedMessages)
	//network.scratch = bytes.NewBuffer(make([]byte, 0, mnet.DefaultReconnectBufferSize))
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
	closed          int64

	dialTimeout      time.Duration
	keepAliveTimeout time.Duration

	clientMaxWriteSize     int
	clientMaxWriteDeadline time.Duration

	parser     *internal.TaggedMessages
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

	cu   sync.RWMutex
	conn net.Conn
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
	if atomic.LoadInt64(&cn.closed) == 1 {
		return mnet.ErrAlreadyClosed
	}
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
	if err := cn.isLive(cm); err != nil {
		return err
	}

	var conn net.Conn
	cn.cu.RLock()
	conn = cn.conn
	cn.cu.RUnlock()

	if conn == nil {
		return mnet.ErrAlreadyClosed
	}

	cn.bu.Lock()
	defer cn.bu.Unlock()
	if cn.buffWriter == nil {
		return mnet.ErrAlreadyClosed
	}

	available := cn.buffWriter.Buffered()
	atomic.StoreInt64(&cn.totalFlushed, int64(available))

	conn.SetWriteDeadline(time.Now().Add(cn.clientMaxWriteDeadline))
	err := cn.buffWriter.Flush()
	if err != nil {
		conn.SetWriteDeadline(time.Time{})
		cn.metrics.Emit(
			metrics.Error(err),
			metrics.WithID(cn.id),
			metrics.With("network", cn.nid),
			metrics.Message("clientNetwork.flush"),
		)
		return err
	}
	conn.SetWriteDeadline(time.Time{})

	return err
}

func (cn *clientNetwork) write(cm mnet.Client, inSize int) (io.WriteCloser, error) {
	if err := cn.isLive(cm); err != nil {
		return nil, err
	}

	var conn net.Conn
	cn.cu.RLock()
	conn = cn.conn
	cn.cu.RUnlock()

	if conn == nil {
		return nil, mnet.ErrAlreadyClosed
	}

	return bufferPool.Get(inSize, func(incoming int, w io.WriterTo) error {
		atomic.AddInt64(&cn.MessageWritten, 1)
		atomic.AddInt64(&cn.totalWritten, int64(incoming))

		cn.bu.Lock()
		defer cn.bu.Unlock()

		if cn.buffWriter == nil {
			return mnet.ErrAlreadyClosed
		}

		//available := cn.buffWriter.Available()
		buffered := cn.buffWriter.Buffered()
		atomic.AddInt64(&cn.totalFlushed, int64(buffered))

		// size of next write.
		toWrite := buffered + incoming

		// add size header
		toWrite += mnet.HeaderLength

		if toWrite >= cn.clientMaxWriteSize {
			conn.SetWriteDeadline(time.Now().Add(cn.clientMaxWriteDeadline))
			if err := cn.buffWriter.Flush(); err != nil {
				conn.SetWriteDeadline(time.Time{})
				return err
			}
			conn.SetWriteDeadline(time.Time{})
		}

		// write length header first.
		header := make([]byte, mnet.HeaderLength)
		binary.BigEndian.PutUint32(header, uint32(incoming))
		cn.buffWriter.Write(header)

		// then flush data alongside header.
		_, err := w.WriteTo(cn.buffWriter)
		return err
	}), nil
}

func (cn *clientNetwork) read(cm mnet.Client) ([]byte, error) {
	if err := cn.isLive(cm); err != nil {
		return nil, err
	}

	indata, _, err := cn.parser.Next()
	atomic.AddInt64(&cn.totalRead, int64(len(indata)))
	return indata, err
}

func (cn *clientNetwork) readLoop(cm mnet.Client, conn net.Conn) {
	defer cn.close(cm)
	defer cn.worker.Done()

	incoming := make([]byte, mnet.MinBufferSize, mnet.MaxBufferSize)
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
		if err := cn.parser.Parse(incoming[:n], nil); err != nil {
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

func (cn *clientNetwork) close(cm mnet.Client) error {
	if err := cn.isLive(cm); err != nil {
		return err
	}

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
		conn.SetWriteDeadline(time.Now().Add(mnet.MaxFlushDeadline))
		cn.buffWriter.Flush()
		conn.SetWriteDeadline(time.Time{})
		cn.buffWriter.Reset(cn.scratch)

		conn.Close()
	})

	atomic.StoreInt64(&cn.closed, 1)

	cn.worker.Wait()

	return nil
}

func (cn *clientNetwork) reconnect(cm mnet.Client, altAddr string) error {
	if err := cn.isLive(cm); err != nil {
		return err
	}

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
		return err
	}

	atomic.StoreInt64(&cn.closed, 0)

	cn.buffWriter.Reset(nil)

	// if offline buffer has data written in, before new connection was started,
	// meaning we have offline data, then first flush this in, then set up new connection as
	// writer. But if error in write occurs like short write, then set back scratch as buff's
	// writer, write back data which was not written then return error.
	//if cn.scratch.Len() != 0 {
	//	if _, werr := cn.scratch.WriteTo(conn); werr != nil {
	//		cn.buffWriter.Reset(cn.scratch)
	//		return werr
	//	}
	//}

	cn.cu.Lock()
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
func (cn *clientNetwork) getConn(_ mnet.Client, addr string) (net.Conn, error) {
	lastSleep := mnet.MinTemporarySleep

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
