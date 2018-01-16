package melon

// UInt32UniqueHash defines a unique hash for UInt32 which can
// be used to reference a given instance within a context.ValueBag or a google context.Context
// value store.
const UInt32UniqueHash = "a334cf504c1433f88e8aaab77d2225c3f7843634"

// UInt32Reader defines reader for uint32 type.
type UInt32Reader interface {
	ReadUInt32() (uint32, error)
}

// UInt32ReadCloser defines reader and closer for uint32 type.
type UInt32ReadCloser interface {
	Closer
	UInt32Reader
}

// UInt32StreamReader defines reader uint32 type.
type UInt32StreamReader interface {
	Read(int) ([]uint32, error)
}

// UInt32StreamReadCloser defines reader and closer for uint32 type.
type UInt32StreamReadCloser interface {
	Closer
	UInt32StreamReader
}

// UInt32Writer defines writer for uint32 type.
type UInt32Writer interface {
	WriteUInt32(uint32) error
}

// UInt32WriteCloser defines writer and closer for uint32 type.
type UInt32WriteCloser interface {
	Closer
	UInt32Writer
}

// UInt32StreamWrite defines writer for uint32 type.
type UInt32StreamWriter interface {
	Write([]uint32) (int, error)
}

// UInt32StreamWriteCloser defines writer and closer for uint32 type.
type UInt32StreamWriteCloser interface {
	Closer
	UInt32StreamWriter
}

// UInt32ReadWriteCloser composes reader types with closer for uint32.
// with associated close method.
type UInt32ReadWriteCloser interface {
	Closer
	UInt32Reader
	UInt32Writer
}

// UInt32StreamReadWriteCloser composes stream types with closer for uint32.
type UInt32Stream interface {
	Closer
	UInt32StreamReader
	UInt32StreamWriter
}
