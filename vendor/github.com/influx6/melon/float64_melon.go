package melon

// Float64UniqueHash defines a unique hash for Float64 which can
// be used to reference a given instance within a context.ValueBag or a google context.Context
// value store.
const Float64UniqueHash = "c11c63ba6f4639c386f2b6ed2ec7dec76ff35a5d"

// Float64Reader defines reader for float64 type.
type Float64Reader interface {
	ReadFloat64() (float64, error)
}

// Float64ReadCloser defines reader and closer for float64 type.
type Float64ReadCloser interface {
	Closer
	Float64Reader
}

// Float64StreamReader defines reader float64 type.
type Float64StreamReader interface {
	Read(int) ([]float64, error)
}

// Float64StreamReadCloser defines reader and closer for float64 type.
type Float64StreamReadCloser interface {
	Closer
	Float64StreamReader
}

// Float64Writer defines writer for float64 type.
type Float64Writer interface {
	WriteFloat64(float64) error
}

// Float64WriteCloser defines writer and closer for float64 type.
type Float64WriteCloser interface {
	Closer
	Float64Writer
}

// Float64StreamWrite defines writer for float64 type.
type Float64StreamWriter interface {
	Write([]float64) (int, error)
}

// Float64StreamWriteCloser defines writer and closer for float64 type.
type Float64StreamWriteCloser interface {
	Closer
	Float64StreamWriter
}

// Float64ReadWriteCloser composes reader types with closer for float64.
// with associated close method.
type Float64ReadWriteCloser interface {
	Closer
	Float64Reader
	Float64Writer
}

// Float64StreamReadWriteCloser composes stream types with closer for float64.
type Float64Stream interface {
	Closer
	Float64StreamReader
	Float64StreamWriter
}
