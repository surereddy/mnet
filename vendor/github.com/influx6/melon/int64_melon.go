package melon

// Int64UniqueHash defines a unique hash for Int64 which can
// be used to reference a given instance within a context.ValueBag or a google context.Context
// value store.
const Int64UniqueHash = "d9735f419182b6c737eba2e4b2d3f05a5b349bc9"

// Int64Reader defines reader for int64 type.
type Int64Reader interface {
	ReadInt64() (int64, error)
}

// Int64ReadCloser defines reader and closer for int64 type.
type Int64ReadCloser interface {
	Closer
	Int64Reader
}

// Int64StreamReader defines reader int64 type.
type Int64StreamReader interface {
	Read(int) ([]int64, error)
}

// Int64StreamReadCloser defines reader and closer for int64 type.
type Int64StreamReadCloser interface {
	Closer
	Int64StreamReader
}

// Int64Writer defines writer for int64 type.
type Int64Writer interface {
	WriteInt64(int64) error
}

// Int64WriteCloser defines writer and closer for int64 type.
type Int64WriteCloser interface {
	Closer
	Int64Writer
}

// Int64StreamWrite defines writer for int64 type.
type Int64StreamWriter interface {
	Write([]int64) (int, error)
}

// Int64StreamWriteCloser defines writer and closer for int64 type.
type Int64StreamWriteCloser interface {
	Closer
	Int64StreamWriter
}

// Int64ReadWriteCloser composes reader types with closer for int64.
// with associated close method.
type Int64ReadWriteCloser interface {
	Closer
	Int64Reader
	Int64Writer
}

// Int64StreamReadWriteCloser composes stream types with closer for int64.
type Int64Stream interface {
	Closer
	Int64StreamReader
	Int64StreamWriter
}
