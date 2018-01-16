package melon

// ByteUniqueHash defines a unique hash for Byte which can
// be used to reference a given instance within a context.ValueBag or a google context.Context
// value store.
const ByteUniqueHash = "9f2e300d92751ef08344907e4adc0481d7f9ae93"

// ByteReader defines reader for byte type.
type ByteReader interface {
	ReadByte() (byte, error)
}

// ByteReadCloser defines reader and closer for byte type.
type ByteReadCloser interface {
	Closer
	ByteReader
}

// ByteStreamReader defines reader byte type.
type ByteStreamReader interface {
	Read(int) ([]byte, error)
}

// ByteStreamReadCloser defines reader and closer for byte type.
type ByteStreamReadCloser interface {
	Closer
	ByteStreamReader
}

// ByteWriter defines writer for byte type.
type ByteWriter interface {
	WriteByte(byte) error
}

// ByteWriteCloser defines writer and closer for byte type.
type ByteWriteCloser interface {
	Closer
	ByteWriter
}

// ByteStreamWrite defines writer for byte type.
type ByteStreamWriter interface {
	Write([]byte) (int, error)
}

// ByteStreamWriteCloser defines writer and closer for byte type.
type ByteStreamWriteCloser interface {
	Closer
	ByteStreamWriter
}

// ByteReadWriteCloser composes reader types with closer for byte.
// with associated close method.
type ByteReadWriteCloser interface {
	Closer
	ByteReader
	ByteWriter
}

// ByteStreamReadWriteCloser composes stream types with closer for byte.
type ByteStream interface {
	Closer
	ByteStreamReader
	ByteStreamWriter
}
