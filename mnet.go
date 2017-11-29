package mnet

import (
	"errors"
	"net"

	"github.com/influx6/faux/metrics"
)

// ConnHandler defines a function which will process a incoming net.Conn,
// else if error is returned then the net.Conn is closed.
type ConnHandler func(Client) error

// ReaderFunc defines a function which takes giving incoming Client and returns associated
// data and error.
type ReaderFunc func(Client) ([]byte, error)

// WriteFunc defines a function type which takes a client and writes out data.
type WriteFunc func(Client, []byte) (int, error)

// ClientSiblingsFunc defines a function which returns a list of sibling funcs.
type ClientSiblingsFunc func(Client) ([]Client, error)

// ClientFunc defines a function type which receives a Client type and
// returns a possible error.
type ClientFunc func(Client) error

// ClientAddrFunc returns a net.Addr associated with a given client else
// an error.
type ClientAddrFunc func(Client) (net.Addr, error)

// ClientExpandBufferFunc expands the internal client collecting buffer.
type ClientExpandBufferFunc func(Client, int) error

// ClientReconnectionFunc defines a function type which receives a Client pointer type and
// is responsible for the reconnection of the client client connection.
type ClientReconnectionFunc func(Client, string) error

// ClientStatisticsFunc defines a function type which returns a Statistics
// structs related to the user.
type ClientStatisticsFunc func(Client) (Statistics, error)

// errors ...
var (
	ErrNoHostNameInAddr              = errors.New("addr must have hostname")
	ErrStillConnected                = errors.New("connection still active")
	ErrAlreadyClosed                 = errors.New("already closed connection")
	ErrReadNotAllowed                = errors.New("reading not allowed")
	ErrWriteNotAllowed               = errors.New("data writing not allowed")
	ErrCloseNotAllowed               = errors.New("closing not allowed")
	ErrBufferExpansionNotAllowed     = errors.New("buffer expansion not allowed")
	ErrFlushNotAllowed               = errors.New("write flushing not allowed")
	ErrSiblingsNotAllowed            = errors.New("siblings retrieval not allowed")
	ErrStatisticsNotProvided         = errors.New("statistics not provided")
	ErrAddrNotProvided               = errors.New("net.Addr is not provided")
	ErrClientReconnectionUnavailable = errors.New("client reconnection not available")
)

// Statistics defines a struct ment to hold granular information regarding
// network activities.
type Statistics struct {
	TotalWrittenMessages int64
	TotalReadMessages    int64
	TotalWrittenInBytes  int64
	TotalReadInBytes     int64
	TotalFlushedInBytes  int64
	TotalClients         int64
	TotalClientsClosed   int64
	TotalReconnects      int64
}

// Client holds a given information regarding a given network connection.
type Client struct {
	ID               string
	NID              string
	Metrics          metrics.Metrics
	LocalAddrFunc    ClientAddrFunc
	RemoteAddrFunc   ClientAddrFunc
	ReaderFunc       ReaderFunc
	WriteFunc        WriteFunc
	FlushFunc        ClientFunc
	CloseFunc        ClientFunc
	SiblingsFunc     ClientSiblingsFunc
	StatisticFunc    ClientStatisticsFunc
	ReconnectionFunc ClientReconnectionFunc
}

// LocalAddr returns local address associated with given client.
func (c Client) LocalAddr() (net.Addr, error) {
	c.Metrics.Emit(
		metrics.WithID(c.ID),
		metrics.With("network", c.NID),
		metrics.Message("Client.LocalAddr"),
	)

	if c.LocalAddrFunc == nil {
		return nil, ErrAddrNotProvided
	}

	addr, err := c.LocalAddrFunc(c)
	if err != nil {
		c.Metrics.Emit(
			metrics.Error(err),
			metrics.WithID(c.ID),
			metrics.With("network", c.NID),
			metrics.Message("Client.LocalAddr"),
		)
		return nil, err
	}

	return addr, nil
}

// RemoteAddr returns remote address associated with given client.
func (c Client) RemoteAddr() (net.Addr, error) {
	c.Metrics.Emit(
		metrics.WithID(c.ID),
		metrics.With("network", c.NID),
		metrics.Message("Client.RemoteAddr"),
	)

	if c.RemoteAddrFunc == nil {
		return nil, ErrAddrNotProvided
	}

	addr, err := c.RemoteAddrFunc(c)
	if err != nil {
		c.Metrics.Emit(
			metrics.Error(err),
			metrics.WithID(c.ID),
			metrics.With("network", c.NID),
			metrics.Message("Client.RemoteAddr"),
		)
		return nil, err
	}

	return addr, nil
}

// Reconnect attempts to reconnect with external endpoint.
// Also allows provision of alternate address to reconnect with.
func (c Client) Reconnect(altAddr string) error {
	c.Metrics.Emit(
		metrics.WithID(c.ID),
		metrics.Message("Client.Reconnect"),
		metrics.With("network", c.NID),
		metrics.With("alternate_addr", altAddr),
	)
	if c.ReconnectionFunc == nil {
		return ErrClientReconnectionUnavailable
	}

	if err := c.ReconnectionFunc(c, altAddr); err != nil {
		c.Metrics.Emit(
			metrics.WithID(c.ID),
			metrics.Message("Client.Reconnect"),
			metrics.Error(err),
			metrics.With("alternate_addr", altAddr),
			metrics.With("network", c.NID),
		)
		return err
	}

	return nil
}

// Read reads the underline data into the provided slice.
func (c Client) Read() ([]byte, error) {
	if c.ReaderFunc == nil {
		return nil, ErrReadNotAllowed
	}

	return c.ReaderFunc(c)
}

// Write writes provided data into connection without any deadline.
func (c Client) Write(data []byte) (int, error) {
	if c.WriteFunc == nil {
		return 0, ErrWriteNotAllowed
	}

	return c.WriteFunc(c, data)
}

// Flush sends all accumulated message within clients buffer into
// connection.
func (c Client) Flush() error {
	if c.FlushFunc == nil {
		return ErrFlushNotAllowed
	}

	return c.FlushFunc(c)
}

// Statistics returns statistics associated with client.j
func (c Client) Statistics() (Statistics, error) {
	c.Metrics.Emit(
		metrics.WithID(c.ID),
		metrics.Message("Client.Statistics"),
		metrics.With("network", c.NID),
	)

	if c.StatisticFunc == nil {
		return Statistics{}, ErrStatisticsNotProvided
	}

	return c.StatisticFunc(c)
}

// Close closes the underline client connection.
func (c Client) Close() error {
	c.Metrics.Emit(
		metrics.WithID(c.ID),
		metrics.Message("Client.Close"),
		metrics.With("network", c.NID),
	)

	if c.CloseFunc == nil {
		return ErrCloseNotAllowed
	}

	if err := c.CloseFunc(c); err != nil {
		c.Metrics.Emit(
			metrics.WithID(c.ID),
			metrics.Message("Client.Close"),
			metrics.Error(err),
			metrics.With("network", c.NID),
		)
		return err
	}

	return nil
}

// Others returns other client associated with the source of this client.
func (c Client) Others() ([]Client, error) {
	defer c.Metrics.Emit(
		metrics.WithID(c.ID),
		metrics.Message("Client.Others"),
		metrics.With("network", c.NID),
	)

	if c.SiblingsFunc == nil {
		return nil, ErrSiblingsNotAllowed
	}

	return c.SiblingsFunc(c)
}
