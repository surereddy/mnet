package mtcp

import (
	"bufio"
	"crypto/tls"
	"encoding/binary"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"context"

	"io"

	"bytes"

	"github.com/influx6/faux/metrics"
	"github.com/influx6/faux/netutils"
	"github.com/influx6/faux/pools/done"
	"github.com/influx6/melon"
	"github.com/influx6/mnet"
	"github.com/influx6/mnet/internal"
	"github.com/influx6/mnet/mlisten"
	uuid "github.com/satori/go.uuid"
)

var (
	bufferPool = done.NewDonePool(218, 20)
)

type networkConn struct {
	nid        string
	id         string
	localAddr  net.Addr
	remoteAddr net.Addr
	metrics    metrics.Metrics
	worker     sync.WaitGroup
	do         sync.Once

	maxWrite    int
	maxDeadline time.Duration

	totalRead      int64
	totalWritten   int64
	totalFlushOut  int64
	totalWriteMsgs int64
	totalReadMsgs  int64
	closed         int64

	sos    *guardedBuffer
	parser *internal.TaggedMessages

	mu   sync.RWMutex
	conn net.Conn

	bu         sync.Mutex
	buffWriter *bufio.Writer
}

// isLive returns an error if networkconn is disconnected from network.
func (nc *networkConn) isLive(cm mnet.Client) error {
	if atomic.LoadInt64(&nc.closed) == 1 {
		return mnet.ErrAlreadyClosed
	}
	return nil
}

func (nc *networkConn) getStatistics(cm mnet.Client) (mnet.ClientStatistic, error) {
	var stats mnet.ClientStatistic
	stats.ID = nc.id
	stats.Local = nc.localAddr
	stats.Remote = nc.remoteAddr
	stats.BytesRead = atomic.LoadInt64(&nc.totalRead)
	stats.BytesFlushed = atomic.LoadInt64(&nc.totalFlushOut)
	stats.BytesWritten = atomic.LoadInt64(&nc.totalWritten)
	stats.MessagesRead = atomic.LoadInt64(&nc.totalReadMsgs)
	stats.MessagesWritten = atomic.LoadInt64(&nc.totalWriteMsgs)
	return stats, nil
}

func (nc *networkConn) flush(cm mnet.Client) error {
	if err := nc.isLive(cm); err != nil {
		return err
	}

	var conn net.Conn
	nc.mu.RLock()
	conn = nc.conn
	nc.mu.RUnlock()

	if conn == nil {
		return mnet.ErrAlreadyClosed
	}

	nc.bu.Lock()
	defer nc.bu.Unlock()
	if nc.buffWriter == nil {
		return mnet.ErrAlreadyClosed
	}

	buffered := nc.buffWriter.Buffered()
	atomic.AddInt64(&nc.totalFlushOut, int64(buffered))

	conn.SetWriteDeadline(time.Now().Add(nc.maxDeadline))
	err := nc.buffWriter.Flush()
	conn.SetWriteDeadline(time.Time{})

	return err
}

// read returns data from the underline message list.
func (nc *networkConn) read(cm mnet.Client) ([]byte, error) {
	if err := nc.isLive(cm); err != nil {
		return nil, err
	}

	indata, _, err := nc.parser.Next()
	atomic.AddInt64(&nc.totalReadMsgs, 1)
	return indata, err
}

func (nc *networkConn) write(cm mnet.Client, inSize int) (io.WriteCloser, error) {
	if err := nc.isLive(cm); err != nil {
		return nil, err
	}

	var conn net.Conn
	nc.mu.RLock()
	conn = nc.conn
	nc.mu.RUnlock()

	if conn == nil {
		return nil, mnet.ErrAlreadyClosed
	}

	return bufferPool.Get(inSize, func(incoming int, w io.WriterTo) error {
		atomic.AddInt64(&nc.totalWriteMsgs, 1)
		atomic.AddInt64(&nc.totalWritten, int64(incoming))

		nc.bu.Lock()
		defer nc.bu.Unlock()

		if nc.buffWriter == nil {
			return mnet.ErrAlreadyClosed
		}

		//available := nc.buffWriter.Available()
		buffered := nc.buffWriter.Buffered()
		atomic.AddInt64(&nc.totalFlushOut, int64(buffered))

		// size of next write.
		toWrite := buffered + incoming

		// add size header
		toWrite += mnet.HeaderLength

		if toWrite >= nc.maxWrite {
			conn.SetWriteDeadline(time.Now().Add(nc.maxDeadline))
			if err := nc.buffWriter.Flush(); err != nil {
				conn.SetWriteDeadline(time.Time{})
				return err
			}
			conn.SetWriteDeadline(time.Time{})
		}

		// write length header first.
		header := make([]byte, mnet.HeaderLength)
		binary.BigEndian.PutUint32(header, uint32(incoming))
		nc.buffWriter.Write(header)

		// then flush data alongside header.
		_, err := w.WriteTo(nc.buffWriter)
		return err
	}), nil
}

