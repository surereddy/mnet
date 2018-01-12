package mnet

import (
	"errors"
	"net"
	"sync"

	"encoding/binary"

	"github.com/influx6/faux/pools/seeker"
)

const (
	HeaderLength  = 4
	MaxHeaderSize = uint32(4294967295)
)

// errors ...
var (
	ErrInvalidHeader = errors.New("invalid header data: max data size 4294967295")
	ErrHeaderLength  = errors.New("invalid data, expected length of 4 for header size")
)

var (
	messagePool = sync.Pool{New: func() interface{} { return new(messageTomb) }}
)

// messageTomb defines a single node upon a linked list which
// contains it's data and a link to the next messageTomb node.
type messageTomb struct {
	Next   *messageTomb
	Target net.Addr
	Data   []byte
}

// TaggedMessages implements a parser which parses incoming messages
// as a series of [MessageSizeLength][Message] joined together, which it
// splits apart into individual message blocks. The parser keeps a series of internal
// linked list which contains already processed message, which has a endless length
// value which allows appending new messages as they arrive.
type TaggedMessages struct {
	scratch *seeker.BufferedPeeker
	mu      sync.RWMutex
	head    *messageTomb
	tail    *messageTomb
}

// Parse implements the necessary procedure for parsing incoming data and
// appropriately splitting them accordingly to their respective parts.
func (smp *TaggedMessages) Parse(d []byte, from net.Addr) error {
	if smp.scratch == nil {
		smp.scratch = seeker.NewBufferedPeeker(nil)
	}

	smp.scratch.Reset(d)

	for smp.scratch.Area() > 0 {
		nextdata := smp.scratch.Next(HeaderLength)
		if len(nextdata) < HeaderLength {
			smp.scratch.Reset(nil)
			return ErrHeaderLength
		}

		nextSize := int(binary.BigEndian.Uint32(nextdata))

		// If scratch is zero and we do have count data, maybe we face a unfinished write.
		if smp.scratch.Area() == 0 {
			smp.scratch.Reset(nil)
			return ErrInvalidHeader
		}

		if nextSize > smp.scratch.Area() {
			smp.scratch.Reset(nil)
			return ErrInvalidHeader
		}

		next := smp.scratch.Next(nextSize)

		// copy data to avoid corruption bug.
		nextCopy := make([]byte, len(next))
		n := copy(nextCopy, next)

		msg := messagePool.New().(*messageTomb)
		msg.Target = from
		msg.Data = nextCopy[:n]
		smp.addMessage(msg)
	}

	return nil
}

// Next returns the next message saved on the parsers linked list.
func (smp *TaggedMessages) Next() ([]byte, net.Addr, error) {
	smp.mu.RLock()
	if smp.tail == nil && smp.head == nil {
		smp.mu.RUnlock()
		return nil, nil, ErrNoDataYet
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
	target := head.Target

	head.Target = nil
	head.Data = nil
	smp.mu.RUnlock()

	messagePool.Put(head)

	return data, target, nil
}

func (smp *TaggedMessages) addMessage(m *messageTomb) {
	smp.mu.Lock()

	if smp.head == nil && smp.tail == nil {
		smp.head = m
		smp.tail = m
		smp.mu.Unlock()
		return
	}

	smp.tail.Next = m
	smp.tail = m
	smp.mu.Unlock()
}

//// MessageDivision implements a parser which parses incoming messages
//// as a series of [MessageSizeLength][Message] joined together, which it
//// splits apart into individual messageTomb blocks. The parser keeps a series of internal
//// linked list which contains already processed messageTomb, which has a endless length
//// value which allows appending new messages as they arrive.
//type MessageDivision struct {
//	division io.WriteCloser
//	mu       sync.RWMutex
//	head     *messageTomb
//	tail     *messageTomb
//	total    int64
//}
//
//// Parse implements the necessary procedure for parsing incoming data and
//// appropriately splitting them accordingly to their respective parts.
//func (md *MessageDivision) Parse(d []byte) error {
//	if md.division == nil {
//
//		// if first data coming in, is not a uint16 header, then respond
//		// with error.
//		if len(d) != HeaderLength {
//			return ErrHeaderLength
//		}
//
//		incoming := binary.BigEndian.Uint32(d)
//		if incoming > MaxHeaderSize {
//			return ErrInvalidHeader
//		}
//
//		md.division = bufferPool.Get(int(incoming), func(rec int, w io.WriterTo) error {
//			bu := bytes.NewBuffer(make([]byte, 0, rec))
//			if _, err := w.WriteTo(bu); err != nil {
//				return err
//			}
//
//			msg := messagePool.New().(*messageTomb)
//			msg.Data = bu.Bytes()
//			md.addMessage(msg)
//
//			bu.Reset()
//			return nil
//		})
//	}
//
//	if _, err := md.division.Write(d); err != nil {
//		if err != done.ErrLimitExceeded {
//			return err
//		}
//
//		md.division.Close()
//		md.division = nil
//	}
//
//	return nil
//}
//
//// Next returns the next messageTomb saved on the parsers linked list.
//func (md *MessageDivision) Next() ([]byte, error) {
//	md.mu.RLock()
//
//	if md.tail == nil && md.head == nil {
//		md.mu.RUnlock()
//		return nil, ErrNoDataYet
//	}
//
//	head := md.head
//	if md.tail == head {
//		md.tail = nil
//		md.head = nil
//	} else {
//		next := head.Next
//		head.Next = nil
//		md.head = next
//	}
//
//	data := head.Data
//
//	head.Data = nil
//	md.mu.RUnlock()
//
//	messagePool.Put(head)
//
//	return data, nil
//}
//
//func (md *MessageDivision) addMessage(m *messageTomb) {
//	md.mu.Lock()
//	if md.head == nil && md.tail == nil {
//		md.head = m
//		md.tail = m
//		md.mu.Unlock()
//		return
//	}
//
//	md.tail.Next = m
//	md.tail = m
//	md.mu.Unlock()
//}
