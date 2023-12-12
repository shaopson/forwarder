package message

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
)

func Send(any interface{}, writer io.Writer) error {
	data, err := encode(any)
	if err != nil {
		return err
	}
	_, err = writer.Write(data)
	return err
}

func Read(reader io.Reader) (Message, error) {
	buf := make([]byte, 2)
	n, err := reader.Read(buf)
	if err != nil {
		return nil, err
	}
	flag := string(buf[:n])
	var length int32
	if err := binary.Read(reader, binary.BigEndian, &length); err != nil {
		return nil, err
	}
	data := make([]byte, length)
	if _, err := io.ReadFull(reader, data); err != nil {
		return nil, err
	}
	return decode(flag, data)
}

func encode(any interface{}) ([]byte, error) {
	typ := reflect.TypeOf(any)
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	flag, ok := typeFlagMap[typ]
	if !ok {
		return nil, fmt.Errorf("Not support Message type:%s", typ)
	}
	data, err := json.Marshal(any)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(nil)
	buf.WriteString(flag)
	binary.Write(buf, binary.BigEndian, int32(len(data)))
	buf.Write(data)
	return buf.Bytes(), nil
}

func decode(flag string, data []byte) (Message, error) {
	typ, ok := flagTypeMap[flag]
	if !ok {
		return nil, errors.New("Unknown message flag:" + flag)
	}
	msg := reflect.New(typ).Interface()
	if err := json.Unmarshal(data, msg); err != nil {
		return nil, err
	}
	return msg, nil
}
