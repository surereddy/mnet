package melon

// Complex64UniqueHash defines a unique hash for Complex64 which can
// be used to reference a given instance within a context.ValueBag or a google context.Context
// value store.
const Complex64UniqueHash = "161e6337d01e5b8bd4e79ac4060fde11263b2d17"

// Complex64Reader defines reader for complex64 type.
type Complex64Reader interface {
	ReadComplex64() (complex64, error)
}

// Complex64ReadCloser defines reader and closer for complex64 type.
type Complex64ReadCloser interface {
	Closer
	Complex64Reader
}

// Complex64StreamReader defines reader complex64 type.
type Complex64StreamReader interface {
	Read(int) ([]complex64, error)
}

// Complex64StreamReadCloser defines reader and closer for complex64 type.
type Complex64StreamReadCloser interface {
	Closer
	Complex64StreamReader
}

// Complex64Writer defines writer for complex64 type.
type Complex64Writer interface {
	WriteComplex64(complex64) error
}

// Complex64WriteCloser defines writer and closer for complex64 type.
type Complex64WriteCloser interface {
	Closer
	Complex64Writer
}

// Complex64StreamWrite defines writer for complex64 type.
type Complex64StreamWriter interface {
	Write([]complex64) (int, error)
}

// Complex64StreamWriteCloser defines writer and closer for complex64 type.
type Complex64StreamWriteCloser interface {
	Closer
	Complex64StreamWriter
}

// Complex64ReadWriteCloser composes reader types with closer for complex64.
// with associated close method.
type Complex64ReadWriteCloser interface {
	Closer
	Complex64Reader
	Complex64Writer
}

// Complex64StreamReadWriteCloser composes stream types with closer for complex64.
type Complex64Stream interface {
	Closer
	Complex64StreamReader
	Complex64StreamWriter
}
