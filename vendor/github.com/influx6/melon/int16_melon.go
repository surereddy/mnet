package melon

// Int16UniqueHash defines a unique hash for Int16 which can
// be used to reference a given instance within a context.ValueBag or a google context.Context
// value store.
const Int16UniqueHash = "3b1227b09fd0719b0d9a553739bfa14479fc0a09"

// Int16Reader defines reader for int16 type.
type Int16Reader interface {
	ReadInt16() (int16, error)
}

// Int16ReadCloser defines reader and closer for int16 type.
type Int16ReadCloser interface {
	Closer
	Int16Reader
}

// Int16StreamReader defines reader int16 type.
type Int16StreamReader interface {
	Read(int) ([]int16, error)
}

// Int16StreamReadCloser defines reader and closer for int16 type.
type Int16StreamReadCloser interface {
	Closer
	Int16StreamReader
}

// Int16Writer defines writer for int16 type.
type Int16Writer interface {
	WriteInt16(int16) error
}

// Int16WriteCloser defines writer and closer for int16 type.
type Int16WriteCloser interface {
	Closer
	Int16Writer
}

// Int16StreamWrite defines writer for int16 type.
type Int16StreamWriter interface {
	Write([]int16) (int, error)
}

// Int16StreamWriteCloser defines writer and closer for int16 type.
type Int16StreamWriteCloser interface {
	Closer
	Int16StreamWriter
}

// Int16ReadWriteCloser composes reader types with closer for int16.
// with associated close method.
type Int16ReadWriteCloser interface {
	Closer
	Int16Reader
	Int16Writer
}

// Int16StreamReadWriteCloser composes stream types with closer for int16.
type Int16Stream interface {
	Closer
	Int16StreamReader
	Int16StreamWriter
}
