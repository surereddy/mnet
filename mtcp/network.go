package mtcp

import (
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/influx6/faux/context"
	"github.com/influx6/faux/metrics"
	"github.com/influx6/melon"
	"github.com/influx6/mnet"
	"github.com/influx6/mnet/mlisten"
	uuid "github.com/satori/go.uuid"
)

const (
	minSleep               = 10 * time.Millisecond
	maxSleep               = 2 * time.Second
	oneMB                  = 1024 * 1024
	minBufferSize          = 512
	maxBufferSize          = 1024 * minBufferSize
	clientMinInitialBuffer = 8096
	clientMaxBuffer        = 1024 * minBufferSize
	clientWriteDeadline    = 600 * time.Millisecond
)

// errors ...
var (
	ErrAlreadyClosed    = errors.New("already closed connection")
	ErrNoDataYet        = errors.New("data is not yet available for reading")
	ErrInvalidWriteSize = errors.New("returned write size does not match data len")
	ErrParseErr         = errors.New("Failed to parse data")
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
	flushr     chan struct{}
	closer     chan struct{}
	worker     sync.WaitGroup
	do         sync.Once

	scratch bytes.Buffer
	network *Network

	mu   sync.Mutex
	Err  error
	conn net.Conn
	head *message
	tail *message

	buffWriter *mnet.BufferedIntervalWriter
	bw         *mnet.SizeAppendBufferredWriter
}

func (nc *networkConn) write(cm mnet.Client, data []byte) (int, error) {
	nc.mu.Lock()
	if nc.Err != nil {
		nc.mu.Unlock()
		return 0, nc.Err
	}
	nc.mu.Unlock()

	if nc.bw != nil {
		return nc.bw.Write(data)
	}

	return 0, ErrAlreadyClosed
}

func (nc *networkConn) closeConnection() error {
	defer nc.network.Metrics.Emit(
		metrics.WithID(nc.id),
		metrics.With("network", nc.network.ID),
		metrics.Message("networkConn.closeConnection"),
	)

	nc.mu.Lock()
	if nc.Err != nil {
		nc.mu.Unlock()
		return nc.Err
	}
	nc.mu.Unlock()

	nc.network.cu.Lock()
	delete(nc.network.clients, nc.id)
	nc.network.cu.Unlock()

	nc.buffWriter.Flush()

	nc.mu.Lock()
	nc.Err = ErrAlreadyClosed
	nc.mu.Unlock()

	nc.do.Do(func() {
		close(nc.closer)
	})

	nc.worker.Wait()

	nc.mu.Lock()
	nc.conn.Close()
	nc.conn = nil
	nc.mu.Unlock()

	nc.buffWriter = nil
	nc.bw = nil

	return nc.Err
}

func (nc *networkConn) closeConn(cm mnet.Client) error {
	return nc.closeConnection()
}

func (nc *networkConn) flushAll(cm mnet.Client) error {
	if len(nc.flushr) == 0 {
		select {
		case nc.flushr <- struct{}{}:
		default:
		}
	}
	return nil
}

// read returns data from the underline message list.
func (nc *networkConn) read(cm mnet.Client) ([]byte, error) {
	return nc.getMessage()
}

func (nc *networkConn) getMessage() ([]byte, error) {
	nc.mu.Lock()
	defer nc.mu.Unlock()

	if nc.Err != nil {
		return nil, nc.Err
	}

	if nc.tail == nil && nc.head == nil {
		return nil, ErrNoDataYet
	}

	head := nc.head
	if nc.tail == head {
		nc.tail = nil
		nc.head = nil
	} else {
		next := head.next
		head.next = nil
		nc.head = next
	}

	return head.data, nil
}

func (nc *networkConn) addMessage(m *message) {
	nc.mu.Lock()
	defer nc.mu.Unlock()

	if nc.Err != nil {
		return
	}

	if nc.head == nil && nc.tail == nil {
		nc.head = m
		nc.tail = m
		return
	}

	nc.tail.next = m
	nc.tail = m
}

func (nc *networkConn) flushloop() {
	defer nc.worker.Done()

	for {
		select {
		case <-nc.ctx.Done():
			return
		case <-nc.closer:
			return
		case _, ok := <-nc.flushr:
			if !ok {
				return
			}

			nc.mu.Lock()
			if nc.conn == nil {
				nc.mu.Unlock()
				return
			}
			nc.mu.Unlock()

			err := nc.bw.Flush()
			if err != nil {
				nc.network.Metrics.Emit(
					metrics.Error(err),
					metrics.WithID(nc.id),
					metrics.Message("networkConn.flushloop"),
					metrics.With("network", nc.network.ID),
				)

				// Dealing with slow consumer, close it.
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					nc.network.Metrics.Emit(
						metrics.Error(err),
						metrics.WithID(nc.id),
						metrics.Message("Connection seems to be a close consumer: closing"),
						metrics.With("network", nc.network.ID),
					)

					nc.closeConnection()
				}
			}
		}
	}
}

