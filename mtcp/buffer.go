package mtcp

import (
	"encoding/binary"
	"io"
)

// WriteBuffer implements a custom buffer ontop of a given io.Writer, where
// it's length is added to the underline writer onces all data has being collected.
type WriteBuffer struct {
	w    io.Writer
	data []byte
	c    int
	min  int
}

// NewWriteBuffer returns new instance of writebuffer.
func NewWriteBuffer(w io.Writer, size int) *WriteBuffer {
	return &WriteBuffer{
		w:    w,
		min:  size,
		data: make([]byte, 0, size),
	}
}

// Flush attempts to write underline data into underline writer.
func (wb *WriteBuffer) Flush() error {
	if wb.c == 0 {
		return nil
	}

	data := wb.data[:wb.c]
	count := wb.c

	header := make([]byte, 2)
	binary.BigEndian.PutUint16(header, uint16(wb.c))
	header = append(header, data...)

	wb.c = 0
	wb.data = wb.data[:0]

	n, err := wb.w.Write(header)
	if err != nil {
		return err
	}

	if n < count {
		return io.ErrShortWrite
	}

	return nil
}

// Write delivers provided data into internal buffer.
func (wb *WriteBuffer) Write(d []byte) (int, error) {
	if len(d) == 0 {
		return 0, nil
	}

	wb.data = append(wb.data, d...)
	wb.c += len(d)
	return len(d), nil
}
