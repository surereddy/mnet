package melon

// MapOfStringUniqueHash defines a unique hash for MapOfString which can
// be used to reference a given instance within a context.ValueBag or a google context.Context
// value store.
const MapOfStringUniqueHash = "de4cf6eba7a008b25c4561bf3c1885a2c02fc96f"

// MapOfStringReader defines reader for map[string]string type.
type MapOfStringReader interface {
	ReadMapOfString() (map[string]string, error)
}

// MapOfStringReadCloser defines reader and closer for map[string]string type.
type MapOfStringReadCloser interface {
	Closer
	MapOfStringReader
}

// MapOfStringStreamReader defines reader map[string]string type.
type MapOfStringStreamReader interface {
	Read(int) ([]map[string]string, error)
}

// MapOfStringStreamReadCloser defines reader and closer for map[string]string type.
type MapOfStringStreamReadCloser interface {
	Closer
	MapOfStringStreamReader
}

// MapOfStringWriter defines writer for map[string]string type.
type MapOfStringWriter interface {
	WriteMapOfString(map[string]string) error
}

// MapOfStringWriteCloser defines writer and closer for map[string]string type.
type MapOfStringWriteCloser interface {
	Closer
	MapOfStringWriter
}

// MapOfStringStreamWrite defines writer for map[string]string type.
type MapOfStringStreamWriter interface {
	Write([]map[string]string) (int, error)
}

// MapOfStringStreamWriteCloser defines writer and closer for map[string]string type.
type MapOfStringStreamWriteCloser interface {
	Closer
	MapOfStringStreamWriter
}

// MapOfStringReadWriteCloser composes reader types with closer for map[string]string.
// with associated close method.
type MapOfStringReadWriteCloser interface {
	Closer
	MapOfStringReader
	MapOfStringWriter
}

// MapOfStringStreamReadWriteCloser composes stream types with closer for map[string]string.
type MapOfStringStream interface {
	Closer
	MapOfStringStreamReader
	MapOfStringStreamWriter
}
