package message

import (
	"net"
	"reflect"
)

type Message interface{}

const (
	AuthRequestFlag   = "<A"
	AuthResponseFlag  = ">A"
	ProxyRequestFlag  = "<P"
	ProxyResponseFlag = ">P"
	PingFlag          = "<p"
	PongFlag          = ">q"
	CloseProxyFlag    = "<C"
	UDPMessageFlag    = "UM"
)

type AuthRequest struct {
	AuthToken string `json:"auth_token,omitempty"`
	HostName  string `json:"host_name,omitempty"`
	Os        string `json:"os,omitempty"`
}

type AuthResponse struct {
	ClientId string `json:"client_id,omitempty"`
	Result   string `json:"result,omitempty"`
}

type ProxyRequest struct {
	ClientId   string `json:"client_id,omitempty"`
	ProxyName  string `json:"name,omitempty"`
	ProxyType  string `json:"type,omitempty"`
	ProxyIp    string `json:"ip,omitempty"`
	ProxyPort  string `json:"port,omitempty"`
	RemoteIp   string `json:"remote_ip,omitempty"`
	RemotePort string `json:"remote_port"`
}

type ProxyResponse struct {
	ProxyName string `json:"name,omitempty"`
	Result    string `json:"result,omitempty"`
}

type CloseProxy struct {
	ClientId  string `json:"client_id,omitempty"`
	ProxyName string `json:"name,omitempty"`
}

type Ping struct {
	ClientId  string `json:"client_id,omitempty"`
	Timestamp int64  `json:"timestamp,omitempty"`
}

type Pong struct {
	Timestamp int64 `json:"timestamp,omitempty"`
}

//type TCPConnect struct {
//
//}

type UDPMessage struct {
	RemoteAddr *net.UDPAddr `json:"r,omitempty"`
	Content    []byte       `json:"c,omitempty"`
}

var flagMessageMap = map[string]interface{}{
	AuthRequestFlag:   AuthRequest{},
	AuthResponseFlag:  AuthResponse{},
	ProxyRequestFlag:  ProxyRequest{},
	ProxyResponseFlag: ProxyResponse{},
	PingFlag:          Ping{},
	PongFlag:          Pong{},
	CloseProxyFlag:    CloseProxy{},
	UDPMessageFlag:    UDPMessage{},
}

var typeFlagMap = map[reflect.Type]string{}
var flagTypeMap = map[string]reflect.Type{}

func init() {
	for flag, msg := range flagMessageMap {
		typ := reflect.TypeOf(msg)
		typeFlagMap[typ] = flag
	}
	for typ, flag := range typeFlagMap {
		flagTypeMap[flag] = typ
	}
}
