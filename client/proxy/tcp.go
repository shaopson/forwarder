package proxy

import (
	"fmt"
	"github.com/hashicorp/yamux"
	"github.com/shaopson/forwarder/pipe"
	log "github.com/shaopson/grlog"
	"io"
	"net"
)

type TCPProxy struct {
	name       string
	config     *Config
	localAddr  string
	remoteAddr string
	conn       net.Conn
	session    *yamux.Session
}

func NewTCPProxy(cfg *Config, conn net.Conn) (*TCPProxy, error) {
	return &TCPProxy{
		name:       cfg.Name,
		config:     cfg,
		localAddr:  net.JoinHostPort(cfg.LocalIp, cfg.LocalPort),
		remoteAddr: net.JoinHostPort(cfg.RemoteIp, cfg.RemotePort),
		conn:       conn,
	}, nil
}

func (self *TCPProxy) Run() {
	cfg := yamux.DefaultConfig()
	cfg.LogOutput = io.Discard
	session, err := yamux.Server(self.conn, cfg)
	if err != nil {
		log.Error("init yamux server error:%s", err)
		return
	}
	self.session = session
	log.Info("%s start", self)
	for {
		conn, err := session.Accept()
		if err != nil {
			log.Error("session closed")
			return
		}
		log.Info("%s accept connect", self)
		go self.handleConnection(conn)
	}
}

func (self *TCPProxy) handleConnection(conn net.Conn) {
	targetConn, err := net.Dial("tcp", self.localAddr)
	if err != nil {
		conn.Close()
		log.Error("%s connect target failure:%s", self, err)
		return
	}
	pipe.Join(conn, targetConn)
}

func (self *TCPProxy) Close() {
	self.session.Close()
	self.conn.Close()
	log.Info("%s close", self)
}

func (self *TCPProxy) Name() string {
	return self.name
}

func (self *TCPProxy) LocalAddr() string {
	return self.localAddr
}

func (self *TCPProxy) RemoteAddr() string {
	return self.remoteAddr
}

func (self *TCPProxy) String() string {
	return fmt.Sprintf("tcp proxy '%s'", self.name)
}
