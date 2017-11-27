package mnet_test

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"

	"github.com/influx6/faux/tests"
	"github.com/influx6/mnet"
)

func TestBufferedIntervalWriterWithImmediateFlush(t *testing.T) {
	var writer bytes.Buffer
	bu := mnet.NewBufferedIntervalWriter(&writer, 512, 1*time.Second)

	content := []byte("Thunder world, Reckage before the dawn")
	contentLen := len(content)

	if _, err := bu.Write(content); err != nil {
		tests.FailedWithError(err, "Should have written message to writer.")
	}
	tests.Passed("Should have written message to writer.")

	if writer.Len() != 0 {
		tests.Failed("Should have no content with underline writer for buffered data")
	}
	tests.Passed("Should have no content with underline writer for buffered data")

	if err := bu.Flush(); err != nil {
		tests.FailedWithError(err, "Should have successfully flushed data into buffer")
	}
	tests.Passed("Should have successfully flushed data into buffer")

	if writer.Len() == 0 {
		tests.Failed("Should have content within underline writer for buffered data")
	}
	tests.Passed("Should have content within underline writer for buffered data")

	if writtenContent := writer.Next(contentLen); !bytes.Equal(content, writtenContent) {
		tests.Info("Expected: %+q", content)
		tests.Info("Received: %+q", writtenContent)
		tests.Failed("Should have successfully matched written message with original")
	}
	tests.Passed("Should have successfully matched written message with original")

	if err := bu.Close(); err != nil {
		tests.FailedWithError(err, "Should have closed writer.")
	}
	tests.Passed("Should have closed writer.")

	if _, err := bu.Write(content); err == nil {
		tests.Failed("Should have failed to write message to writer.")
	}
	tests.Passed("Should have failed to write message to writer.")
}

func TestBufferedIntervalWriterWithTimedFlush(t *testing.T) {
	var writer bytes.Buffer
	bu := mnet.NewBufferedIntervalWriter(&writer, 512, 50*time.Millisecond)

	content := []byte("Thunder world, Reckage before the dawn")
	contentLen := len(content)

	if _, err := bu.Write(content); err != nil {
		tests.FailedWithError(err, "Should have written message to writer.")
	}
	tests.Passed("Should have written message to writer.")

	if writer.Len() != 0 {
		tests.Failed("Should have no content with underline writer for buffered data")
	}
	tests.Passed("Should have no content with underline writer for buffered data")

	<-time.After(60 * time.Millisecond)

	if writer.Len() == 0 {
		tests.Failed("Should have content within underline writer for buffered data")
	}
	tests.Passed("Should have content within underline writer for buffered data")

	if writtenContent := writer.Next(contentLen); !bytes.Equal(content, writtenContent) {
		tests.Info("Expected: %+q", content)
		tests.Info("Received: %+q", writtenContent)
		tests.Failed("Should have successfully matched written message with original")
	}
	tests.Passed("Should have successfully matched written message with original")

	if err := bu.Close(); err != nil {
		tests.FailedWithError(err, "Should have closed writer.")
	}
	tests.Passed("Should have closed writer.")

	if _, err := bu.Write(content); err == nil {
		tests.Failed("Should have failed to write message to writer.")
	}
	tests.Passed("Should have failed to write message to writer.")
}

func TestSizeAppendBufferedWriter(t *testing.T) {
	var writer bytes.Buffer
	bu := mnet.NewSizeAppenBuffereddWriter(&writer, 512)

	content := []byte("Thunder world, Reckage before the dawn")
	contentLen := len(content)

	if _, err := bu.Write(content); err != nil {
		tests.FailedWithError(err, "Should have written message to writer.")
	}
	tests.Passed("Should have written message to writer.")

	if writer.Len() != 0 {
		tests.Failed("Should have no content with underline writer for buffered data")
	}
	tests.Passed("Should have no content with underline writer for buffered data")

	if err := bu.Flush(); err != nil {
		tests.FailedWithError(err, "Should have successfully flushed data into buffer")
	}
	tests.Passed("Should have successfully flushed data into buffer")

	headerLen := int(binary.BigEndian.Uint16(writer.Next(2)))
	if headerLen != contentLen {
		tests.Info("Content Length: %d", contentLen)
		tests.Info("Header Length: %d", headerLen)
		tests.Failed("Should have successfully matched length header of content from buffer to content length")
	}
	tests.Passed("Should have successfully matched length header of content from buffer to content length")

	if writtenContent := writer.Next(contentLen); !bytes.Equal(content, writtenContent) {
		tests.Info("Expected: %+q", content)
		tests.Info("Received: %+q", writtenContent)
		tests.Failed("Should have successfully matched written message with original")
	}
	tests.Passed("Should have successfully matched written message with original")
}

type writtenBuffer struct {
	c            int
	totalWritten int
}

func (b *writtenBuffer) Write(d []byte) (int, error) {
	b.c += 1
	b.totalWritten += len(d)
	return len(d), nil
}

func BenchmarkBufferedIntervalBuffer(b *testing.B) {
	var writer writtenBuffer
	bu := mnet.NewBufferedIntervalWriter(&writer, 5024, 1*time.Second)

	content := []byte("Thunder world, Reckage before the dawn")

	var at int
	for i := 0; i < b.N; i++ {
		at++
		bu.Write(content)
	}

	b.Logf("Written to buffer a total of %d times after %d runs with %d bytes\n", writer.c, at, writer.totalWritten)
}
