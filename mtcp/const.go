package mtcp

import "time"

const (
	// MinTemporarySleep sets the minimum, initial sleep a network should
	// take when facing a Temporary net error.
	MinTemporarySleep = 10 * time.Millisecond

	// MaxTemporarySleep sets the maximum, allowed sleep a network should
	// take when facing a Temporary net error.
	MaxTemporarySleep = 1 * time.Second

	// MaxFlushDeadline sets the maximum, allowed duration for flushing data
	MaxFlushDeadline = 3 * time.Second

	// MinBufferSize sets the initial size of space of the slice
	// used to read in content from a net.Conn in the connections
	// read loop.
	MinBufferSize = 512

	// MaxBufferSize sets the maximum size allowed for all reads
	// used in the readloop of a client's net.Conn.
	MaxBufferSize = 69560

	// DefaultDialTimeout sets the default maximum time in seconds allowed before
	// a net.Dialer exits attempt to dial a network.
	DefaultDialTimeout = 3 * time.Second

	// DefaultKeepAlive sets the default maximum time to keep alive a tcp connection
	// during no-use. It is used by net.Dialer.
	DefaultKeepAlive = 3 * time.Minute

	// DefaultReconnectBufferSize sets the size of the buffer during reconnection.
	DefaultReconnectBufferSize = 1024 * 1024 * 8
)
