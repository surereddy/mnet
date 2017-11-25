package mlisten

import "net"

// ConnStream encapsulates giving  underline net.Connection where da
type ConnStream struct {
	net.Conn
}

// ReadByte writes provided byte into underline connection
func (cn ConnStream) ReadByte() (byte, error) {
	content := make([]byte, 1)
	_, err := cn.Conn.Read(content)
	return content[0], err
}

// Read writes provided byte into underline connection
func (cn ConnStream) Read(d int) ([]byte, error) {
	content := make([]byte, d)
	_, err := cn.Conn.Read(content)
	return content, err
}

// WriteByte writes provided byte into underline connection
func (cn ConnStream) WriteByte(bu byte) error {
	_, err := cn.Conn.Write([]byte{bu})
	return err
}
