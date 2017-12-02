package mnet

import (
	"encoding/binary"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

// ErrItemsInBuffer sets error to be returned if attempts are made to expand buffer
// while items exist in it.
var ErrItemsInBuffer = errors.New("buffer has pending content")

// BufferedIntervalWriter implements a interval buffering writer which
// given the provided size will fill it's buffer until either
// content reach given size or given duration is met for flushing.
// Each write to a BufferedIntervalWriter resets it timer, ensuring
// as much writes is collected before hitting final writer to
// minimal write calls for buffer.
type BufferedIntervalWriter struct {
	timer                   *time.Timer
	w                       io.Writer
	dur                     time.Duration
	mu                      sync.RWMutex
	timerStopped            bool
	buff                    []byte
	c                       int
	err                     error
	size                    int
	totalReceived           int64
	totalFlushed            int64
	totalWrittenDirectlyToW int64
	lastWrittenDirectlyToW  int64
	lastFlushed             int64
	lastWritten             int64
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

// Reset resets the internal writer to a new writer to be used.
func (bu *BufferedIntervalWriter) Reset(w io.Writer) error {
	bu.mu.Lock()
	bu.w = w
	bu.err = nil

	// If writer is nil and we have not stopped timer, then stop it
	if w == nil && !bu.timerStopped {
		bu.timerStopped = true
		bu.timer.Stop()
	}

	// If writer aint nil and we just got stopped, then start it up
	// again.
	if w != nil && bu.timerStopped {
		bu.timerStopped = false
		bu.timer.Reset(bu.dur)
	}

	bu.mu.Unlock()
	return nil
}

// StopTimer stops the internal flush timer of the writer.
func (bu *BufferedIntervalWriter) StopTimer() error {
	bu.timer.Stop()
	bu.mu.Lock()
	bu.timerStopped = true
	bu.mu.Unlock()
	return nil
}

// Flushed returns total flushed in bytes into buffer since start.
func (bu *BufferedIntervalWriter) Flushed() int {
	return int(atomic.LoadInt64(&bu.totalFlushed))
}

// LastWrittenDirectly returns total written in bytes straight to writer instead of buffer since start.
func (bu *BufferedIntervalWriter) LastWrittenDirectly() int {
	return int(atomic.LoadInt64(&bu.lastWrittenDirectlyToW))
}

// WrittenDirectly returns total written in bytes straight to writer instead of buffer since start.
func (bu *BufferedIntervalWriter) WrittenDirectly() int {
	return int(atomic.LoadInt64(&bu.totalWrittenDirectlyToW))
}

// Written returns total written in bytes into buffer since start.
func (bu *BufferedIntervalWriter) Written() int {
	return int(atomic.LoadInt64(&bu.totalReceived))
}

// Length returns the current total length of items in buffer.
func (bu *BufferedIntervalWriter) Length() int {
	return int(atomic.LoadInt64(&bu.lastWritten))
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

	if bu.c == 0 && nextSize > bu.size {
		return bu.w.Write(d)
	}

	if nextSize > bu.size {
		bu.mu.Unlock()
		if err := bu.Flush(); err != nil {
			return 0, err
		}
		bu.mu.Lock()
	}

	bu.timer.Reset(bu.dur)

	copied := copy(bu.buff[bu.c:bu.size], d)
	atomic.AddInt64(&bu.totalReceived, int64(copied))
	atomic.AddInt64(&bu.lastWritten, int64(copied))

	bu.c += copied
	bu.buff = bu.buff[0:bu.c]
	bu.mu.Unlock()

	return copied, nil
}

// Flush sends given data into underline writer.
func (bu *BufferedIntervalWriter) Flush() error {
	bu.mu.Lock()

	if bu.err != nil {
		bu.mu.Unlock()
		return bu.err
	}

	if bu.c == 0 {
		bu.mu.Unlock()
		return nil
	}

	if bu.w == nil {
		bu.mu.Unlock()
		return ErrWriteNotAllowed
	}

	bu.timer.Reset(bu.dur)

	n, err := bu.w.Write(bu.buff[0:bu.c])
	atomic.AddInt64(&bu.totalFlushed, int64(n))
	atomic.StoreInt64(&bu.lastFlushed, int64(n))
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

		bu.mu.Unlock()
		bu.timer.Stop()

		return err
	}

	bu.c = 0
	bu.buff = bu.buff[:0]
	atomic.StoreInt64(&bu.lastWritten, 0)

	bu.mu.Unlock()

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
	bu.w = nil
	bu.totalFlushed = 0
	bu.totalReceived = 0
	bu.totalWrittenDirectlyToW = 0
	bu.lastFlushed = 0
	bu.lastWritten = 0

	bu.err = io.ErrClosedPipe

	return nil
}

// SizeAppendBufferredWriter implements a custom buffer ontop of a given io.Writer, where
// it's length is added to the underline writer onces all data has being collected.
type SizeAppendBufferredWriter struct {
	w    io.Writer
	data []byte
	err  error
	c    int
	rc   int
	rcc  int64
	fl   int64
	m    int
}

// NewSizeAppenBuffereddWriter returns a new SizeAppendBufferredWriter whoes provided size is
// given a minimum size for it's buffer, where written data will
// at the point of flushing have a uin16 2-length header which stores
// the length of the message before written to underline writer.
// Ensure to always provide a big enough minimum buffer size to host data
// to be written, this ensures the array is big enough and wont incur
// multiple expansion when data is being written into buffer.
func NewSizeAppenBuffereddWriter(w io.Writer, maxBufferSize int) *SizeAppendBufferredWriter {
	// account for header length size.
	maxBufferSize += 2

	buff := make([]byte, 2, maxBufferSize)
	return &SizeAppendBufferredWriter{
		w:    w,
		data: buff,
		c:    2,
		fl:   0,
		m:    maxBufferSize,
	}
}

// LengthInBuffer returns current length of items in buffer.
func (wb *SizeAppendBufferredWriter) LengthInBuffer() int {
	return int(atomic.LoadInt64(&wb.rcc))
}

// TotalFlushed returns the current total items flushed in buffer.
func (wb *SizeAppendBufferredWriter) TotalFlushed() int {
	return int(atomic.LoadInt64(&wb.fl))
}

// Flush attempts to write underline data into underline writer.
func (wb *SizeAppendBufferredWriter) Flush() error {
	if wb.err != nil {
		return wb.err
	}

	if wb.c == 2 {
		return nil
	}

	data := wb.data[0:wb.c]
	binary.BigEndian.PutUint16(data, uint16(wb.rc))

	count := len(data)
	n, err := wb.w.Write(data)
	if n < count && err == nil {
		err = io.ErrShortWrite
	}

	atomic.AddInt64(&wb.fl, int64(n))

	if err != nil {
		if n != 0 && n < count {
			rest := data[n:]
			copy(wb.data[2:wb.m], rest)
			wb.c = len(rest) + 2
			wb.rc = len(rest)
			atomic.StoreInt64(&wb.rcc, int64(wb.rc))
		}

		if err != io.ErrShortWrite {
			wb.err = err
		}

		return err
	}

	wb.c = 2
	wb.rc = 0
	atomic.StoreInt64(&wb.rcc, 0)
	wb.data = wb.data[:2]

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

	// if we have filled up possible size area, then flush first.
	if len(d)+wb.c > wb.m {
		if err := wb.Flush(); err != nil {
			return 0, err
		}
	}

	copied := copy(wb.data[wb.c:wb.m], d)
	wb.c += copied
	wb.rc = wb.c - 2
	atomic.StoreInt64(&wb.rcc, int64(wb.rc))

	return len(d), nil
}

// BufferedPeeker implements a custom buffer structure which
// allows peeks, reversal of index location of provided byte slice.
// It helps to minimize memory allocation.
type BufferedPeeker struct {
	l int
	d []byte
	c int
}

// NewBufferedPeeker returns new instance of BufferedPeeker.
func NewBufferedPeeker(d []byte) *BufferedPeeker {
	return &BufferedPeeker{
		d: d,
		l: len(d),
	}
}

// Reset sets giving buffers memory slice and sets appropriate
// settings.
func (b *BufferedPeeker) Reset(bm []byte) {
	b.d = bm
	b.l = len(bm)
	b.c = 0
}

// Length returns total length of slice.
func (b *BufferedPeeker) Length() int {
	return len(b.d)
}

// Area returns current available length from current index.
func (b *BufferedPeeker) Area() int {
	return len(b.d[b.c:])
}

// Reverse reverses previous index back a giving length
// It reverse any length back to 0 if length provided exceeds
// underline slice length.
func (b *BufferedPeeker) Reverse(n int) {
	back := b.c - n
	if back <= 0 && b.c == 0 {
		return
	}

	if back <= 0 {
		b.c = 0
		return
	}

	b.c = back
}

// Next returns bytes around giving range.
// If area is beyond slice length, then the rest of slice is returned.
func (b *BufferedPeeker) Next(n int) []byte {
	if b.c+n >= b.l {
		p := b.c
		b.c = b.l
		return b.d[p:]
	}

	area := b.d[b.c : b.c+n]
	b.c += n
	return area
}

// Peek returns the next bytes of giving n length. It
// will not move index beyond current bounds, but returns
// current area if within slice length.
// If area is beyond slice length, then the rest of slice is returned.
func (b *BufferedPeeker) Peek(n int) []byte {
	if b.c+n >= b.l {
		return b.d[b.c:]
	}

	return b.d[b.c : b.c+n]
}
