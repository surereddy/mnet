package melon

// UInt8UniqueHash defines a unique hash for UInt8 which can
// be used to reference a given instance within a context.ValueBag or a google context.Context
// value store.
const UInt8UniqueHash = "e5de7d77d24ebbe134934f7c5f8dc3fafb344aba"

// UInt8Reader defines reader for uint8 type.
type UInt8Reader interface {
	ReadUInt8() (uint8, error)
}

// UInt8ReadCloser defines reader and closer for uint8 type.
type UInt8ReadCloser interface {
	Closer
	UInt8Reader
}

// UInt8StreamReader defines reader uint8 type.
type UInt8StreamReader interface {
	Read(int) ([]uint8, error)
}

// UInt8StreamReadCloser defines reader and closer for uint8 type.
type UInt8StreamReadCloser interface {
	Closer
	UInt8StreamReader
}

// UInt8Writer defines writer for uint8 type.
type UInt8Writer interface {
	WriteUInt8(uint8) error
}

// UInt8WriteCloser defines writer and closer for uint8 type.
type UInt8WriteCloser interface {
	Closer
	UInt8Writer
}

// UInt8StreamWrite defines writer for uint8 type.
type UInt8StreamWriter interface {
	Write([]uint8) (int, error)
}

// UInt8StreamWriteCloser defines writer and closer for uint8 type.
type UInt8StreamWriteCloser interface {
	Closer
	UInt8StreamWriter
}

// UInt8ReadWriteCloser composes reader types with closer for uint8.
// with associated close method.
type UInt8ReadWriteCloser interface {
	Closer
	UInt8Reader
	UInt8Writer
}

// UInt8StreamReadWriteCloser composes stream types with closer for uint8.
type UInt8Stream interface {
	Closer
	UInt8StreamReader
	UInt8StreamWriter
}
