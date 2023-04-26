package main

import (
	"fmt"
	"github.com/dev-shao/iprp/log"
	"github.com/dev-shao/iprp/message"
	"github.com/dev-shao/iprp/pipe"
	"net"
)

type Proxy struct {
	manager  *Manager
	name     string
	port     string
	listener net.Listener
}

func NewProxy(name string, port string, mgr *Manager) (*Proxy, error) {
	proxy := &Proxy{
		manager: mgr,
		name:    name,
		port:    port,
	}
	addr := net.JoinHostPort("", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	proxy.listener = listener
	log.Info("%s listen on:%s", proxy.name, proxy.port)
	return proxy, nil
}

func (self *Proxy) Name() string {
	return self.name
}

func (self *Proxy) Port() string {
	return self.port
}

func (self *Proxy) Run() {
	for {
		conn, err := self.listener.Accept()
		if err != nil {
			log.Info("%s listener closed", self)
			return
		}
		go self.handle(conn)
	}
}

func (self *Proxy) Close() {
	self.listener.Close()
	log.Info("%s closed", self)
}

func (self *Proxy) handle(conn net.Conn) {
	//get pipe connect from manager
	pipeConn, err := self.manager.PopPipeConn()
	if err != nil {
		return
	}
	//send work request
	srcIp, srcPort, _ := net.SplitHostPort(conn.RemoteAddr().String())
	dstIp, dstPort, _ := net.SplitHostPort(conn.LocalAddr().String())
	req := &message.WorkMessage{
		ProxyName: self.name,
		SrcIp:     srcIp,
		SrcPort:   srcPort,
		DstIp:     dstIp,
		DstPort:   dstPort,
	}
	if err := message.Send(req, pipeConn); err != nil {
		log.Warn("%s send work message fail:%s", self, err)
		pipeConn.Close()
		return
	}
	//connect join pipe
	pipe.Join(conn, pipeConn)
}

func (self *Proxy) String() string {
	return fmt.Sprintf("<Proxy:%s>", self.name)
}
