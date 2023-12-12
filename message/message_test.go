package message

import (
	"bytes"
	"testing"
)

func TestEncode(t *testing.T) {
	m := &AuthRequest{
		AuthToken: "auth",
		HostName:  "localhost",
		Os:        "linux",
	}
	data, err := encode(m)
	if err != nil {
		t.Error(err)
	}
	t.Log(data)
}

func TestSend(t *testing.T) {
	m := &AuthRequest{
		AuthToken: "auth",
		HostName:  "localhost",
		Os:        "linux",
	}
	writer := bytes.NewBuffer(make([]byte, 100))
	if err := Send(m, writer); err != nil {
		t.Error(err)
	}
	t.Log(writer.String())
}

func TestRead(t *testing.T) {
	m := &AuthRequest{
		AuthToken: "auth",
		HostName:  "localhost",
		Os:        "linux",
	}
	data, _ := encode(m)
	buf := bytes.NewBuffer(data)
	msg, err := Read(buf)
	if err != nil {
		t.Error(err)
	}
	t.Log(msg)
}