func (nc *networkConn) closeConnection(cm mnet.Client) error {
	if err := nc.isLive(cm); err != nil {
		return err
	}

	defer nc.metrics.Emit(
		metrics.WithID(nc.id),
		metrics.With("network", nc.nid),
		metrics.Message("networkConn.closeConnection"),
	)

	atomic.StoreInt64(&nc.closed, 1)

	var closeErr error
	nc.do.Do(func() {

		var lastSOS []byte
		nc.sos.Do(func(bu *bytes.Buffer) {
			defer bu.Reset()
			lastSOS = bu.Bytes()
		})

		nc.bu.Lock()
		defer nc.bu.Unlock()

		nc.conn.SetWriteDeadline(time.Now().Add(nc.maxDeadline))
		nc.buffWriter.Flush()
		nc.conn.SetWriteDeadline(time.Time{})

		if len(lastSOS) != 0 {
			nc.conn.SetWriteDeadline(time.Now().Add(nc.maxDeadline))
			nc.buffWriter.Write(lastSOS)
			nc.buffWriter.Flush()
			nc.conn.SetWriteDeadline(time.Time{})
		}

		closeErr = nc.conn.Close()
	})

	nc.mu.Lock()
	nc.conn = nil
	nc.mu.Unlock()

	nc.worker.Wait()

	nc.bu.Lock()
	nc.buffWriter = nil
	nc.bu.Unlock()

	return closeErr
}

func (nc *networkConn) getRemoteAddr(cm mnet.Client) (net.Addr, error) {
	return nc.remoteAddr, nil
}

func (nc *networkConn) getLocalAddr(cm mnet.Client) (net.Addr, error) {
	return nc.localAddr, nil
}

func (nc *networkConn) closeConn(cm mnet.Client) error {
	return nc.closeConnection(cm)
}

