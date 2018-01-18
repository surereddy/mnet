package melon

// StringUniqueHash defines a unique hash for String which can
// be used to reference a given instance within a context.ValueBag or a google context.Context
// value store.
const StringUniqueHash = "62a027764a58e833c7beb34a33ccb1584d611d17"

// StringReader defines reader for string type.
type StringReader interface {
	ReadString() (string, error)
}

// StringReadCloser defines reader and closer for string type.
type StringReadCloser interface {
	Closer
	StringReader
}

// StringStreamReader defines reader string type.
type StringStreamReader interface {
	Read(int) ([]string, error)
}

// StringStreamReadCloser defines reader and closer for string type.
type StringStreamReadCloser interface {
	Closer
	StringStreamReader
}

// StringWriter defines writer for string type.
type StringWriter interface {
	WriteString(string) error
}

// StringWriteCloser defines writer and closer for string type.
type StringWriteCloser interface {
	Closer
	StringWriter
}

// StringStreamWrite defines writer for string type.
type StringStreamWriter interface {
	Write([]string) (int, error)
}

// StringStreamWriteCloser defines writer and closer for string type.
type StringStreamWriteCloser interface {
	Closer
	StringStreamWriter
}

// StringReadWriteCloser composes reader types with closer for string.
// with associated close method.
type StringReadWriteCloser interface {
	Closer
	StringReader
	StringWriter
}

// StringStreamReadWriteCloser composes stream types with closer for string.
type StringStream interface {
	Closer
	StringStreamReader
	StringStreamWriter
}
