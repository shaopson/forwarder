package proxy

import (
	"fmt"
	"github.com/shaopson/forwarder/message"
	log "github.com/shaopson/grlog"
	"net"
	"sync"
	"time"
)

var UDPBufSize = 4096
var UDPReadTimeout = time.Second * 30

type UDPProxy struct {
	name            string
	config          *Config
	localAddr       *net.UDPAddr
	remoteAddr      string
	pipe            net.Conn
	readChan        chan *message.UDPMessage
	sendChan        chan *message.UDPMessage
	mutex           sync.RWMutex
	addrConnMapping map[string]*net.UDPConn
}

func NewUDPProxy(cfg *Config, conn net.Conn) (*UDPProxy, error) {
	addr := net.JoinHostPort(cfg.LocalIp, cfg.LocalPort)
	laddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}
	return &UDPProxy{
		name:            cfg.Name,
		config:          cfg,
		pipe:            conn,
		localAddr:       laddr,
		remoteAddr:      net.JoinHostPort(cfg.RemoteIp, cfg.RemotePort),
		readChan:        make(chan *message.UDPMessage, 20),
		sendChan:        make(chan *message.UDPMessage, 20),
		addrConnMapping: make(map[string]*net.UDPConn),
	}, nil
}

func (self *UDPProxy) Run() {
	log.Info("%s start", self)
	go self.awaitPipeSend()
	go self.awaitPipeRead()

	var err error
	for {
		select {
		case msg, ok := <-self.readChan:
			if !ok {
				return
			}
			addr := msg.RemoteAddr.String()
			self.mutex.Lock()
			udpConn, ok := self.addrConnMapping[addr]
			if !ok {
				udpConn, err = net.DialUDP("udp", nil, self.localAddr)
				if err != nil {
					self.mutex.Unlock()
					log.Warn("%s", err)
					continue
				}
				self.addrConnMapping[addr] = udpConn
			}
			self.mutex.Unlock()
			if _, err = udpConn.Write(msg.Content); err != nil {
				self.mutex.Lock()
				delete(self.addrConnMapping, addr)
				self.mutex.Unlock()
				udpConn.Close()
			}
			if !ok {
				go self.awaitRead(udpConn, msg.RemoteAddr)
			}
		}
	}
}

func (self *UDPProxy) awaitRead(udpConn *net.UDPConn, raddr *net.UDPAddr) {
	defer func() {
		self.mutex.Lock()
		delete(self.addrConnMapping, raddr.String())
		self.mutex.Unlock()
		udpConn.Close()
	}()

	buf := make([]byte, UDPBufSize)
	for {
		udpConn.SetReadDeadline(time.Now().Add(UDPReadTimeout))
		n, _, err := udpConn.ReadFromUDP(buf)
		if err != nil {
			return
		}
		udpConn.SetReadDeadline(time.Time{})
		m := &message.UDPMessage{
			Content:    buf[:n],
			RemoteAddr: raddr,
		}
		self.sendChan <- m
	}
}

func (self *UDPProxy) awaitPipeRead() {
	defer close(self.readChan)
	for {
		msg, err := message.Read(self.pipe)
		if err != nil {
			log.Warn("pipe read error:%s", err)
			return
		}
		m, ok := msg.(*message.UDPMessage)
		if !ok {
			log.Warn("receive invalid message")
			continue
		}
		self.readChan <- m
	}
}

func (self *UDPProxy) awaitPipeSend() {
	for msg := range self.sendChan {
		if err := message.Send(msg, self.pipe); err != nil {
			log.Warn("pipe send error:%s", err)
			return
		}
	}
}

func (self *UDPProxy) Close() {
	self.pipe.Close()
	log.Info("%s close", self)
}

func (self *UDPProxy) Name() string {
	return self.name
}

func (self *UDPProxy) LocalAddr() string {
	return self.localAddr.String()
}

func (self *UDPProxy) RemoteAddr() string {
	return self.remoteAddr
}

func (self *UDPProxy) String() string {
	return fmt.Sprintf("udp proxy '%s'", self.name)
}
