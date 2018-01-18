package melon

// BoolUniqueHash defines a unique hash for Bool which can
// be used to reference a given instance within a context.ValueBag or a google context.Context
// value store.
const BoolUniqueHash = "189fa523325f8a701fd00ad1b0cd386b4b629299"

// BoolReader defines reader for bool type.
type BoolReader interface {
	ReadBool() (bool, error)
}

// BoolReadCloser defines reader and closer for bool type.
type BoolReadCloser interface {
	Closer
	BoolReader
}

// BoolStreamReader defines reader bool type.
type BoolStreamReader interface {
	Read(int) ([]bool, error)
}

// BoolStreamReadCloser defines reader and closer for bool type.
type BoolStreamReadCloser interface {
	Closer
	BoolStreamReader
}

// BoolWriter defines writer for bool type.
type BoolWriter interface {
	WriteBool(bool) error
}

// BoolWriteCloser defines writer and closer for bool type.
type BoolWriteCloser interface {
	Closer
	BoolWriter
}

// BoolStreamWrite defines writer for bool type.
type BoolStreamWriter interface {
	Write([]bool) (int, error)
}

// BoolStreamWriteCloser defines writer and closer for bool type.
type BoolStreamWriteCloser interface {
	Closer
	BoolStreamWriter
}

// BoolReadWriteCloser composes reader types with closer for bool.
// with associated close method.
type BoolReadWriteCloser interface {
	Closer
	BoolReader
	BoolWriter
}

// BoolStreamReadWriteCloser composes stream types with closer for bool.
type BoolStream interface {
	Closer
	BoolStreamReader
	BoolStreamWriter
}
