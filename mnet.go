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

// SiblingsFunc defines a function which returns a list of sibling funcs.
type SiblingsFunc func(Client) ([]Client, error)

// ClientFunc defines a function type which receives a Client type and
// returns a possible error.
type ClientFunc func(Client) error

// errors ...
var (
	ErrReadNotAllowed     = errors.New("reading not allowed")
	ErrWriteNotAllowed    = errors.New("data writing not allowed")
	ErrCloseNotAllowed    = errors.New("closing not allowed")
	ErrFlushNotAllowed    = errors.New("write flushing not allowed")
	ErrSiblingsNotAllowed = errors.New("siblings retrieval not allowed")
)

// Client holds a given information regarding a given network connection.
type Client struct {
	ID           string
	NID          string
	LocalAddr    net.Addr
	RemoteAddr   net.Addr
	Metrics      metrics.Metrics
	ReaderFunc   ReaderFunc
	WriteFunc    WriteFunc
	FlushFunc    ClientFunc
	CloseFunc    ClientFunc
	SiblingsFunc SiblingsFunc
}

// Read reads the underline data into the provided slice.
func (c Client) Read() ([]byte, error) {
	defer c.Metrics.Emit(
		metrics.Message("Client.Read"),
		metrics.WithID(c.ID),
		metrics.With("network", c.NID),
	)

	if c.ReaderFunc == nil {
		return nil, ErrReadNotAllowed
	}

	return c.ReaderFunc(c)
}

// Flush sends all accumulated message within clients buffer into
// connection.
func (c Client) Flush() error {
	defer c.Metrics.Emit(
		metrics.Message("Client.Flush"),
		metrics.WithID(c.ID),
		metrics.With("network", c.NID),
	)

	if c.FlushFunc == nil {
		return ErrFlushNotAllowed
	}

	return c.FlushFunc(c)
}

// Write writes provided data into connection without any deadline.
func (c Client) Write(data []byte) (int, error) {
	defer c.Metrics.Emit(
		metrics.WithID(c.ID),
		metrics.Message("Client.Write"),
		metrics.With("network", c.NID),
	)

	if c.WriteFunc == nil {
		return 0, ErrWriteNotAllowed
	}

	return c.WriteFunc(c, data)
}

// Close closes the underline client connection.
func (c Client) Close() error {
	defer c.Metrics.Emit(
		metrics.WithID(c.ID),
		metrics.Message("Client.Close"),
		metrics.With("network", c.NID),
	)

	if c.CloseFunc == nil {
		return ErrCloseNotAllowed
	}

	return c.CloseFunc(c)
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
