package mudp

import (
	"context"
	"encoding/binary"
	"net"
	"sync/atomic"
	"time"

	"sync"

	"bufio"

	"io"

	"github.com/influx6/faux/metrics"
	"github.com/influx6/faux/netutils"
	"github.com/influx6/faux/pools/done"
	"github.com/influx6/mnet"
	uuid "github.com/satori/go.uuid"
)

var (
	bufferPool = done.NewDonePool(218, 20)
)

// targetConn targets a giving connections net.Addr,
// using the underline udp connection.
type targetConn struct {
	target net.Addr
	conn   *net.UDPConn
	//notDirect bool
}

// Write implements the io.Writer logic.
func (t targetConn) Write(d []byte) (int, error) {
	//if t.notDirect
	return t.conn.WriteTo(d, t.target)
}

type netClient struct {
	totalRead      int64
	totalWritten   int64
	totalFlushOut  int64
	totalWriteMsgs int64
	totalReadMsgs  int64
	id             string
	nid            string
	mainAddr       net.Addr
	localAddr      net.Addr
	remoteAddr     net.Addr
	maxWrite       int
	ctx            context.Context
	maxDeadline    time.Duration
	metrics        metrics.Metrics
	parser         *mnet.TaggedMessages
	closedCounter  int64
	bu             sync.RWMutex
	buffers        map[string]*bufio.Writer
	cu             sync.Mutex
	conn           *net.UDPConn
}

func (nc *netClient) getRemoteAddr(_ mnet.Client) (net.Addr, error) {
	return nc.remoteAddr, nil
}

func (nc *netClient) getLocalAddr(_ mnet.Client) (net.Addr, error) {
	return nc.localAddr, nil
}

