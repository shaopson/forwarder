package message

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"reflect"
)

var ErrInvalidMsg = errors.New("Invalid Message")
var ErrUnknownMsg = errors.New("unknown Message Type")

const (
	RequestFlag   = '<'
	ResponseFlag  = '>'
	TypeHeartbeat = 'h'
	TypeLogin     = 'l'
	TypeProxy     = 'p'
	TypePipe      = 'q'
	TypeWork      = 'w'
)

var TypeRequMap = map[byte]interface{}{
	TypeHeartbeat: HeartbeatRequest{},
	TypeLogin:     LoginRequest{},
	TypeProxy:     ProxyRequest{},
	TypePipe:      PipeMessage{},
	TypeWork:      WorkMessage{},
}

var TypeRespMap = map[byte]interface{}{
	TypeHeartbeat: HeartbeatResponse{},
	TypeLogin:     LoginResponse{},
	TypeProxy:     ProxyResponse{},
	TypePipe:      PipeMessage{},
	TypeWork:      WorkMessage{},
}

type Message interface{}

// login server
type LoginRequest struct {
	Version   uint8  `json:"version,omitempty"`
	AuthCode  string `json:"auth_code,omitempty"`
	PipeCount int    `json:"pipe_count,omitempty"`
	HostName  string `json:"host_name,omitempty"`
	Os        string `json:"os,omitempty"`
}

type LoginResponse struct {
	ClientId string `json:"client_id,omitempty"`
	Result   string `json:"result,omitempty"`
}

// register proxy
type ProxyRequest struct {
	ProxyName string `json:"name,omitempty"`
	ProxyType string `json:"type,omitempty"`
	ProxyPort string `json:"port,omitempty"`
	Enable    bool   `json:"enable,omitempty"`
}

type ProxyResponse struct {
	ProxyName string `json:"name,omitempty"`
	ProxyPort string `json:"port,omitempty"`
	Enable    bool   `json:"enable,omitempty"`
	Result    string `json:"result,omitempty"`
}

// heartbeat
type HeartbeatRequest struct {
	ClientId  string `json:"client_id,omitempty"`
	Timestamp int64  `json:"timestamp,omitempty"`
}

type HeartbeatResponse struct {
	Timestamp int64  `json:"timestamp,omitempty"`
	Result    string `json:"result,omitempty"`
}

type PipeMessage struct {
	ClientId string `json:"client_id,omitempty"`
}

type WorkMessage struct {
	ProxyName string `json:"name,omitempty"`
	SrcIp     string `json:"src_ip,omitempty"`
	SrcPort   string `json:"src_port,omitempty"`
	DstIp     string `json:"dst_ip,omitempty"`
	DstPort   string `json:"dst_port,omitempty"`
}

/*
message: [<|>:1 byte] + [code:1 byte] + [length:4 byte] + [data:(length)]
"<" : request
">" : response
*/

func Write(flag, typ byte, data []byte, writer io.Writer) error {
	buf := bytes.NewBuffer(nil)
	buf.WriteByte(flag)
	buf.WriteByte(typ)
	binary.Write(buf, binary.BigEndian, int32(len(data)))
	buf.Write(data)
	_, err := writer.Write(buf.Bytes())
	return err
}

func Read(reader io.Reader) (flag byte, typ byte, data []byte, err error) {
	buf := make([]byte, 2)
	_, err = reader.Read(buf)
	if err != nil {
		return
	}
	flag = buf[0]
	typ = buf[1]
	var length int32
	if err = binary.Read(reader, binary.BigEndian, &length); err != nil {
		return
	}
	if length < 0 {
		err = ErrInvalidMsg
		return
	}
	data = make([]byte, length)
	n, err := io.ReadFull(reader, data)
	if err != nil {
		err = ErrInvalidMsg
	} else if int32(n) != length {
		err = ErrInvalidMsg
	}
	return
}

func Get(reader io.Reader) (Message, error) {
	flag, typ, data, err := Read(reader)
	if err != nil {
		return nil, err
	}
	var any interface{}
	var ok bool
	if flag == RequestFlag {
		any, ok = TypeRequMap[typ]
	} else if flag == ResponseFlag {
		any, ok = TypeRespMap[typ]
	}
	if !ok {
		return nil, ErrUnknownMsg
	}
	msg := reflect.New(reflect.TypeOf(any)).Interface()
	err = json.Unmarshal(data, msg)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func Send(any interface{}, writer io.Writer) error {
	flag, typ, err := getMsgInfo(any)
	if err != nil {
		return err
	}
	data, err := json.Marshal(any)
	if err != nil {
		return err
	}
	return Write(flag, typ, data, writer)
}

func getMsgInfo(msg interface{}) (flag byte, typ byte, err error) {
	switch msg.(type) {
	case *LoginRequest:
		flag = RequestFlag
		typ = TypeLogin
	case *LoginResponse:
		flag = ResponseFlag
		typ = TypeLogin
	case *HeartbeatRequest:
		flag = RequestFlag
		typ = TypeHeartbeat
	case *HeartbeatResponse:
		flag = ResponseFlag
		typ = TypeHeartbeat
	case *ProxyRequest:
		flag = RequestFlag
		typ = TypeProxy
	case *ProxyResponse:
		flag = ResponseFlag
		typ = TypeProxy
	case *PipeMessage:
		flag = RequestFlag
		typ = TypePipe
	case *WorkMessage:
		flag = RequestFlag
		typ = TypeWork
	default:
		err = ErrUnknownMsg
	}
	return
}
