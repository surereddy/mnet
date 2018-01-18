package melon

// Complex128UniqueHash defines a unique hash for Complex128 which can
// be used to reference a given instance within a context.ValueBag or a google context.Context
// value store.
const Complex128UniqueHash = "1e22bb114616c1ea2c888d34d5d519a35cc9103c"

// Complex128Reader defines reader for complex128 type.
type Complex128Reader interface {
	ReadComplex128() (complex128, error)
}

// Complex128ReadCloser defines reader and closer for complex128 type.
type Complex128ReadCloser interface {
	Closer
	Complex128Reader
}

// Complex128StreamReader defines reader complex128 type.
type Complex128StreamReader interface {
	Read(int) ([]complex128, error)
}

// Complex128StreamReadCloser defines reader and closer for complex128 type.
type Complex128StreamReadCloser interface {
	Closer
	Complex128StreamReader
}

// Complex128Writer defines writer for complex128 type.
type Complex128Writer interface {
	WriteComplex128(complex128) error
}

// Complex128WriteCloser defines writer and closer for complex128 type.
type Complex128WriteCloser interface {
	Closer
	Complex128Writer
}

// Complex128StreamWrite defines writer for complex128 type.
type Complex128StreamWriter interface {
	Write([]complex128) (int, error)
}

// Complex128StreamWriteCloser defines writer and closer for complex128 type.
type Complex128StreamWriteCloser interface {
	Closer
	Complex128StreamWriter
}

// Complex128ReadWriteCloser composes reader types with closer for complex128.
// with associated close method.
type Complex128ReadWriteCloser interface {
	Closer
	Complex128Reader
	Complex128Writer
}

// Complex128StreamReadWriteCloser composes stream types with closer for complex128.
type Complex128Stream interface {
	Closer
	Complex128StreamReader
	Complex128StreamWriter
}
