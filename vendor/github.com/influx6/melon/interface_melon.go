package melon

// InterfaceUniqueHash defines a unique hash for Interface which can
// be used to reference a given instance within a context.ValueBag or a google context.Context
// value store.
const InterfaceUniqueHash = "4b3ea5d9542287d43b2938e455ad877a8cc1b573"

// InterfaceReader defines reader for interface{} type.
type InterfaceReader interface {
	ReadInterface() (interface{}, error)
}

// InterfaceReadCloser defines reader and closer for interface{} type.
type InterfaceReadCloser interface {
	Closer
	InterfaceReader
}

// InterfaceStreamReader defines reader interface{} type.
type InterfaceStreamReader interface {
	Read(int) ([]interface{}, error)
}

// InterfaceStreamReadCloser defines reader and closer for interface{} type.
type InterfaceStreamReadCloser interface {
	Closer
	InterfaceStreamReader
}

// InterfaceWriter defines writer for interface{} type.
type InterfaceWriter interface {
	WriteInterface(interface{}) error
}

// InterfaceWriteCloser defines writer and closer for interface{} type.
type InterfaceWriteCloser interface {
	Closer
	InterfaceWriter
}

// InterfaceStreamWrite defines writer for interface{} type.
type InterfaceStreamWriter interface {
	Write([]interface{}) (int, error)
}

// InterfaceStreamWriteCloser defines writer and closer for interface{} type.
type InterfaceStreamWriteCloser interface {
	Closer
	InterfaceStreamWriter
}

// InterfaceReadWriteCloser composes reader types with closer for interface{}.
// with associated close method.
type InterfaceReadWriteCloser interface {
	Closer
	InterfaceReader
	InterfaceWriter
}

// InterfaceStreamReadWriteCloser composes stream types with closer for interface{}.
type InterfaceStream interface {
	Closer
	InterfaceStreamReader
	InterfaceStreamWriter
}
