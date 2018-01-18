package msocks

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
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
	wsState    = ws.StateServerSide
	bufferPool = done.NewDonePool(218, 20)
)

type socketConn struct {
	totalRead      int64
	totalWritten   int64
	totalFlushOut  int64
	totalWriteMsgs int64
	totalReadMsgs  int64
	id             string
	nid            string
	localAddr      net.Addr
	remoteAddr     net.Addr
	maxWrite       int
	maxDeadline    time.Duration
	handshake      ws.Handshake
	metrics        metrics.Metrics
	parser         *internal.TaggedMessages
	closedCounter  int64
	waiter         sync.WaitGroup
	wsReader       *wsutil.Reader
	bu             sync.Mutex
	header         []byte
	wsWriter       *wsutil.Writer
	cu             sync.Mutex
	conn           net.Conn
}

func (sc *socketConn) getRemoteAddr(_ mnet.Client) (net.Addr, error) {
	return sc.remoteAddr, nil
}

func (sc *socketConn) getLocalAddr(_ mnet.Client) (net.Addr, error) {
	return sc.localAddr, nil
}

func (sc *socketConn) getStatistics(_ mnet.Client) (mnet.ClientStatistic, error) {
	var stats mnet.ClientStatistic
	stats.ID = sc.id
	stats.Local = sc.localAddr
	stats.Remote = sc.remoteAddr
	stats.BytesRead = atomic.LoadInt64(&sc.totalRead)
	stats.BytesFlushed = atomic.LoadInt64(&sc.totalFlushOut)
	stats.BytesWritten = atomic.LoadInt64(&sc.totalWritten)
	stats.MessagesRead = atomic.LoadInt64(&sc.totalReadMsgs)
	stats.MessagesWritten = atomic.LoadInt64(&sc.totalWriteMsgs)
	return stats, nil
}

func (sc *socketConn) write(mn mnet.Client, size int) (io.WriteCloser, error) {
	if err := sc.isAlive(mn); err != nil {
		return nil, err
	}

	var conn net.Conn
	sc.cu.Lock()
	conn = sc.conn
	sc.cu.Unlock()

	var writer *wsutil.Writer
	sc.bu.Lock()
	writer = sc.wsWriter
	sc.bu.Unlock()

	if conn == nil {
		return nil, mnet.ErrAlreadyClosed
	}

	if writer == nil {
		return nil, mnet.ErrAlreadyClosed
	}

	return bufferPool.Get(size, func(d int, from io.WriterTo) error {
		atomic.AddInt64(&sc.totalWriteMsgs, 1)
		atomic.AddInt64(&sc.totalWritten, int64(d))

		sc.bu.Lock()
		defer sc.bu.Unlock()

		buffered := writer.Buffered()
		atomic.AddInt64(&sc.totalFlushOut, int64(buffered))

		// size of next write.
		toWrite := buffered + d

		// add size header
		toWrite += mnet.HeaderLength

		if toWrite >= sc.maxWrite {
			conn.SetWriteDeadline(time.Now().Add(sc.maxDeadline))
			if err := writer.Flush(); err != nil {
				conn.SetWriteDeadline(time.Time{})
				sc.metrics.Emit(
					metrics.Error(err),
					metrics.WithID(sc.id),
					metrics.With("network", sc.nid),
					metrics.Message("Connection failed to preflush existing data before new write"),
				)
				return err
			}
			conn.SetWriteDeadline(time.Time{})
		}

		// write length header first.
		binary.BigEndian.PutUint32(sc.header, uint32(d))
		writer.Write(sc.header)

		// then flush data alongside header.
		_, err := from.WriteTo(writer)
		return err
	}), nil
}

func (sc *socketConn) isAlive(_ mnet.Client) error {
	if atomic.LoadInt64(&sc.closedCounter) == 1 {
		return mnet.ErrAlreadyClosed
	}
	return nil
}

func (sc *socketConn) read(mn mnet.Client) ([]byte, error) {
	data, _, err := sc.readFrom(mn)
	return data, err
}

func (sc *socketConn) readFrom(mn mnet.Client) ([]byte, net.Addr, error) {
	if err := sc.isAlive(mn); err != nil {
		return nil, nil, err
	}

	indata, from, err := sc.parser.Next()
	atomic.AddInt64(&sc.totalReadMsgs, 1)
	return indata, from, err
}

func (sc *socketConn) flush(mn mnet.Client) error {
	if err := sc.isAlive(mn); err != nil {
		return err
	}

	var conn net.Conn
	sc.cu.Lock()
	conn = sc.conn
	sc.cu.Unlock()

	var writer *wsutil.Writer
	sc.bu.Lock()
	writer = sc.wsWriter
	sc.bu.Unlock()

	if conn == nil {
		return mnet.ErrAlreadyClosed
	}

	if writer == nil {
		return mnet.ErrAlreadyClosed
	}

	if writer.Buffered() != 0 {
		conn.SetWriteDeadline(time.Now().Add(sc.maxDeadline))
		if err := writer.Flush(); err != nil {
			conn.SetWriteDeadline(time.Time{})
			sc.metrics.Emit(
				metrics.Error(err),
				metrics.WithID(sc.id),
				metrics.Message("Connection failed to flush data"),
				metrics.With("network", sc.nid),
			)
			return err
		}
		conn.SetWriteDeadline(time.Time{})
	}

	return nil
}

