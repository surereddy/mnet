package mnet

import (
	"errors"
	"io"
	"net"

	"github.com/influx6/faux/metrics"
)

// ConnHandler defines a function which will process a incoming net.Conn,
// else if error is returned then the net.Conn is closed.
type ConnHandler func(Client) error

// ReaderFunc defines a function which takes giving incoming Client and returns associated
// data and error.
type ReaderFunc func(Client) ([]byte, error)

// ReaderFromFunc defines a function which takes giving incoming Client and returns associated
// data and error.
type ReaderFromFunc func(Client) ([]byte, net.Addr, error)

// WriteToFunc defines a function type which takes a client and writes out data.
type WriteToFunc func(Client, net.Addr, int) (io.WriteCloser, error)

// WriteFunc defines a function type which takes a client and writes out data.
type WriteFunc func(Client, int) (io.WriteCloser, error)

// ClientSiblingsFunc defines a function which returns a list of sibling funcs.
type ClientSiblingsFunc func(Client) ([]Client, error)

// ClientFunc defines a function type which receives a Client type and
// returns a possible error.
type ClientFunc func(Client) error

// ClientWithAddrFunc defines a function type which receives a Client type and
// and net.Addr and returns a possible error.
type ClientWithAddrFunc func(Client, net.Addr) error

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
type ClientStatisticsFunc func(Client) (ClientStatistic, error)

// errors ...
var (
	ErrNoDataYet                     = errors.New("data is not yet available for reading")
	ErrNoHostNameInAddr              = errors.New("addr must have hostname")
	ErrStillConnected                = errors.New("connection still active")
	ErrAlreadyClosed                 = errors.New("already closed connection")
	ErrReadNotAllowed                = errors.New("reading not allowed")
	ErrReadFromNotAllowed            = errors.New("reading from a addr not allowed")
	ErrLiveCheckNotAllowed           = errors.New("live status not allowed or supported")
	ErrWriteNotAllowed               = errors.New("data writing not allowed")
	ErrWriteToAddrNotAllowed         = errors.New("data writing to a addr not allowed")
	ErrCloseNotAllowed               = errors.New("closing not allowed")
	ErrBufferExpansionNotAllowed     = errors.New("buffer expansion not allowed")
	ErrFlushNotAllowed               = errors.New("write flushing not allowed")
	ErrFlushToAddrNotAllowed         = errors.New("write flushing to target addr not allowed")
	ErrSiblingsNotAllowed            = errors.New("siblings retrieval not allowed")
	ErrStatisticsNotProvided         = errors.New("statistics not provided")
	ErrAddrNotProvided               = errors.New("net.Addr is not provided")
	ErrClientReconnectionUnavailable = errors.New("client reconnection not available")
)

// NetworkStatistic defines a struct ment to hold granular information regarding
// network activities.
type NetworkStatistic struct {
	ID           string
	LocalAddr    net.Addr
	RemoteAddr   net.Addr
	TotalClients int64
	TotalClosed  int64
	TotalOpened  int64
	TotalActive  int64
}

// ClientStatistic defines a struct ment to hold granular information regarding
// network activities.
type ClientStatistic struct {
	ID              string
	Local           net.Addr
	Remote          net.Addr
	MessagesRead    int64
	MessagesWritten int64
	BytesWritten    int64
	BytesRead       int64
	BytesFlushed    int64
	Reconnects      int64
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
	CloseFunc        ClientFunc
	LiveFunc         ClientFunc
	FlushFunc        ClientFunc
	FlushAddrFunc    ClientWithAddrFunc
	SiblingsFunc     ClientSiblingsFunc
	StatisticFunc    ClientStatisticsFunc
	ReconnectionFunc ClientReconnectionFunc
	//ReaderFromFunc   ReaderFromFunc
	//WriteToFunc      WriteToFunc
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

// Live returns an error if client is not currently live or connected to
// network. Allows to know current status of client.
func (c Client) Live() error {
	c.Metrics.Emit(
		metrics.WithID(c.ID),
		metrics.Message("Client.Live"),
		metrics.With("network", c.NID),
	)
	if c.LiveFunc == nil {
		return ErrLiveCheckNotAllowed
	}

	if err := c.LiveFunc(c); err != nil {
		c.Metrics.Emit(
			metrics.Error(err),
			metrics.WithID(c.ID),
			metrics.Message("Client.Live"),
			metrics.With("network", c.NID),
		)
		return err
	}

	return nil
}

// Reconnect attempts to reconnect with external endpoint.
// Also allows provision of alternate address to reconnect with.
// NOTE: Not all may implement this has it's optional.
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

// ReadFrom reads the underline data into the provided connection
// returning senders address and data.
// NOTE: Not all may implement this has it's optional.
//func (c Client) ReadFrom() ([]byte, net.Addr, error) {
//	if c.ReaderFromFunc == nil {
//		return nil, nil, ErrReadFromNotAllowed
//	}
//
//	return c.ReaderFromFunc(c)
//}

// WriteTo writes provided data into connection targeting giving address.
// NOTE: Not all may implement this has it's optional.
//func (c Client) WriteTo(addr net.Addr, toWriteSize int) (io.WriteCloser, error) {
//	if c.WriteFunc == nil {
//		return nil, ErrWriteToAddrNotAllowed
//	}
//
//	return c.WriteToFunc(c, addr, toWriteSize)
//}

// Read reads the underline data into the provided slice.
func (c Client) Read() ([]byte, error) {
	if c.ReaderFunc == nil {
		return nil, ErrReadNotAllowed
	}

	return c.ReaderFunc(c)
}

// Write writes provided data into connection without any deadline.
func (c Client) Write(toWriteSize int) (io.WriteCloser, error) {
	if c.WriteFunc == nil {
		return nil, ErrWriteNotAllowed
	}

	return c.WriteFunc(c, toWriteSize)
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
func (c Client) Statistics() (ClientStatistic, error) {
	c.Metrics.Emit(
		metrics.WithID(c.ID),
		metrics.Message("Client.Statistics"),
		metrics.With("network", c.NID),
	)

	if c.StatisticFunc == nil {
		return ClientStatistic{}, ErrStatisticsNotProvided
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
// NOTE: Not all may implement this has it's optional.
// WARNING: And those who implement this method must ensure the Read method do not
// exists, to avoid conflict of internal behaviour. More so, only the
// server client handler should ever have access to Read/ReadFrom methods.
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
