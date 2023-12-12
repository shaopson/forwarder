package crypto

import (
	"crypto/cipher"
	"crypto/rc4"
	"net"
	"time"
)

type Conn struct {
	stream cipher.Stream
	conn   net.Conn
}

func NewConn(conn net.Conn, key string) (*Conn, error) {
	stream, err := rc4.NewCipher([]byte(key))
	if err != nil {
		return nil, err
	}
	return &Conn{
		stream: stream,
		conn:   conn,
	}, nil
}

func (self *Conn) Read(b []byte) (n int, err error) {
	n, err = self.conn.Read(b)
	self.stream.XORKeyStream(b[:n], b[:n])
	return n, err
}

func (self *Conn) Write(b []byte) (n int, err error) {
	c := make([]byte, len(b))
	self.stream.XORKeyStream(c, b)
	return self.conn.Write(c)
}

func (self *Conn) Close() error {
	return self.conn.Close()
}

func (self *Conn) LocalAddr() net.Addr {
	return self.conn.LocalAddr()
}

func (self *Conn) RemoteAddr() net.Addr {
	return self.conn.RemoteAddr()
}

func (self *Conn) SetDeadline(t time.Time) error {
	return self.conn.SetDeadline(t)
}

func (self *Conn) SetReadDeadline(t time.Time) error {
	return self.conn.SetReadDeadline(t)
}

func (self *Conn) SetWriteDeadline(t time.Time) error {
	return self.conn.SetWriteDeadline(t)
}