// readLoop handles the necessary operation of reading data from the
// underline connection.
func (nc *networkConn) readLoop(cm mnet.Client) {
	defer nc.closeConnection(cm)
	defer nc.worker.Done()

	var cn net.Conn
	nc.mu.RLock()
	cn = nc.conn
	nc.mu.RUnlock()

	if cn == nil {
		return
	}

	incoming := make([]byte, mnet.MinBufferSize, mnet.MaxBufferSize)

	for {
		n, err := cn.Read(incoming)
		if err != nil {
			nc.sos.Do(func(bu *bytes.Buffer) {
				bu.WriteString("-ERR ")
				bu.WriteString(err.Error())
			})
			nc.metrics.Emit(
				metrics.Error(err),
				metrics.WithID(nc.id),
				metrics.Message("Connection failed to read: closing"),
				metrics.With("network", nc.nid),
			)
			return
		}

		// if nothing was read, skip.
		if n == 0 && len(incoming) == 0 {
			continue
		}

		// Send into go-routine (critical path)?
		if err := nc.parser.Parse(incoming[:n], nil); err != nil {
			nc.sos.Do(func(bu *bytes.Buffer) {
				bu.WriteString("-ERR ")
				bu.WriteString(err.Error())
			})

			nc.metrics.Emit(
				metrics.Error(err),
				metrics.WithID(nc.id),
				metrics.Message("ParseError"),
				metrics.With("network", nc.nid),
			)
			return
		}

		atomic.AddInt64(&nc.totalRead, int64(n))

		// Lets resize buffer within area.
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

type networkAction func(*Network)

// Network defines a network which runs ontop of provided mnet.ConnHandler.
type Network struct {
	ID         string
	Addr       string
	ServerName string
	TLS        *tls.Config
	Handler    mnet.ConnHandler
	Metrics    metrics.Metrics

	totalClients int64
	totalClosed  int64
	totalActive  int64
	totalOpened  int64

	// ClientMaxWriteDeadline defines max time before all clients collected writes must be written to the connection.
	ClientMaxWriteDeadline time.Duration

	// ClientMaxWriteSize sets given max size of buffer for client, each client writes collected
	// till flush must not exceed else will not be buffered and will be written directly.
	ClientMaxWriteSize int

	raddr    net.Addr
	pool     chan func()
	cu       sync.RWMutex
	clients  map[string]*networkConn
	ctx      context.Context
	routines sync.WaitGroup
}

// Start initializes the network listener.
func (n *Network) Start(ctx context.Context) error {
	if n.ctx != nil {
		return nil
	}

	if n.Metrics == nil {
		n.Metrics = metrics.New()
	}

	if n.ID == "" {
		n.ID = uuid.NewV4().String()
	}

	n.Addr = netutils.GetAddr(n.Addr)
	if n.ServerName == "" {
		host, _, _ := net.SplitHostPort(n.Addr)
		n.ServerName = host
	}

	defer n.Metrics.Emit(
		metrics.Message("Network.Start"),
		metrics.With("network", n.ID),
		metrics.With("addr", n.Addr),
		metrics.With("serverName", n.ServerName),
		metrics.WithID(n.ID),
	)

	if n.TLS != nil && !n.TLS.InsecureSkipVerify {
		n.TLS.ServerName = n.ServerName
	}

	stream, err := mlisten.Listen("tcp", n.Addr, n.TLS)
	if err != nil {
		return err
	}

	n.ctx = ctx
	n.raddr = stream.Addr()
	n.pool = make(chan func(), 0)
	n.clients = make(map[string]*networkConn)

	if n.ClientMaxWriteSize <= 0 {
		n.ClientMaxWriteSize = mnet.MaxBufferSize
	}

	if n.ClientMaxWriteDeadline <= 0 {
		n.ClientMaxWriteDeadline = mnet.MaxFlushDeadline
	}

	n.routines.Add(2)
	go n.runStream(stream)
	go n.endLogic(ctx, stream)

	return nil
}

func (n *Network) endLogic(ctx context.Context, stream melon.ConnReadWriteCloser) {
	defer n.routines.Done()

	<-ctx.Done()

	n.cu.RLock()
	for _, conn := range n.clients {
		n.cu.RUnlock()
		conn.closeConnection(mnet.Client{})
		n.cu.RLock()
	}
	n.cu.RUnlock()

	if err := stream.Close(); err != nil {
		n.Metrics.Emit(
			metrics.Error(err),
			metrics.Message("Network.endLogic"),
			metrics.With("network", n.ID),
			metrics.With("addr", n.Addr),
			metrics.With("serverName", n.ServerName),
			metrics.WithID(n.ID),
		)
	}
}

// Statistics returns statics associated with Network.
func (n *Network) Statistics() mnet.NetworkStatistic {
	var stats mnet.NetworkStatistic
	stats.ID = n.ID
	stats.LocalAddr = n.raddr
	stats.RemoteAddr = n.raddr
	stats.TotalClients = atomic.LoadInt64(&n.totalClients)
	stats.TotalClosed = atomic.LoadInt64(&n.totalClosed)
	stats.TotalActive = atomic.LoadInt64(&n.totalActive)
	stats.TotalOpened = atomic.LoadInt64(&n.totalOpened)
	return stats
}

// Wait is called to ensure network ended.
func (n *Network) Wait() {
	n.routines.Wait()
}

func (n *Network) getOtherClients(cm mnet.Client) ([]mnet.Client, error) {
	n.cu.Lock()
	defer n.cu.Unlock()

	var clients []mnet.Client
	for id, conn := range n.clients {
		if id == cm.ID {
			continue
		}

		var client mnet.Client
		client.NID = n.ID
		client.ID = conn.id
		client.Metrics = n.Metrics
		client.LiveFunc = conn.isLive
		client.WriteFunc = conn.write
		client.FlushFunc = conn.flush
		client.StatisticFunc = conn.getStatistics
		client.RemoteAddrFunc = conn.getRemoteAddr
		client.LocalAddrFunc = conn.getLocalAddr
		client.SiblingsFunc = n.getOtherClients
		clients = append(clients, client)
	}

	return clients, nil
}

// runStream runs the process of listening for new connections and
// creating appropriate client objects which will handle behaviours
// appropriately.
func (n *Network) runStream(stream melon.ConnReadWriteCloser) {
	defer n.routines.Done()

	defer n.Metrics.Emit(
		metrics.With("network", n.ID),
		metrics.Message("Network.runStream"),
		metrics.With("addr", n.Addr),
		metrics.With("serverName", n.ServerName),
		metrics.WithID(n.ID),
	)

	initial := mnet.MinTemporarySleep

	for {
		newConn, err := stream.ReadConn()
		if err != nil {
			n.Metrics.Send(metrics.Entry{
				ID:      n.ID,
				Field:   metrics.Field{"err": err},
				Message: "Failed to read connection",
			})

			if err == mlisten.ErrListenerClosed {
				return
			}

			netErr, ok := err.(net.Error)
			if !ok {
				continue
			}

			if netErr.Temporary() {
				time.Sleep(initial)
				initial *= 2

				if initial >= mnet.MaxTemporarySleep {
					initial = mnet.MinTemporarySleep
				}
			}

			continue
		}

		atomic.AddInt64(&n.totalClients, 1)
		atomic.AddInt64(&n.totalActive, 1)
		atomic.AddInt64(&n.totalOpened, 1)
		go func(conn net.Conn) {
			defer atomic.AddInt64(&n.totalActive, -1)
			defer atomic.AddInt64(&n.totalOpened, -1)
			defer atomic.AddInt64(&n.totalClosed, 1)

			uuid := uuid.NewV4().String()

			client := mnet.Client{
				ID:      uuid,
				NID:     n.ID,
				Metrics: n.Metrics,
			}

			n.Metrics.Emit(
				metrics.WithID(n.ID),
				metrics.With("network", n.ID),
				metrics.With("client_id", uuid),
				metrics.With("network-addr", n.Addr),
				metrics.With("serverName", n.ServerName),
				metrics.Info("New Client Connection"),
				metrics.With("local_addr", conn.LocalAddr()),
				metrics.With("remote_addr", conn.RemoteAddr()),
			)

			cn := new(networkConn)
			cn.id = uuid
			cn.nid = n.ID
			cn.conn = conn
			cn.metrics = n.Metrics
			cn.parser = new(internal.TaggedMessages)
			cn.localAddr = conn.LocalAddr()
			cn.remoteAddr = conn.RemoteAddr()
			cn.maxWrite = n.ClientMaxWriteSize
			cn.maxDeadline = n.ClientMaxWriteDeadline
			cn.buffWriter = bufio.NewWriterSize(conn, n.ClientMaxWriteSize)
			cn.sos = newGuardedBuffer(512)

			client.LiveFunc = cn.isLive
			client.ReaderFunc = cn.read
			client.WriteFunc = cn.write
			client.FlushFunc = cn.flush
			client.CloseFunc = cn.closeConn
			client.LocalAddrFunc = cn.getLocalAddr
			client.StatisticFunc = cn.getStatistics
			client.SiblingsFunc = n.getOtherClients
			client.RemoteAddrFunc = cn.getRemoteAddr

			cn.worker.Add(1)
			go cn.readLoop(client)

			n.cu.Lock()
			n.clients[uuid] = cn
			n.cu.Unlock()

			atomic.AddInt64(&n.totalClients, 1)
			if err := n.Handler(client); err != nil {
				client.Close()
			}

			n.cu.Lock()
			delete(n.clients, uuid)
			n.cu.Unlock()
		}(newConn)
	}
}

type guardedBuffer struct {
	ml sync.Mutex
	bu *bytes.Buffer
}

func newGuardedBuffer(size int) *guardedBuffer {
	return &guardedBuffer{
		bu: bytes.NewBuffer(make([]byte, 0, size)),
	}
}

func (gb *guardedBuffer) Do(b func(*bytes.Buffer)) {
	gb.ml.Lock()
	defer gb.ml.Unlock()
	b(gb.bu)
}
