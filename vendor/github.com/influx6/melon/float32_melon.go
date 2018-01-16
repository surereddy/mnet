package melon

// Float32UniqueHash defines a unique hash for Float32 which can
// be used to reference a given instance within a context.ValueBag or a google context.Context
// value store.
const Float32UniqueHash = "c7cd93eba0df834db18da1b14dc29a7551a1dd80"

// Float32Reader defines reader for float32 type.
type Float32Reader interface {
	ReadFloat32() (float32, error)
}

// Float32ReadCloser defines reader and closer for float32 type.
type Float32ReadCloser interface {
	Closer
	Float32Reader
}

// Float32StreamReader defines reader float32 type.
type Float32StreamReader interface {
	Read(int) ([]float32, error)
}

// Float32StreamReadCloser defines reader and closer for float32 type.
type Float32StreamReadCloser interface {
	Closer
	Float32StreamReader
}

// Float32Writer defines writer for float32 type.
type Float32Writer interface {
	WriteFloat32(float32) error
}

// Float32WriteCloser defines writer and closer for float32 type.
type Float32WriteCloser interface {
	Closer
	Float32Writer
}

// Float32StreamWrite defines writer for float32 type.
type Float32StreamWriter interface {
	Write([]float32) (int, error)
}

// Float32StreamWriteCloser defines writer and closer for float32 type.
type Float32StreamWriteCloser interface {
	Closer
	Float32StreamWriter
}

// Float32ReadWriteCloser composes reader types with closer for float32.
// with associated close method.
type Float32ReadWriteCloser interface {
	Closer
	Float32Reader
	Float32Writer
}

// Float32StreamReadWriteCloser composes stream types with closer for float32.
type Float32Stream interface {
	Closer
	Float32StreamReader
	Float32StreamWriter
}
