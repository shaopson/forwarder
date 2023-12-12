package proxy

import (
	"fmt"
	"github.com/shaopson/forwarder/message"
	log "github.com/shaopson/grlog"
	"net"
	"sync"
)

var UDPBufSize = 4096

type UDPProxy struct {
	name       string
	localAddr  string
	remoteAddr string
	pipe       net.Conn
	udpConn    *net.UDPConn
	readChan   chan *message.UDPMessage
	sendChan   chan *message.UDPMessage
	doneChan   chan struct{}
	once       sync.Once
}

func NewUDPProxy(name, laddr, raddr string, conn net.Conn) (*UDPProxy, error) {
	localAddr, err := net.ResolveUDPAddr("udp", laddr)
	if err != nil {
		return nil, err
	}
	udpConn, err := net.ListenUDP("udp", localAddr)
	if err != nil {
		return nil, err
	}
	p := &UDPProxy{
		name:       name,
		localAddr:  laddr,
		remoteAddr: raddr,
		pipe:       conn,
		udpConn:    udpConn,
		readChan:   make(chan *message.UDPMessage, 20),
		sendChan:   make(chan *message.UDPMessage, 20),
		doneChan:   make(chan struct{}),
		once:       sync.Once{},
	}
	return p, nil
}

func (self *UDPProxy) Run() {
	log.Info("%s listen on %s", self, self.localAddr)

	go self.awaitUDPSend()
	go self.awaitUDPRead()
	go self.awaitPipeSend()
	go self.awaitPipeRead()

	<-self.doneChan
}

func (self *UDPProxy) Close() {
	self.pipe.Close()
	self.udpConn.Close()
	log.Info("%s close", self)
}

func (self *UDPProxy) awaitUDPRead() {
	defer self.done()
	defer close(self.readChan)
	buf := make([]byte, UDPBufSize)
	for {
		n, addr, err := self.udpConn.ReadFromUDP(buf)
		if err != nil {
			log.Warn("%s", err)
			return
		}
		msg := &message.UDPMessage{
			Content:    buf[:n],
			RemoteAddr: addr,
		}
		self.readChan <- msg
	}
}

func (self *UDPProxy) awaitUDPSend() {
	for msg := range self.sendChan {
		self.udpConn.WriteToUDP(msg.Content, msg.RemoteAddr)
	}
}

func (self *UDPProxy) awaitPipeRead() {
	defer self.done()
	defer close(self.sendChan)
	for {
		msg, err := message.Read(self.pipe)
		if err != nil {
			log.Warn("%s pipe read error:%s", self, err)
			return
		}
		if m, ok := msg.(*message.UDPMessage); !ok {
			log.Warn("%s pipe read invalid message", self)
			continue
		} else {
			self.sendChan <- m
		}
	}
}

func (self *UDPProxy) awaitPipeSend() {
	for msg := range self.readChan {
		if err := message.Send(msg, self.pipe); err != nil {
			log.Warn("%s pipe send error:%s", self, err)
			return
		}
	}
}

func (self *UDPProxy) done() {
	self.once.Do(func() {
		self.doneChan <- struct{}{}
		close(self.doneChan)
	})
}

func (self *UDPProxy) Name() string {
	return ""
}

func (self *UDPProxy) LocalAddr() string {
	return self.localAddr
}

func (self *UDPProxy) RemoteAddr() string {
	return self.remoteAddr
}

func (self *UDPProxy) String() string {
	return fmt.Sprintf("udp proxy '%s'", self.name)
}
