package melon

// UInt64UniqueHash defines a unique hash for UInt64 which can
// be used to reference a given instance within a context.ValueBag or a google context.Context
// value store.
const UInt64UniqueHash = "9dfe93d26e38e94f8348337651317c3130bb021d"

// UInt64Reader defines reader for uint64 type.
type UInt64Reader interface {
	ReadUInt64() (uint64, error)
}

// UInt64ReadCloser defines reader and closer for uint64 type.
type UInt64ReadCloser interface {
	Closer
	UInt64Reader
}

// UInt64StreamReader defines reader uint64 type.
type UInt64StreamReader interface {
	Read(int) ([]uint64, error)
}

// UInt64StreamReadCloser defines reader and closer for uint64 type.
type UInt64StreamReadCloser interface {
	Closer
	UInt64StreamReader
}

// UInt64Writer defines writer for uint64 type.
type UInt64Writer interface {
	WriteUInt64(uint64) error
}

// UInt64WriteCloser defines writer and closer for uint64 type.
type UInt64WriteCloser interface {
	Closer
	UInt64Writer
}

// UInt64StreamWrite defines writer for uint64 type.
type UInt64StreamWriter interface {
	Write([]uint64) (int, error)
}

// UInt64StreamWriteCloser defines writer and closer for uint64 type.
type UInt64StreamWriteCloser interface {
	Closer
	UInt64StreamWriter
}

// UInt64ReadWriteCloser composes reader types with closer for uint64.
// with associated close method.
type UInt64ReadWriteCloser interface {
	Closer
	UInt64Reader
	UInt64Writer
}

// UInt64StreamReadWriteCloser composes stream types with closer for uint64.
type UInt64Stream interface {
	Closer
	UInt64StreamReader
	UInt64StreamWriter
}
