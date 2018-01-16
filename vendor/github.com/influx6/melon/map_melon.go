package melon

// MapUniqueHash defines a unique hash for Map which can
// be used to reference a given instance within a context.ValueBag or a google context.Context
// value store.
const MapUniqueHash = "ba70ac0926874b71d829907f558c032da235489c"

// MapReader defines reader for map[string]interface{} type.
type MapReader interface {
	ReadMap() (map[string]interface{}, error)
}

// MapReadCloser defines reader and closer for map[string]interface{} type.
type MapReadCloser interface {
	Closer
	MapReader
}

// MapStreamReader defines reader map[string]interface{} type.
type MapStreamReader interface {
	Read(int) ([]map[string]interface{}, error)
}

// MapStreamReadCloser defines reader and closer for map[string]interface{} type.
type MapStreamReadCloser interface {
	Closer
	MapStreamReader
}

// MapWriter defines writer for map[string]interface{} type.
type MapWriter interface {
	WriteMap(map[string]interface{}) error
}

// MapWriteCloser defines writer and closer for map[string]interface{} type.
type MapWriteCloser interface {
	Closer
	MapWriter
}

// MapStreamWrite defines writer for map[string]interface{} type.
type MapStreamWriter interface {
	Write([]map[string]interface{}) (int, error)
}

// MapStreamWriteCloser defines writer and closer for map[string]interface{} type.
type MapStreamWriteCloser interface {
	Closer
	MapStreamWriter
}

// MapReadWriteCloser composes reader types with closer for map[string]interface{}.
// with associated close method.
type MapReadWriteCloser interface {
	Closer
	MapReader
	MapWriter
}

// MapStreamReadWriteCloser composes stream types with closer for map[string]interface{}.
type MapStream interface {
	Closer
	MapStreamReader
	MapStreamWriter
}
