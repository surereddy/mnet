package melon

import "net"

// ConnUniqueHash defines a unique hash for Conn which can
// be used to reference a given instance within a context.ValueBag or a google context.Context
// value store.
const ConnUniqueHash = "e227c185e376f84717e05beb52bde64ab47d6838"

// ConnReader defines reader for net.Conn type.
type ConnReader interface {
	ReadConn() (net.Conn, error)
}

// ConnReadCloser defines reader and closer for net.Conn type.
type ConnReadCloser interface {
	Closer
	ConnReader
}

// ConnStreamReader defines reader net.Conn type.
type ConnStreamReader interface {
	Read(int) ([]net.Conn, error)
}

// ConnStreamReadCloser defines reader and closer for net.Conn type.
type ConnStreamReadCloser interface {
	Closer
	ConnStreamReader
}

// ConnWriter defines writer for net.Conn type.
type ConnWriter interface {
	WriteConn(net.Conn) error
}

// ConnWriteCloser defines writer and closer for net.Conn type.
type ConnWriteCloser interface {
	Closer
	ConnWriter
}

// ConnStreamWrite defines writer for net.Conn type.
type ConnStreamWriter interface {
	Write([]net.Conn) (int, error)
}

// ConnStreamWriteCloser defines writer and closer for net.Conn type.
type ConnStreamWriteCloser interface {
	Closer
	ConnStreamWriter
}

// ConnReadWriteCloser composes reader types with closer for net.Conn.
// with associated close method.
type ConnReadWriteCloser interface {
	Closer
	ConnReader
	ConnWriter
}

// ConnStreamReadWriteCloser composes stream types with closer for net.Conn.
type ConnStream interface {
	Closer
	ConnStreamReader
	ConnStreamWriter
}
