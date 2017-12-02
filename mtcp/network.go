package mtcp

import (
	"crypto/tls"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/influx6/faux/context"
	"github.com/influx6/faux/metrics"
	"github.com/influx6/faux/netutils"
	"github.com/influx6/melon"
	"github.com/influx6/mnet"
	"github.com/influx6/mnet/mlisten"
	uuid "github.com/satori/go.uuid"
)

const (
	// MinTemporarySleep sets the minimum, initial sleep a network should
	// take when facing a Temporary net error.
	MinTemporarySleep = 10 * time.Millisecond

	// MaxTemporarySleep sets the maximum, allowed sleep a network should
	// take when facing a Temporary net error.
	MaxTemporarySleep = 1 * time.Second

	// MaxFlushDeadline sets the maximum, allowed duration for flushing data
	MaxFlushDeadline = 2 * time.Second

	// MinBufferSize sets the initial size of space of the slice
	// used to read in content from a net.Conn in the connections
	// read loop.
	MinBufferSize = 512

	// MaxBufferSize sets the maximum size allowed for all reads
	// used in the readloop of a client's net.Conn.
	MaxBufferSize = 69560

	// ServerClientMaxCollectBufferSize sets the collecting buffer size of a Clients.Write
	// which will be helded till Flushed, unless the size of data had exceeded
	// this size, then rest data gets flushed. Always ensure data written is
	// within this given size or split properly.
	ServerClientMaxCollectBufferSize = 60720

	// ClientMaxCollectBufferSize sets the collecting buffer size of a Clients.Write
	// which will be helded till Flushed, unless the size of data had exceeded
	// this size, then rest data gets flushed. Always ensure data written is
	// within this given size or split properly.
	ClientMaxCollectBufferSize = 69500

	// ClientMaxNetConnWriteBuffer sets the maximum allowed buffer size for the interval
	// writer which limits total call to net.Conn.Write. The buffer collects
	// data till the ClientMaxNetConnWriteBuffer and writes such to the net.Conn.
	ClientMaxNetConnWriteBuffer = 69599

	// ClientWriteNetConnDeadline sets the maximum time to await a call to Client.Flush
	// which will reset the writer to collect more data before writing. This helps
	// to both buffer writting data and minimize calls to net.Conn.Write and improve
	// performance.
	ClientWriteNetConnDeadline = 60 * time.Millisecond
)

type message struct {
	data []byte
	next *message
}

type networkConn struct {
	id         string
	ctx        context.CancelContext
	localAddr  net.Addr
	remoteAddr net.Addr
	closer     chan struct{}
	worker     sync.WaitGroup
	do         sync.Once
	parser     *mnet.SizedMessageParser

	totalReadIn   int64
	totalWriteOut int64
	totalFlushOut int64
	totalInBuff   int64
	totalInCBuff  int64

	network *Network

	mu   sync.RWMutex
	Err  error
	conn net.Conn

	buffWriter *mnet.BufferedIntervalWriter
	bw         *mnet.SizeAppendBufferredWriter
}

func (nc *networkConn) getStatistics(cm mnet.Client) (mnet.Statistics, error) {
	var stats mnet.Statistics
	stats.TotalClients = atomic.LoadInt64(&nc.network.totalClients)
	stats.TotalReadInBytes = atomic.LoadInt64(&nc.totalReadIn)
	stats.TotalFlushedInBytes = atomic.LoadInt64(&nc.totalFlushOut)
	stats.TotalWrittenInBytes = atomic.LoadInt64(&nc.totalWriteOut)
	stats.TotalClientsClosed = atomic.LoadInt64(&nc.network.totalClosedClients)
	stats.TotalBytesInBuffer = atomic.LoadInt64(&nc.totalInBuff)
	stats.TotalBytesInCollectBuffer = atomic.LoadInt64(&nc.totalInCBuff)
	return stats, nil
}

func (nc *networkConn) write(cm mnet.Client, data []byte) (int, error) {
	nc.mu.RLock()
	if nc.Err != nil {
		nc.mu.RUnlock()
		return 0, nc.Err
	}
	nc.mu.RUnlock()

	atomic.AddInt64(&nc.totalWriteOut, int64(len(data)))
	atomic.AddInt64(&nc.network.totalClientsWriteOut, int64(len(data)))

	if nc.bw != nil {
		return nc.bw.Write(data)
	}

	return 0, mnet.ErrAlreadyClosed
}

