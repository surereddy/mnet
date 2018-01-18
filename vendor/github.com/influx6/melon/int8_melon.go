package melon

// Int8UniqueHash defines a unique hash for Int8 which can
// be used to reference a given instance within a context.ValueBag or a google context.Context
// value store.
const Int8UniqueHash = "da945acc637aec1ea9d814cc20b2ed6e794ef8b1"

// Int8Reader defines reader for int8 type.
type Int8Reader interface {
	ReadInt8() (int8, error)
}

// Int8ReadCloser defines reader and closer for int8 type.
type Int8ReadCloser interface {
	Closer
	Int8Reader
}

// Int8StreamReader defines reader int8 type.
type Int8StreamReader interface {
	Read(int) ([]int8, error)
}

// Int8StreamReadCloser defines reader and closer for int8 type.
type Int8StreamReadCloser interface {
	Closer
	Int8StreamReader
}

// Int8Writer defines writer for int8 type.
type Int8Writer interface {
	WriteInt8(int8) error
}

// Int8WriteCloser defines writer and closer for int8 type.
type Int8WriteCloser interface {
	Closer
	Int8Writer
}

// Int8StreamWrite defines writer for int8 type.
type Int8StreamWriter interface {
	Write([]int8) (int, error)
}

// Int8StreamWriteCloser defines writer and closer for int8 type.
type Int8StreamWriteCloser interface {
	Closer
	Int8StreamWriter
}

// Int8ReadWriteCloser composes reader types with closer for int8.
// with associated close method.
type Int8ReadWriteCloser interface {
	Closer
	Int8Reader
	Int8Writer
}

// Int8StreamReadWriteCloser composes stream types with closer for int8.
type Int8Stream interface {
	Closer
	Int8StreamReader
	Int8StreamWriter
}