func (sc *socketConn) close(mn mnet.Client) error {
	if err := sc.isAlive(mn); err != nil {
		return err
	}

	atomic.StoreInt64(&sc.closedCounter, 1)

	var conn net.Conn

	sc.cu.Lock()
	conn = sc.conn
	sc.conn = nil
	sc.cu.Unlock()

	if conn == nil {
		return mnet.ErrAlreadyClosed
	}

	var writer *wsutil.Writer
	sc.bu.Lock()
	writer = sc.wsWriter
	sc.wsWriter = nil
	sc.bu.Unlock()

	if writer.Buffered() != 0 {
		conn.SetWriteDeadline(time.Now().Add(sc.maxDeadline))
		writer.Flush()
		conn.SetWriteDeadline(time.Time{})
	}
	writer.Reset(nil, wsState, ws.OpBinary)

	sc.cu.Lock()
	sc.wsWriter = nil
	sc.wsReader = nil
	sc.cu.Unlock()

	err := conn.Close()

	sc.waiter.Wait()

	return err
}

// readLoop handles the necessary operation of reading data from the
// underline connection.
func (sc *socketConn) readLoop(conn net.Conn, reader *wsutil.Reader) {
	defer sc.close(mnet.Client{})
	defer sc.waiter.Done()

	incoming := make([]byte, mnet.MinBufferSize, mnet.MaxBufferSize)

	for {
		frame, err := reader.NextFrame()
		if err != nil {
			sc.metrics.Emit(
				metrics.Error(err),
				metrics.WithID(sc.id),
				metrics.With("client", sc.id),
				metrics.With("network", sc.nid),
				metrics.Message("Connection failed to read next frame"),
			)
			return
		}

		if int(frame.Length) > len(incoming) && int(frame.Length) < mnet.MaxBufferSize {
			incoming = incoming[:int(frame.Length)]
		}

		n, err := reader.Read(incoming)
		if err != nil {
			sc.metrics.Emit(
				metrics.Error(err),
				metrics.WithID(sc.id),
				metrics.Message("Connection failed to read: closing"),
				metrics.With("network", sc.nid),
			)
			return
		}

		// if nothing was read, skip.
		if n == 0 && len(incoming) == 0 {
			continue
		}

		sc.metrics.Send(metrics.Entry{
			Message: "Received websocket message",
			Field: metrics.Field{
				"data": string(incoming[:n]),
			},
		})

		// Send into go-routine (critical path)?
		if err := sc.parser.Parse(incoming[:n], nil); err != nil {
			sc.metrics.Emit(
				metrics.Error(err),
				metrics.WithID(sc.id),
				metrics.Message("Connection failed to read: closing"),
				metrics.With("network", sc.nid),
			)
			return
		}

		atomic.AddInt64(&sc.totalRead, int64(n))

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

// Network defines a network which runs ontop of provided mnet.ConnHandler.
type Network struct {
	ID         string
	Addr       string
	ServerName string
	TLS        *tls.Config
	Upgrader   *ws.Upgrader
	Handler    mnet.ConnHandler
	Metrics    metrics.Metrics

	totalClients int64
	totalClosed  int64
	totalActive  int64
	totalOpened  int64
	started      int64

	// ClientMaxWriteDeadline defines max time before all clients collected writes must be written to the connection.
	MaxDeadline time.Duration

	// ClientMaxWriteSize sets given max size of buffer for client, each client writes collected
	// till flush must not exceed else will not be buffered and will be written directly.
	MaxWriteSize int

	raddr    net.Addr
	cu       sync.RWMutex
	clients  map[string]*socketConn
	routines sync.WaitGroup
}

func (n *Network) isAlive() error {
	if atomic.LoadInt64(&n.started) == 0 {
		return errors.New("not started yet")
	}
	return nil
}

// Start initializes the network listener.
func (n *Network) Start(ctx context.Context) error {
	if err := n.isAlive(); err == nil {
		return nil
	}

	if n.ID == "" {
		n.ID = uuid.NewV4().String()
	}

	if n.Metrics == nil {
		n.Metrics = metrics.New()
	}

	if n.Upgrader == nil {
		n.Upgrader = &ws.Upgrader{}
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

	n.raddr = stream.Addr()
	n.clients = make(map[string]*socketConn)

	if n.MaxWriteSize <= 0 {
		n.MaxWriteSize = mnet.MaxBufferSize
	}

	if n.MaxDeadline <= 0 {
		n.MaxDeadline = mnet.MaxFlushDeadline
	}

	n.routines.Add(1)
	go n.handleConnections(ctx, stream)
	go func() {
		<-ctx.Done()
		stream.Close()
	}()

	return nil
}

func (n *Network) addClient(ctx context.Context, conn net.Conn, hs ws.Handshake) {
	atomic.AddInt64(&n.totalClients, 1)
	atomic.AddInt64(&n.totalActive, 1)
	atomic.AddInt64(&n.totalOpened, 1)

	wsReader := wsutil.NewReader(conn, wsState)
	wsWriter := wsutil.NewWriterSize(conn, wsState, ws.OpBinary, n.MaxWriteSize)

	client := new(socketConn)
	client.nid = n.ID
	client.conn = conn
	client.handshake = hs
	client.metrics = n.Metrics
	client.wsReader = wsReader
	client.wsWriter = wsWriter
	client.maxWrite = n.MaxWriteSize
	client.header = make([]byte, mnet.HeaderLength)
	client.id = uuid.NewV4().String()
	client.maxDeadline = n.MaxDeadline
	client.localAddr = conn.LocalAddr()
	client.remoteAddr = conn.RemoteAddr()
	client.parser = new(internal.TaggedMessages)

	defer n.Metrics.Emit(
		metrics.Message("Network.addClient: add new client"),
		metrics.With("network", n.ID),
		metrics.With("addr", n.Addr),
		metrics.With("serverName", n.ServerName),
		metrics.With("client_id", client.id),
		metrics.With("client_addr", conn.RemoteAddr()),
		metrics.WithID(n.ID),
	)

	client.waiter.Add(1)
	go func() {
		defer atomic.AddInt64(&n.totalClients, -1)
		defer atomic.AddInt64(&n.totalActive, -1)
		defer atomic.AddInt64(&n.totalClosed, 1)

		client.readLoop(conn, wsReader)

		n.cu.Lock()
		delete(n.clients, client.remoteAddr.String())
		n.cu.Unlock()
	}()

	var mclient mnet.Client
	mclient.Metrics = n.Metrics
	mclient.NID = n.ID
	mclient.ID = client.id
	mclient.CloseFunc = client.close
	mclient.WriteFunc = client.write
	mclient.ReaderFunc = client.read
	mclient.FlushFunc = client.flush
	mclient.LiveFunc = client.isAlive
	mclient.StatisticFunc = client.getStatistics
	mclient.LiveFunc = client.isAlive
	mclient.LocalAddrFunc = client.getLocalAddr
	mclient.RemoteAddrFunc = client.getRemoteAddr
	mclient.SiblingsFunc = func(_ mnet.Client) ([]mnet.Client, error) {
		return n.getAllClient(client.remoteAddr), nil
	}

	n.routines.Add(1)
	go func(mc mnet.Client, addr net.Addr) {
		defer n.routines.Done()

		if err := n.Handler(mc); err != nil {
			atomic.StoreInt64(&client.closedCounter, 1)
			n.Metrics.Emit(
				metrics.Error(err),
				metrics.WithID(mclient.ID),
				metrics.Message("Connection handler failed"),
				metrics.With("network", n.ID),
				metrics.With("addr", addr),
			)
		}

	}(mclient, client.remoteAddr)
}

func (n *Network) getAllClient(addr net.Addr) []mnet.Client {
	n.cu.Lock()
	defer n.cu.Unlock()

	var clients []mnet.Client
	for _, conn := range n.clients {
		if conn.remoteAddr == addr {
			continue
		}

		var client mnet.Client
		client.NID = n.ID
		client.ID = conn.id
		client.Metrics = n.Metrics
		client.LiveFunc = conn.isAlive
		client.WriteFunc = conn.write
		client.FlushFunc = conn.flush
		client.StatisticFunc = conn.getStatistics
		client.RemoteAddrFunc = conn.getRemoteAddr
		client.LocalAddrFunc = conn.getLocalAddr
		client.SiblingsFunc = func(_ mnet.Client) ([]mnet.Client, error) {
			return n.getAllClient(conn.remoteAddr), nil
		}
		clients = append(clients, client)
	}

	return clients
}

// handleConnections runs the process of listening for new connections and
// creating appropriate client objects which will handle behaviours
// appropriately.
func (n *Network) handleConnections(ctx context.Context, stream melon.ConnReadWriteCloser) {
	defer n.routines.Done()
	defer n.closeClientConnections(ctx)

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

		handshake, err := n.Upgrader.Upgrade(newConn)
		if err != nil {
			n.Metrics.Send(metrics.Entry{
				ID: n.ID,
				Field: metrics.Field{
					"err":         err,
					"remote_addr": newConn.RemoteAddr(),
					"local_addr":  newConn.LocalAddr(),
				},
				Message: "Failed to upgrade connection",
			})

			newConn.Close()
			continue
		}

		n.Metrics.Send(metrics.Entry{
			ID: n.ID,
			Field: metrics.Field{
				"handshake":   handshake,
				"remote_addr": newConn.RemoteAddr(),
				"local_addr":  newConn.LocalAddr(),
			},
			Message: "Upgraded connection successfully",
		})

		n.addClient(ctx, newConn, handshake)
	}
}

func (n *Network) closeClientConnections(ctx context.Context) {
	n.cu.RLock()
	defer n.cu.RUnlock()
	for _, conn := range n.clients {
		conn.close(mnet.Client{})
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
