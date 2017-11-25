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
	minSleep      = 10 * time.Millisecond
	maxSleep      = 2 * time.Second
	maxActorWait  = 10 * time.Second
	maxWorkWait   = 500 * time.Millisecond
	writeDeadline = 3 * time.Second
	maxWriteSize  = (1024 * 4)
	maxReadSize   = (1024 * 4)
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
	id       string
	ctx      context.CancelContext
	lastRead int
	flush    chan struct{}
	close    chan struct{}
	worker   sync.WaitGroup

	su      sync.Mutex
	scratch bytes.Buffer

	network *Network

	mu   sync.Mutex
	Err  error
	conn net.Conn
	head *message
	tail *message
	bu   sync.Mutex
	bw   *bytes.Buffer
}

func (nc *networkConn) write(cm mnet.Client, data []byte) (int, error) {

	defer nc.network.Metrics.Emit(
		metrics.Message("networkConn.write"),

		metrics.WithID(nc.id),
		metrics.With("network_id", nc.network.ID),
	)

	nc.mu.Lock()
	if nc.Err != nil {
		nc.mu.Unlock()
		return 0, nc.Err
	}
	nc.mu.Unlock()

	nc.bu.Lock()
	defer nc.bu.Unlock()
	return nc.bw.Write(data)
}

func (nc *networkConn) closeConn(cm mnet.Client) error {
	nc.mu.Lock()
	if nc.close == nil {
		nc.mu.Unlock()
		return nil
	}
	nc.mu.Unlock()

	nc.network.cu.Lock()
	delete(nc.network.clients, nc.id)
	nc.network.cu.Unlock()

	nc.mu.Lock()
	select {
	case nc.close <- struct{}{}:
	default:
	}
	err := nc.conn.Close()
	nc.mu.Unlock()

	nc.worker.Wait()

	nc.mu.Lock()
	nc.close = nil
	nc.conn = nil
	nc.mu.Unlock()

	nc.bu.Lock()
	nc.bw = nil
	nc.bu.Unlock()

	return err
}

func (nc *networkConn) flushAll(cm mnet.Client) error {
	if len(nc.flush) == 0 {
		select {
		case nc.flush <- struct{}{}:
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
		case <-nc.close:
			return
		case _, ok := <-nc.flush:
			if !ok {
				return
			}

			nc.mu.Lock()
			cn := nc.conn
			nc.mu.Unlock()

			nc.bu.Lock()
			data := nc.bw.Bytes()
			nc.bw.Reset()
			nc.bu.Unlock()

			header := make([]byte, 2)
			binary.BigEndian.PutUint16(header, uint16(len(data)))
			header = append(header, data...)

			if _, err := cn.Write(header); err != nil {
				nc.network.Metrics.Emit(
					metrics.Error(err),
					metrics.WithID(nc.id),
					metrics.Message("networkConn.flushloop"),
					metrics.With("network", nc.network.ID),
				)
			}
		}
	}
}

func (nc *networkConn) processMessage(data []byte) {
	nc.su.Lock()
	defer nc.su.Unlock()

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
		nc.addMessage(&message{data: next[:nextSize]})
	}
}

// readLoop handles the necessary operation of reading data from the
// underline connection.
func (nc *networkConn) readLoop() {
	nc.lastRead = 512
	incoming := make([]byte, 512)

	defer nc.worker.Done()

	nc.mu.Lock()
	cn := nc.conn
	nc.mu.Unlock()

	for {
		n, err := cn.Read(incoming)
		if err != nil {
			nc.network.Metrics.Emit(
				metrics.Error(err),
				metrics.Message("Connection failed to read"),
				metrics.WithID(nc.id),
				metrics.With("network", nc.network.ID),
			)

			nc.mu.Lock()
			nc.Err = err
			nc.mu.Unlock()
			return
		}

		// if nothing was read, skip.
		if n == 0 && len(incoming) == 0 {
			continue
		}

		// Send into go-routine (critical path)?
		go nc.processMessage(incoming[:n])

		if n > nc.lastRead && n < 512 {
			incoming = make([]byte, 512)
		}

		if n > nc.lastRead && n > 512 {
			incoming = make([]byte, n)
		}

		if n < nc.lastRead && n > 512 {
			incoming = make([]byte, nc.lastRead/2)
		}

		if n < nc.lastRead && n < 512 {
			incoming = make([]byte, nc.lastRead)
		}

		nc.lastRead = n
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

	// MaxWorkerWorkWait defines max time for worker to wait actively for working before dying out.
	MaxWorkerWorkWait time.Duration

	// MaxWorkWait defines the max time to wait for work to be accepted else spawn new worker for work.
	MaxWorkWait time.Duration

	// MaxWriteDeadline defines max deadline to be waited for, for clients conn to write data out.
	MaxWriteDeadline time.Duration

	MaxReaderSize int64
	MaxWriterSize int64

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

	if n.MaxWorkWait <= 0 {
		n.MaxWorkWait = maxWorkWait
	}

	if n.MaxWorkerWorkWait <= 0 {
		n.MaxWorkerWorkWait = maxActorWait
	}

	if n.MaxReaderSize <= 0 {
		n.MaxReaderSize = maxReadSize
	}

	if n.MaxWriterSize <= 0 {
		n.MaxWriterSize = maxWriteSize
	}

	if n.MaxWriteDeadline <= 0 {
		n.MaxWriteDeadline = writeDeadline
	}

	n.routines.Add(2)
	go n.runStream(stream)
	go n.endLogic(ctx, stream)

	return nil
}

func (n *Network) endLogic(ctx context.CancelContext, stream melon.ConnReadWriteCloser) {
	defer n.routines.Done()
	<-ctx.Done()

	for id, conn := range n.clients {
		conn.closeConn(mnet.Client{ID: id})
	}

	stream.Close()
}

// Wait is called to ensure network ended.
func (n *Network) Wait() {
	n.routines.Wait()
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
		conn, err := stream.ReadConn()
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
		cn.bw = bytes.NewBuffer(nil)
		cn.flush = make(chan struct{}, 10)
		cn.close = make(chan struct{}, 0)
		client.ReaderFunc = cn.read
		client.WriteFunc = cn.write
		client.CloseFunc = cn.closeConn
		client.FlushFunc = cn.flushAll

		cn.worker.Add(2)
		go cn.readLoop()
		go cn.flushloop()

		n.cu.Lock()
		n.clients[uuid] = cn
		n.cu.Unlock()

		go func() {
			if err := n.Handler(client); err != nil {
				client.Close()
			}
		}()
	}
}