func (nc *networkConn) processMessage(data []byte) {
	nc.scratch.Write(data)

	for nc.scratch.Len() > 0 {
		nextdata := nc.scratch.Next(2)
		if len(nextdata) < 2 {
			nc.scratch.Write(nextdata)
			return
		}

		nextSize := int(binary.BigEndian.Uint16(nextdata))

		// If scratch is zero and we do have count data, maybe we face a unfinished write.
		if nc.scratch.Len() == 0 {
			nc.scratch.Write(nextdata)
			return
		}

		if nextSize > nc.scratch.Len() {
			rest := nc.scratch.Bytes()
			restruct := append(nextdata, rest...)
			nc.scratch.Reset()
			nc.scratch.Write(restruct)
			return
		}

		next := nc.scratch.Next(nextSize)
		nc.addMessage(&message{data: next})
	}
}

// readLoop handles the necessary operation of reading data from the
// underline connection.
func (nc *networkConn) readLoop() {
	defer nc.closeConnection()
	defer nc.worker.Done()

	nc.mu.Lock()
	if nc.conn == nil {
		return
	}
	cn := nc.conn
	nc.mu.Unlock()

	incoming := make([]byte, minBufferSize, maxBufferSize)

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
		nc.processMessage(incoming[:n])

		// Lets shrink buffer abit within area.
		if n == len(incoming) && n < maxBufferSize {
			incoming = incoming[0 : minBufferSize*2]
		}

		if n < len(incoming)/2 && len(incoming) > minBufferSize {
			incoming = incoming[0 : len(incoming)/2]
		}

		if n > len(incoming) && len(incoming) > minBufferSize && n < maxBufferSize {
			incoming = incoming[0 : maxBufferSize/2]
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

	stream, err := mlisten.Listen("tcp", n.Addr, n.TLS)
	if err != nil {
		return err
	}

	n.ctx = ctx
	n.pool = make(chan func(), 0)
	n.clients = make(map[string]*networkConn)

	if n.ClientInitialWriteSize <= 0 {
		n.ClientInitialWriteSize = clientMinInitialBuffer
	}

	if n.ClientMaxWriteDeadline <= 0 {
		n.ClientMaxWriteDeadline = clientWriteDeadline
	}

	if n.ClientMaxWriteSize <= 0 {
		n.ClientMaxWriteSize = clientMaxBuffer
	}

	n.routines.Add(2)
	go n.runStream(stream)
	go n.endLogic(ctx, stream)

	return nil
}

func (n *Network) endLogic(ctx context.CancelContext, stream melon.ConnReadWriteCloser) {
	defer n.routines.Done()

	<-ctx.Done()
	for _, conn := range n.clients {
		conn.closeConnection()
	}
	stream.Close()
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
			ID:         id,
			NID:        n.ID,
			Metrics:    n.Metrics,
			LocalAddr:  conn.localAddr,
			RemoteAddr: conn.remoteAddr,
		}
		client.WriteFunc = conn.write
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

	initial := minSleep

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

				if initial >= maxSleep {
					initial = minSleep
				}
			}

			continue
		}

		go func(conn net.Conn) {
			uuid := uuid.NewV4().String()

			client := mnet.Client{
				ID:         uuid,
				NID:        n.ID,
				LocalAddr:  conn.LocalAddr(),
				RemoteAddr: conn.RemoteAddr(),
				Metrics:    n.Metrics,
			}

			n.Metrics.Emit(
				metrics.WithID(n.ID),
				metrics.With("network", n.ID),
				metrics.With("client_id", uuid),
				metrics.Info("New Client Connection"),
				metrics.With("local_addr", client.LocalAddr),
				metrics.With("remote_addr", client.RemoteAddr),
			)

			cn := new(networkConn)
			cn.id = uuid
			cn.ctx = n.ctx
			cn.network = n
			cn.conn = conn
			cn.localAddr = client.LocalAddr
			cn.remoteAddr = client.RemoteAddr
			cn.flushr = make(chan struct{}, 10)
			cn.closer = make(chan struct{}, 0)
			cn.buffWriter = mnet.NewBufferedIntervalWriter(conn, n.ClientMaxWriteSize, n.ClientMaxWriteDeadline)
			cn.bw = mnet.NewSizeAppenBuffereddWriter(cn.buffWriter, n.ClientInitialWriteSize)

			client.ReaderFunc = cn.read
			client.WriteFunc = cn.write
			client.CloseFunc = cn.closeConn
			client.FlushFunc = cn.flushAll
			client.SiblingsFunc = n.getOtherClients

			cn.worker.Add(2)

			go cn.readLoop()
			go cn.flushloop()

			n.cu.Lock()
			n.clients[uuid] = cn
			n.cu.Unlock()

			if err := n.Handler(client); err != nil {
				client.Close()
			}
		}(newConn)
	}
}
