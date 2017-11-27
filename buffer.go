package mnet

import (
	"encoding/binary"
	"io"
	"sync"
	"time"
)

// BufferedIntervalWriter implements a interval buffering writer which
// given the provided size will fill it's buffer until either
// content reach given size or given duration is met for flushing.
// Each write to a BufferedIntervalWriter resets it timer, ensuring
// as much writes is collected before hitting final writer to
// minimal write calls for buffer.
type BufferedIntervalWriter struct {
	timer *time.Timer
	w     io.Writer
	dur   time.Duration
	mu    sync.RWMutex
	buff  []byte
	c     int
	err   error
	size  int
}

// NewBufferedIntervalWriter returns new BufferedIntervalWriter whoes has at least the specified size.
func NewBufferedIntervalWriter(w io.Writer, bufferSize int, writeInterval time.Duration) *BufferedIntervalWriter {
	bu := &BufferedIntervalWriter{
		w:    w,
		dur:  writeInterval,
		size: bufferSize,
		buff: make([]byte, 0, bufferSize),
	}

	bu.timer = time.AfterFunc(writeInterval, func() { bu.Flush() })

	return bu
}

// Flush sends given data into underline writer.
func (bu *BufferedIntervalWriter) Flush() error {
	bu.mu.Lock()
	defer bu.mu.Unlock()

	if bu.err != nil {
		return bu.err
	}

	if bu.c == 0 {
		return nil
	}

	bu.timer.Reset(bu.dur)

	n, err := bu.w.Write(bu.buff[0:bu.c])
	if err == nil && n < bu.c {
		err = io.ErrShortWrite
	}

	if err != nil {
		if n != 0 && n < bu.c {
			copy(bu.buff[:bu.c-n], bu.buff[n:bu.c])
			bu.c -= n
		}

		if err != io.ErrShortWrite {
			bu.err = err
		}

		bu.timer.Stop()
		return err
	}

	bu.c = 0
	bu.buff = bu.buff[:0]
	return nil
}

// Close closes the BufferedIntervalWriter, stops the timer, clears
// all references and sets an err ensuring the instance can not be
// re-used.
func (bu *BufferedIntervalWriter) Close() error {
	bu.mu.Lock()

	if bu.err != nil {
		bu.mu.Unlock()
		return bu.err
	}

	bu.timer.Stop()
	bu.timer = nil
	bu.mu.Unlock()

	bu.Flush()

	bu.mu.Lock()
	defer bu.mu.Unlock()
	if bu.err != nil {
		return bu.err
	}

	bu.buff = nil
	bu.c = 0

	bu.err = io.ErrClosedPipe

	return nil
}

// Length returns the current total length of items in buffer.
func (bu *BufferedIntervalWriter) Length() int {
	bu.mu.RLock()
	defer bu.mu.RUnlock()
	return bu.c
}

// Write writes given data into internal buffer ensuring writes
// occur safely until buffer is full.
// It is a critical part area, so make mutex call Lock/Unlock
// without defer.
func (bu *BufferedIntervalWriter) Write(d []byte) (int, error) {
	bu.mu.Lock()

	if bu.err != nil {
		bu.mu.Unlock()
		return 0, bu.err
	}

	if len(d) == 0 {
		bu.mu.Unlock()
		return 0, nil
	}

	// Get the next size of data if new data was added,
	// if it exceed maximum, then flush it
	dLen := len(d)
	nextSize := bu.c + dLen

	if nextSize >= bu.size {
		n, err := bu.w.Write(d)
		if n < dLen && err == nil {
			err = io.ErrShortWrite
		}

		if err != nil {
			if err != io.ErrShortWrite {
				bu.err = err
			}
		}

		bu.mu.Unlock()
		return n, err
	}

	bu.timer.Reset(bu.dur)

	copied := copy(bu.buff[bu.c:bu.size], d)
	bu.c += copied
	bu.buff = bu.buff[0:bu.c]
	bu.mu.Unlock()

	return copied, nil
}

// SizeAppendBufferredWriter implements a custom buffer ontop of a given io.Writer, where
// it's length is added to the underline writer onces all data has being collected.
type SizeAppendBufferredWriter struct {
	w    io.Writer
	data []byte
	err  error
	c    int
	m    int
}

// NewSizeAppenBuffereddWriter returns a new SizeAppendBufferredWriter whoes provided size is
// given a minimum size for it's buffer, where written data will
// at the point of flushing have a uin16 2-length header which stores
// the length of the message before written to underline writer.
// Ensure to always provide a big enough minimum buffer size to host data
// to be written, this ensures the array is big enough and wont incur
// multiple expansion when data is being written into buffer.
func NewSizeAppenBuffereddWriter(w io.Writer, minBufferSize int) *SizeAppendBufferredWriter {
	return &SizeAppendBufferredWriter{
		w:    w,
		m:    minBufferSize,
		data: make([]byte, 0, minBufferSize),
	}
}

// Length returns the current total length of items in buffer.
func (wb *SizeAppendBufferredWriter) Length() int {
	return wb.c
}

// Flush attempts to write underline data into underline writer.
func (wb *SizeAppendBufferredWriter) Flush() error {
	if wb.err != nil {
		return wb.err
	}

	if wb.c == 0 {
		return nil
	}

	data := wb.data[0:wb.c]
	header := make([]byte, 2, wb.c+50)
	binary.BigEndian.PutUint16(header, uint16(wb.c))
	copy(header[2:cap(header)], data)
	header = header[0 : wb.c+2]

	count := len(header)
	n, err := wb.w.Write(header)
	if n < count && err == nil {
		err = io.ErrShortWrite
	}

	if err != nil {
		if n != 0 && n < count {
			copy(wb.data[0:wb.c-n], wb.data[n:wb.c])
			wb.c -= n
		}

		if err != io.ErrShortWrite {
			wb.err = err
		}

		return err
	}

	wb.c = 0
	wb.data = wb.data[:0]

	return nil
}

// Write delivers provided data into internal buffer.
func (wb *SizeAppendBufferredWriter) Write(d []byte) (int, error) {
	if wb.err != nil {
		return 0, wb.err
	}

	if len(d) == 0 {
		return 0, nil
	}

	wb.data = append(wb.data, d...)
	wb.c += len(d)

	return len(d), nil
}
