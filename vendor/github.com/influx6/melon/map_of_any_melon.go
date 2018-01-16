package melon

// MapOfAnyUniqueHash defines a unique hash for MapOfAny which can
// be used to reference a given instance within a context.ValueBag or a google context.Context
// value store.
const MapOfAnyUniqueHash = "c2d03452b8860af6906e2c8a6e847f371db07988"

// MapOfAnyReader defines reader for map[interface{}]interface{} type.
type MapOfAnyReader interface {
	ReadMapOfAny() (map[interface{}]interface{}, error)
}

// MapOfAnyReadCloser defines reader and closer for map[interface{}]interface{} type.
type MapOfAnyReadCloser interface {
	Closer
	MapOfAnyReader
}

// MapOfAnyStreamReader defines reader map[interface{}]interface{} type.
type MapOfAnyStreamReader interface {
	Read(int) ([]map[interface{}]interface{}, error)
}

// MapOfAnyStreamReadCloser defines reader and closer for map[interface{}]interface{} type.
type MapOfAnyStreamReadCloser interface {
	Closer
	MapOfAnyStreamReader
}

// MapOfAnyWriter defines writer for map[interface{}]interface{} type.
type MapOfAnyWriter interface {
	WriteMapOfAny(map[interface{}]interface{}) error
}

// MapOfAnyWriteCloser defines writer and closer for map[interface{}]interface{} type.
type MapOfAnyWriteCloser interface {
	Closer
	MapOfAnyWriter
}

// MapOfAnyStreamWrite defines writer for map[interface{}]interface{} type.
type MapOfAnyStreamWriter interface {
	Write([]map[interface{}]interface{}) (int, error)
}

// MapOfAnyStreamWriteCloser defines writer and closer for map[interface{}]interface{} type.
type MapOfAnyStreamWriteCloser interface {
	Closer
	MapOfAnyStreamWriter
}

// MapOfAnyReadWriteCloser composes reader types with closer for map[interface{}]interface{}.
// with associated close method.
type MapOfAnyReadWriteCloser interface {
	Closer
	MapOfAnyReader
	MapOfAnyWriter
}

// MapOfAnyStreamReadWriteCloser composes stream types with closer for map[interface{}]interface{}.
type MapOfAnyStream interface {
	Closer
	MapOfAnyStreamReader
	MapOfAnyStreamWriter
}
