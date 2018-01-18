package melon

// ErrorUniqueHash defines a unique hash for Error which can
// be used to reference a given instance within a context.ValueBag or a google context.Context
// value store.
const ErrorUniqueHash = "d3e8aa638ed54763fe05bd5404e80abdfc12b4f5"

// ErrorReader defines reader for error type.
type ErrorReader interface {
	ReadError() (error, error)
}

// ErrorReadCloser defines reader and closer for error type.
type ErrorReadCloser interface {
	Closer
	ErrorReader
}

// ErrorStreamReader defines reader error type.
type ErrorStreamReader interface {
	Read(int) ([]error, error)
}

// ErrorStreamReadCloser defines reader and closer for error type.
type ErrorStreamReadCloser interface {
	Closer
	ErrorStreamReader
}

// ErrorWriter defines writer for error type.
type ErrorWriter interface {
	WriteError(error) error
}

// ErrorWriteCloser defines writer and closer for error type.
type ErrorWriteCloser interface {
	Closer
	ErrorWriter
}

// ErrorStreamWrite defines writer for error type.
type ErrorStreamWriter interface {
	Write([]error) (int, error)
}

// ErrorStreamWriteCloser defines writer and closer for error type.
type ErrorStreamWriteCloser interface {
	Closer
	ErrorStreamWriter
}

// ErrorReadWriteCloser composes reader types with closer for error.
// with associated close method.
type ErrorReadWriteCloser interface {
	Closer
	ErrorReader
	ErrorWriter
}

// ErrorStreamReadWriteCloser composes stream types with closer for error.
type ErrorStream interface {
	Closer
	ErrorStreamReader
	ErrorStreamWriter
}