func (nc *netClient) getStatistics(_ mnet.Client) (mnet.ClientStatistic, error) {
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

func (nc *netClient) write(jm mnet.Client, size int) (io.WriteCloser, error) {
	return nc.writeTo(jm, nc.mainAddr, size)
}

func (nc *netClient) writeTo(_ mnet.Client, addr net.Addr, size int) (io.WriteCloser, error) {
	select {
	case <-nc.ctx.Done():
		return nil, mnet.ErrAlreadyClosed
	default:
		if atomic.LoadInt64(&nc.closedCounter) == 1 {
			return nil, mnet.ErrAlreadyClosed
		}

		var conn *net.UDPConn
		nc.cu.Lock()
		conn = nc.conn
		nc.cu.Unlock()

		if conn == nil {
			return nil, mnet.ErrAlreadyClosed
		}

		nc.bu.Lock()
		if _, ok := nc.buffers[addr.String()]; !ok {
			nc.buffers[addr.String()] = bufio.NewWriterSize(targetConn{
				conn:   conn,
				target: addr,
			}, nc.maxWrite)
		}
		nc.bu.Unlock()

		return bufferPool.Get(size, func(d int, from io.WriterTo) error {
			atomic.AddInt64(&nc.totalWriteMsgs, 1)
			atomic.AddInt64(&nc.totalWritten, int64(d))

			nc.bu.RLock()
			defer nc.bu.RUnlock()

			buffer := nc.buffers[addr.String()]

			buffered := buffer.Buffered()
			atomic.AddInt64(&nc.totalFlushOut, int64(buffered))

			// size of next write.
			toWrite := buffered + d

			// add size header
			toWrite += mnet.HeaderLength

			if toWrite >= nc.maxWrite {
				if err := buffer.Flush(); err != nil {
					return err
				}
			}

			// write length header first.
			header := make([]byte, mnet.HeaderLength)
			binary.BigEndian.PutUint32(header, uint32(d))
			buffer.Write(header)

			// then flush data alongside header.
			_, err := from.WriteTo(buffer)
			return err
		}), nil
	}
}

func (nc *netClient) isAlive(_ mnet.Client) error {
	if atomic.LoadInt64(&nc.closedCounter) == 1 {
		return mnet.ErrAlreadyClosed
	}
	return nil
}

func (nc *netClient) readFrom(_ mnet.Client) ([]byte, net.Addr, error) {
	select {
	case <-nc.ctx.Done():
		return nil, nil, mnet.ErrAlreadyClosed
	default:
		if atomic.LoadInt64(&nc.closedCounter) == 1 {
			return nil, nil, mnet.ErrAlreadyClosed
		}

		indata, from, err := nc.parser.Next()
		atomic.AddInt64(&nc.totalReadMsgs, 1)
		return indata, from, err
	}
}

func (nc *netClient) flushAddr(_ mnet.Client, addr net.Addr) error {
	select {
	case <-nc.ctx.Done():
		return mnet.ErrAlreadyClosed
	default:
		if atomic.LoadInt64(&nc.closedCounter) == 1 {
			return mnet.ErrAlreadyClosed
		}

		var conn *net.UDPConn

		nc.cu.Lock()
		conn = nc.conn
		nc.cu.Unlock()

		if conn == nil {
			return mnet.ErrAlreadyClosed
		}

		nc.bu.RLock()
		defer nc.bu.RUnlock()

		if buffer, ok := nc.buffers[addr.String()]; ok {
			conn.SetWriteDeadline(time.Now().Add(nc.maxDeadline))
			err := buffer.Flush()
			conn.SetWriteDeadline(time.Time{})
			return err
		}

		return nil
	}
}

func (nc *netClient) flush(_ mnet.Client) error {
	select {
	case <-nc.ctx.Done():
		return mnet.ErrAlreadyClosed
	default:
		if atomic.LoadInt64(&nc.closedCounter) == 1 {
			return mnet.ErrAlreadyClosed
		}

		var conn *net.UDPConn

		nc.cu.Lock()
		conn = nc.conn
		nc.cu.Unlock()

		if conn == nil {
			return mnet.ErrAlreadyClosed
		}

		nc.bu.RLock()
		defer nc.bu.RUnlock()

		for _, buffer := range nc.buffers {
			conn.SetWriteDeadline(time.Now().Add(nc.maxDeadline))
			buffer.Flush()
			conn.SetWriteDeadline(time.Time{})
		}

		return nil
	}
}

func (nc *netClient) close(_ mnet.Client) error {
	select {
	case <-nc.ctx.Done():
		return mnet.ErrAlreadyClosed
	default:
		if atomic.LoadInt64(&nc.closedCounter) == 1 {
			return mnet.ErrAlreadyClosed
		}

		atomic.StoreInt64(&nc.closedCounter, 1)

		var conn *net.UDPConn

		nc.cu.Lock()
		conn = nc.conn
		nc.conn = nil
		nc.cu.Unlock()

		if conn == nil {
			return mnet.ErrAlreadyClosed
		}

		nc.bu.Lock()
		defer nc.bu.Unlock()

		for key, buffer := range nc.buffers {
			conn.SetWriteDeadline(time.Now().Add(nc.maxDeadline))
			buffer.Flush()
			conn.SetWriteDeadline(time.Time{})
			buffer.Reset(nil)
			delete(nc.buffers, key)
		}

		return nil
	}
}

func (nc *netClient) handleMessage(data []byte, target net.Addr) error {
	select {
	case <-nc.ctx.Done():
		return mnet.ErrAlreadyClosed
	default:
		if atomic.LoadInt64(&nc.closedCounter) == 1 {
			return mnet.ErrAlreadyClosed
		}

		err := nc.parser.Parse(data, target)
		if err == nil {
			atomic.AddInt64(&nc.totalRead, int64(len(data)))
		}

		return err
	}
}

// Network defines a network which runs ontop of provided mnet.ConnHandler.
type Network struct {
	ID                 string
	Addr               string
	Network            string
	ServerName         string
	Multicast          bool
	MulticastInterface *net.Interface
	Handler            mnet.ConnHandler
	Metrics            metrics.Metrics

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

	ctx     context.Context
	addr    *net.UDPAddr
	raddr   net.Addr
	laddr   net.Addr
	rung    sync.WaitGroup
	cu      sync.RWMutex
	clients map[string]*netClient
	mu      sync.Mutex
	conn    *net.UDPConn
}

// Start boots up the server and initializes all internals to make
// itself ready for servicing requests.
func (n *Network) Start(ctx context.Context) error {
	if n.ctx != nil {
		return nil
	}

	if n.Network == "" {
		n.Network = "udp"
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

	udpAddr, err := net.ResolveUDPAddr(n.Network, n.Addr)
	if err != nil {
		return err
	}

	n.addr = udpAddr

	var serverConn *net.UDPConn
	if n.Multicast && n.MulticastInterface != nil {
		serverConn, err = net.ListenMulticastUDP(n.Network, n.MulticastInterface, n.addr)
	} else {
		serverConn, err = net.ListenUDP(n.Network, n.addr)
	}

	if err != nil {
		return err
	}

	n.raddr = serverConn.RemoteAddr()
	n.laddr = serverConn.LocalAddr()

	n.mu.Lock()
	n.conn = serverConn
	n.mu.Unlock()

	n.ctx = ctx
	n.rung.Add(1)
	go n.handleConnections(ctx, serverConn)

	return nil
}

// Wait blocks the call till all go-routines created by network has shutdown.
func (n *Network) Wait() {
	n.rung.Wait()
}

func (n *Network) getAllClient(skipAddr net.Addr) []mnet.Client {
	n.cu.RLock()
	defer n.cu.RUnlock()

	var clients []mnet.Client
	for _, client := range n.clients {
		if client.mainAddr == skipAddr {
			continue
		}
		var mclient mnet.Client
		mclient.NID = n.ID
		mclient.ID = client.id
		mclient.Metrics = n.Metrics
		mclient.WriteFunc = client.write
		mclient.FlushFunc = client.flush
		mclient.FlushAddrFunc = client.flushAddr
		mclient.WriteToFunc = client.writeTo
		mclient.LiveFunc = client.isAlive
		mclient.StatisticFunc = client.getStatistics
		mclient.LocalAddrFunc = client.getLocalAddr
		mclient.RemoteAddrFunc = client.getRemoteAddr
		mclient.SiblingsFunc = func(_ mnet.Client) ([]mnet.Client, error) {
			return n.getAllClient(client.localAddr), nil
		}

		clients = append(clients, mclient)
	}

	return clients
}

func (n *Network) getClient(addr net.Addr, core *net.UDPConn, h mnet.ConnHandler) *netClient {
	n.cu.Lock()
	defer n.cu.Unlock()

	client, ok := n.clients[addr.String()]
	if !ok {
		client = new(netClient)
		client.nid = n.ID
		client.conn = core
		client.mainAddr = addr
		client.remoteAddr = addr
		client.localAddr = n.laddr
		client.metrics = n.Metrics
		client.parser = new(mnet.TaggedMessages)
		client.id = uuid.NewV4().String()
		client.maxWrite = n.MaxWriterSize
		client.maxDeadline = n.MaxWriteDeadline
		client.buffers = make(map[string]*bufio.Writer)
		client.buffers[addr.String()] = bufio.NewWriterSize(targetConn{conn: core, target: addr}, n.MaxWriterSize)

		var mclient mnet.Client
		mclient.Metrics = n.Metrics
		mclient.NID = n.ID
		mclient.ID = client.id
		mclient.CloseFunc = client.close
		mclient.WriteFunc = client.write
		mclient.WriteToFunc = client.writeTo
		mclient.ReaderFromFunc = client.readFrom
		mclient.FlushFunc = client.flush
		mclient.LiveFunc = client.isAlive
		mclient.FlushAddrFunc = client.flushAddr
		mclient.StatisticFunc = client.getStatistics
		mclient.LiveFunc = client.isAlive
		mclient.LocalAddrFunc = client.getLocalAddr
		mclient.RemoteAddrFunc = client.getRemoteAddr
		mclient.SiblingsFunc = func(_ mnet.Client) ([]mnet.Client, error) {
			return n.getAllClient(addr), nil
		}

		n.rung.Add(1)
		go func(mc mnet.Client, addr net.Addr) {
			defer n.rung.Done()

			if err := h(mc); err != nil {
				atomic.StoreInt64(&client.closedCounter, 1)
			}

			n.cu.Lock()
			defer n.cu.Unlock()
			delete(n.clients, addr.String())
		}(mclient, addr)

		// store client in map.
		n.clients[addr.String()] = client
	}

	return client
}

func (n *Network) handleConnections(ctx context.Context, core *net.UDPConn) {
	defer n.handleCloseConnections(ctx)
	defer n.rung.Done()

	done := ctx.Done()

	incoming := make([]byte, mnet.MinBufferSize, mnet.MaxBufferSize)

	for {
		select {
		case <-done:
			return
		default:
			nn, addr, err := core.ReadFrom(incoming)
			if err != nil {
				n.Metrics.Send(metrics.Entry{
					ID:      n.ID,
					Message: "failed to read message from connection",
					Level:   metrics.ErrorLvl,
					Field: metrics.Field{
						"err":  err,
						"addr": addr,
					},
				})
				continue
			}

			client := n.getClient(addr, core, n.Handler)
			if err := client.handleMessage(incoming[:nn], addr); err != nil {
				n.Metrics.Send(metrics.Entry{
					ID:      n.ID,
					Message: "client unable to handle message",
					Level:   metrics.ErrorLvl,
					Field: metrics.Field{
						"err":  err,
						"addr": addr,
						"data": string(incoming[:nn]),
					},
				})
				continue
			}

			// Lets resize buffer within area.
			if nn == len(incoming) && nn < mnet.MaxBufferSize {
				incoming = incoming[0 : mnet.MinBufferSize*2]
			}

			if nn < len(incoming)/2 && len(incoming) > mnet.MinBufferSize {
				incoming = incoming[0 : len(incoming)/2]
			}

			if nn > len(incoming) && len(incoming) > mnet.MinBufferSize && nn < mnet.MaxBufferSize {
				incoming = incoming[0 : mnet.MaxBufferSize/2]
			}
		}
	}
}

func (n *Network) handleCloseConnections(ctx context.Context) {
	defer n.rung.Done()

	n.cu.RLock()
	for _, client := range n.clients {
		atomic.StoreInt64(&client.closedCounter, 1)
	}
	n.cu.RUnlock()

	n.cu.Lock()
	n.clients = make(map[string]*netClient)
	n.cu.Unlock()
}

// Statistics returns statics associated with Network.
func (n *Network) Statistics() mnet.NetworkStatistic {
	var stats mnet.NetworkStatistic
	stats.ID = n.ID
	stats.LocalAddr = n.laddr
	stats.RemoteAddr = n.raddr
	stats.TotalClients = atomic.LoadInt64(&n.totalClients)
	stats.TotalClosed = atomic.LoadInt64(&n.totalClosed)
	stats.TotalActive = atomic.LoadInt64(&n.totalActive)
	stats.TotalOpened = atomic.LoadInt64(&n.totalOpened)
	return stats
}
