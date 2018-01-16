package melon

// IntUniqueHash defines a unique hash for Int which can
// be used to reference a given instance within a context.ValueBag or a google context.Context
// value store.
const IntUniqueHash = "e3e6a2908db2b977524568702fabd849a5d5514f"

// IntReader defines reader for int type.
type IntReader interface {
	ReadInt() (int, error)
}

// IntReadCloser defines reader and closer for int type.
type IntReadCloser interface {
	Closer
	IntReader
}

// IntStreamReader defines reader int type.
type IntStreamReader interface {
	Read(int) ([]int, error)
}

// IntStreamReadCloser defines reader and closer for int type.
type IntStreamReadCloser interface {
	Closer
	IntStreamReader
}

// IntWriter defines writer for int type.
type IntWriter interface {
	WriteInt(int) error
}

// IntWriteCloser defines writer and closer for int type.
type IntWriteCloser interface {
	Closer
	IntWriter
}

// IntStreamWrite defines writer for int type.
type IntStreamWriter interface {
	Write([]int) (int, error)
}

// IntStreamWriteCloser defines writer and closer for int type.
type IntStreamWriteCloser interface {
	Closer
	IntStreamWriter
}

// IntReadWriteCloser composes reader types with closer for int.
// with associated close method.
type IntReadWriteCloser interface {
	Closer
	IntReader
	IntWriter
}

// IntStreamReadWriteCloser composes stream types with closer for int.
type IntStream interface {
	Closer
	IntStreamReader
	IntStreamWriter
}