func (nc *networkConn) closeConnection() error {
	defer nc.network.Metrics.Emit(
		metrics.WithID(nc.id),
		metrics.With("network", nc.network.ID),
		metrics.Message("networkConn.closeConnection"),
	)

	nc.mu.RLock()
	if nc.Err != nil {
		nc.mu.RUnlock()
		return nc.Err
	}
	nc.mu.RUnlock()

	nc.network.cu.Lock()
	delete(nc.network.clients, nc.id)
	nc.network.cu.Unlock()

	nc.do.Do(func() {
		// nc.bw.Flush()
		nc.buffWriter.StopTimer()
		nc.conn.SetWriteDeadline(time.Now().Add(MaxFlushDeadline))
		nc.buffWriter.Flush()
		nc.conn.SetWriteDeadline(time.Time{})
		nc.buffWriter.Reset(nil)

		close(nc.closer)

		atomic.AddInt64(&nc.network.totalClosedClients, 1)
		nc.conn.Close()
	})

	nc.mu.Lock()
	nc.Err = mnet.ErrAlreadyClosed
	nc.conn = nil
	nc.mu.Unlock()

	nc.worker.Wait()

	nc.buffWriter = nil
	nc.bw = nil

	return nc.Err
}

func (nc *networkConn) getRemoteAddr(cm mnet.Client) (net.Addr, error) {
	return nc.remoteAddr, nil
}

func (nc *networkConn) getLocalAddr(cm mnet.Client) (net.Addr, error) {
	return nc.localAddr, nil
}

func (nc *networkConn) closeConn(cm mnet.Client) error {
	return nc.closeConnection()
}

func (nc *networkConn) flush(cm mnet.Client) error {
	nc.mu.RLock()
	if nc.Err != nil {
		nc.mu.RUnlock()
		return nc.Err
	}
	nc.mu.RUnlock()

	if nc.bw == nil {
		return mnet.ErrAlreadyClosed
	}

	atomic.StoreInt64(&nc.totalFlushOut, int64(nc.bw.TotalFlushed()))
	atomic.StoreInt64(&nc.totalInCBuff, int64(nc.bw.LengthInBuffer()))
	atomic.StoreInt64(&nc.totalInBuff, int64(nc.buffWriter.Length()))
	return nc.bw.Flush()
}

// read returns data from the underline message list.
func (nc *networkConn) read(cm mnet.Client) ([]byte, error) {
	nc.mu.RLock()
	if nc.Err != nil {
		nc.mu.RUnlock()
		return nil, nc.Err
	}
	nc.mu.RUnlock()

	return nc.parser.Next()
}

