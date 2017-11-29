package mnet

import (
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

var messagePool = sync.Pool{
	New: func() interface{} {
		return new(Message)
	},
}

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
	scratch           *BufferedPeeker
	processedMessages int64
	mu                sync.RWMutex
	head              *Message
	tail              *Message
}

// NewSizedMessageParser returns a new instance of SizedMessageParser.
func NewSizedMessageParser() *SizedMessageParser {
	return &SizedMessageParser{
		scratch: NewBufferedPeeker(nil),
	}
}

// TotalProcessed returns total message processed successfully.
func (smp *SizedMessageParser) TotalProcessed() int {
	return int(atomic.LoadInt64(&smp.processedMessages))
}

// Parse implements the necessary procedure for parsing incoming data and
// appropriately splitting them accordingly to their respective parts.
func (smp *SizedMessageParser) Parse(d []byte) error {
	smp.scratch.Reset(d)

	for smp.scratch.Area() > 0 {
		nextdata := smp.scratch.Next(2)
		if len(nextdata) < 2 {
			smp.scratch.Reverse(2)
			break
		}

		nextSize := int(binary.BigEndian.Uint16(nextdata))

		// If scratch is zero and we do have count data, maybe we face a unfinished write.
		if smp.scratch.Area() == 0 {
			smp.scratch.Reverse(2)
			break
		}

		if nextSize > smp.scratch.Area() {
			smp.scratch.Reverse(2)
			break
		}

		next := smp.scratch.Next(nextSize)
		msg := messagePool.New().(*Message)
		msg.Data = next
		smp.addMessage(msg)
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

	data := head.Data
	head.Data = nil
	messagePool.Put(head)

	return data, nil
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
