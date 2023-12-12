package rc4

import (
	"crypto/cipher"
	"crypto/rc4"
	"io"
)

func NewWriter(w io.Writer, key string) (io.Writer, error) {
	stream, err := rc4.NewCipher([]byte(key))
	if err != nil {
		return nil, err
	}
	return cipher.StreamWriter{S: stream, W: w}, nil
}

func NewReader(r io.Reader, key string) (io.Reader, error) {
	stream, err := rc4.NewCipher([]byte(key))
	if err != nil {
		return nil, err
	}
	return cipher.StreamReader{S: stream, R: r}, nil
}