// readLoop handles the necessary operation of reading data from the
// underline connection.
func (nc *networkConn) readLoop() {
	defer nc.closeConnection()
	defer nc.worker.Done()

	nc.mu.RLock()
	if nc.conn == nil {
		nc.mu.RUnlock()
		return
	}
	cn := nc.conn
	nc.mu.RUnlock()

	incoming := make([]byte, MinBufferSize, MaxBufferSize)

	for {
		n, err := cn.Read(incoming)
		if err != nil {
			nc.network.Metrics.Emit(
				metrics.Error(err),
				metrics.WithID(nc.id),
				metrics.Message("Connection failed to read: closing"),
				metrics.With("network", nc.network.ID),
			)
			break
		}

		// if nothing was read, skip.
		if n == 0 && len(incoming) == 0 {
			continue
		}

		// Send into go-routine (critical path)?
		nc.parser.Parse(incoming[:n])

		atomic.AddInt64(&nc.totalReadIn, int64(n))
		atomic.AddInt64(&nc.network.totalClientsReadIn, int64(n))

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

type networkAction func(*Network)

// Network defines a network which runs ontop of provided mnet.ConnHandler.
type Network struct {
	ID      string
	Addr    string
	TLS     *tls.Config
	Handler mnet.ConnHandler
	Metrics metrics.Metrics

	totalClients         int64
	totalClosedClients   int64
	totalClientsReadIn   int64
	totalClientsWriteOut int64

	// ClientMaxWriteDeadline defines max time before all clients collected writes must be written to the connection.
	ClientMaxWriteDeadline time.Duration

	// ClientInitialWriteSize sets given size of buffer for client's writer as it collects
	// data till flush.
	ClientInitialWriteSize int

	// ClientMaxWriteSize sets given max size of buffer for client, each client writes collected
	// till flush must not exceed else will not be buffered and will be written directly.
	ClientMaxWriteSize int

	pool     chan func()
	cu       sync.RWMutex
	clients  map[string]*networkConn
	ctx      context.CancelContext
	routines sync.WaitGroup
}

// Start initializes the network listener.
func (n *Network) Start(ctx context.CancelContext) error {
	if n.ctx != nil {
		return nil
	}

	if n.Metrics == nil {
		n.Metrics = metrics.New()
	}

	if n.ID == "" {
		n.ID = uuid.NewV4().String()
	}

	defer n.Metrics.Emit(
		metrics.Message("Network.Start"),
		metrics.With("network", n.ID),
		metrics.WithID(n.ID),
	)

	n.Addr = netutils.GetAddr(n.Addr)
	host, _, _ := net.SplitHostPort(n.Addr)
	if n.TLS != nil && !n.TLS.InsecureSkipVerify {
		n.TLS.ServerName = host
	}

	stream, err := mlisten.Listen("tcp", n.Addr, n.TLS)
	if err != nil {
		return err
	}

	n.ctx = ctx
	n.pool = make(chan func(), 0)
	n.clients = make(map[string]*networkConn)

	if n.ClientInitialWriteSize <= 0 {
		n.ClientInitialWriteSize = ServerClientMaxCollectBufferSize
	}

	if n.ClientMaxWriteDeadline <= 0 {
		n.ClientMaxWriteDeadline = ClientWriteNetConnDeadline
	}

	if n.ClientMaxWriteSize <= 0 {
		n.ClientMaxWriteSize = ClientMaxNetConnWriteBuffer
	}

	n.routines.Add(2)
	go n.runStream(stream)
	go n.endLogic(ctx, stream)

	return nil
}

func (n *Network) endLogic(ctx context.CancelContext, stream melon.ConnReadWriteCloser) {
	defer n.routines.Done()

	<-ctx.Done()

	n.cu.RLock()
	for _, conn := range n.clients {
		n.cu.RUnlock()
		conn.closeConnection()
		n.cu.RLock()
	}
	n.cu.RUnlock()

	if err := stream.Close(); err != nil {
		n.Metrics.Emit(
			metrics.Error(err),
			metrics.Message("Network.endLogic"),
			metrics.With("network", n.ID),
			metrics.WithID(n.ID),
		)
	}
}

// Statistics returns statics associated with Network.
func (n *Network) Statistics() mnet.Statistics {
	var stats mnet.Statistics
	stats.TotalClients = atomic.LoadInt64(&n.totalClients)
	stats.TotalReadInBytes = atomic.LoadInt64(&n.totalClientsReadIn)
	stats.TotalWrittenInBytes = atomic.LoadInt64(&n.totalClientsWriteOut)
	stats.TotalClientsClosed = atomic.LoadInt64(&n.totalClosedClients)
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

		client := mnet.Client{
			ID:      id,
			NID:     n.ID,
			Metrics: n.Metrics,
		}
		client.WriteFunc = conn.write
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
		metrics.WithID(n.ID),
	)

	initial := MinTemporarySleep

	for {
		newConn, err := stream.ReadConn()
		if err != nil {
			n.Metrics.Emit(metrics.WithID(n.ID), metrics.Error(err), metrics.Message("Failed to read new connection"))
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

				if initial >= MaxTemporarySleep {
					initial = MinTemporarySleep
				}
			}

			continue
		}

		go func(conn net.Conn) {
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
				metrics.Info("New Client Connection"),
				metrics.With("local_addr", conn.LocalAddr()),
				metrics.With("remote_addr", conn.RemoteAddr()),
			)

			cn := new(networkConn)
			cn.id = uuid
			cn.ctx = n.ctx
			cn.network = n
			cn.conn = conn
			cn.parser = mnet.NewSizedMessageParser()
			cn.localAddr = conn.LocalAddr()
			cn.remoteAddr = conn.RemoteAddr()
			cn.closer = make(chan struct{}, 0)
			cn.buffWriter = mnet.NewBufferedIntervalWriter(conn, n.ClientMaxWriteSize, n.ClientMaxWriteDeadline)
			cn.bw = mnet.NewSizeAppenBuffereddWriter(cn.buffWriter, n.ClientInitialWriteSize)

			client.ReaderFunc = cn.read
			client.WriteFunc = cn.write
			client.FlushFunc = cn.flush
			client.CloseFunc = cn.closeConn
			client.LocalAddrFunc = cn.getLocalAddr
			client.StatisticFunc = cn.getStatistics
			client.SiblingsFunc = n.getOtherClients
			client.RemoteAddrFunc = cn.getRemoteAddr

			cn.worker.Add(1)

			go cn.readLoop()

			n.cu.Lock()
			n.clients[uuid] = cn
			n.cu.Unlock()

			atomic.AddInt64(&n.totalClients, 1)
			if err := n.Handler(client); err != nil {
				client.Close()
			}
		}(newConn)
	}
}
