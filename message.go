package mnet

import (
	"bytes"
	"encoding/binary"
	"errors"
	"sync"
	"sync/atomic"
)

// errors ...
var (
	ErrNoDataYet = errors.New("data is not yet available for reading")
	ErrParseErr  = errors.New("Failed to parse data")
)

// Message defines a single node upon a linked list which
// contains it's data and a link to the next message node.
type Message struct {
	Data []byte
	Next *Message
}

// SizedMessageParser implements a parser which parses incoming messages
// as a series of [MessageSizeLength][Message] joined together, which it
// splits apart into individual message blocks. The parser keeps a series of internal
// linked list which contains already processed message, which has a endless length
// value which allows appending new messages as they arrive.
type SizedMessageParser struct {
	scratch           bytes.Buffer
	processedMessages int64
	mu                sync.RWMutex
	head              *Message
	tail              *Message
}

// TotalProcessed returns total message processed successfully.
func (smp *SizedMessageParser) TotalProcessed() int {
	return int(atomic.LoadInt64(&smp.processedMessages))
}

// Parse implements the necessary procedure for parsing incoming data and
// appropriately splitting them accordingly to their respective parts.
func (smp *SizedMessageParser) Parse(d []byte) error {
	smp.scratch.Write(d)

	for smp.scratch.Len() > 0 {
		nextdata := smp.scratch.Next(2)
		if len(nextdata) < 2 {
			smp.scratch.Write(nextdata)
			break
		}

		nextSize := int(binary.BigEndian.Uint16(nextdata))

		// If scratch is zero and we do have count data, maybe we face a unfinished write.
		if smp.scratch.Len() == 0 {
			smp.scratch.Write(nextdata)
			break
		}

		if nextSize > smp.scratch.Len() {
			rest := smp.scratch.Bytes()
			restruct := append(nextdata, rest...)
			smp.scratch.Reset()
			smp.scratch.Write(restruct)
			break
		}

		next := smp.scratch.Next(nextSize)
		smp.addMessage(&Message{Data: next})
	}

	return nil
}

// Next returns the next message saved on the parsers linked list.
func (smp *SizedMessageParser) Next() ([]byte, error) {
	smp.mu.RLock()
	defer smp.mu.RUnlock()

	if smp.tail == nil && smp.head == nil {
		return nil, ErrNoDataYet
	}

	head := smp.head
	if smp.tail == head {
		smp.tail = nil
		smp.head = nil
	} else {
		next := head.Next
		head.Next = nil
		smp.head = next
	}

	return head.Data, nil
}

func (smp *SizedMessageParser) addMessage(m *Message) {
	smp.mu.Lock()
	defer smp.mu.Unlock()

	atomic.AddInt64(&smp.processedMessages, 1)
	if smp.head == nil && smp.tail == nil {
		smp.head = m
		smp.tail = m
		return
	}

	smp.tail.Next = m
	smp.tail = m
}
