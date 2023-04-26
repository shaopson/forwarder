package main

import (
	"errors"
	"fmt"
	"github.com/dev-shao/iprp/log"
	"github.com/dev-shao/iprp/message"
	"io"
	"net"
	"sync"
	"time"
)

var ErrMgrClosed = errors.New("Manager is closed")
var ErrHBTimeout = errors.New("Heartbeat timeout")
var ErrRecvChClosed = errors.New("Receiver channel closed")
var ErrSendChClosed = errors.New("Send channel closed")

var MaxPipeCount int = 6

// 一个 Manager 管理一个客户端
// 处理请求; 管理这个客户端的所有pipe和proxy;
type Manager struct {
	ClientId          string
	conn              net.Conn //control message connect
	recvChan          chan message.Message
	sendChan          chan message.Message
	proxies           map[string]*Proxy
	mutex             sync.RWMutex
	pipeConnPool      chan net.Conn
	PipeCount         int
	Meta              *message.LoginRequest
	heartbeatTimer    time.Ticker
	heartbeatInterval time.Duration
	lastBeatTime      time.Time
	closed            bool
	closedMutex       sync.RWMutex
	done              chan *Manager //notice server close manager
}

func NewManager(id string, conn net.Conn, done chan *Manager, req *message.LoginRequest) *Manager {
	manager := &Manager{
		ClientId:  id,
		conn:      conn,
		PipeCount: 1,
		Meta:      req,
		done:      done,
		proxies:   make(map[string]*Proxy),
		recvChan:  make(chan message.Message, 10),
		sendChan:  make(chan message.Message, 10),
	}
	if req.PipeCount > 0 {
		manager.PipeCount = req.PipeCount
	}
	if manager.PipeCount > MaxPipeCount {
		manager.PipeCount = MaxPipeCount
	}
	manager.pipeConnPool = make(chan net.Conn, req.PipeCount*2)
	return manager
}

func (self *Manager) Run() {
	defer func() {
		self.done <- self //notice server close manager
	}()
	go self.awaitReceive()
	go self.awaitSend()

	for i := 0; i < self.PipeCount; i++ {
		msg := &message.PipeMessage{
			ClientId: self.ClientId,
		}
		_ = self.SendMassage(msg)
	}

	for {
		msg, err := self.getMessage()
		if err != nil {
			return
		}
		var resp message.Message

		switch req := msg.(type) {
		case *message.HeartbeatRequest:
			log.Debug("%s receive heartbeat", self)
			resp = &message.HeartbeatResponse{
				Timestamp: time.Now().Unix(),
				Result:    "ok",
			}
		case *message.ProxyRequest:
			if req.Enable { // new proxy
				resp = self.RegisterProxy(req)
			} else { // close proxy
				self.DeregisterProxy(req)
			}
		}

		if resp != nil {
			if err := self.SendMassage(resp); err != nil {
				return
			}
		}
	}
}

func (self *Manager) RegisterProxy(req *message.ProxyRequest) *message.ProxyResponse {
	proxy, err := NewProxy(req.ProxyName, req.ProxyPort, self)
	if err != nil {
		resp := &message.ProxyResponse{
			Result: fmt.Sprintf("register proxy error:%s", err),
		}
		return resp
	}

	go proxy.Run()

	self.mutex.Lock()
	self.proxies[proxy.Name()] = proxy
	self.mutex.Unlock()
	log.Info("%s register proxy:%s", self, proxy)

	return &message.ProxyResponse{
		ProxyName: proxy.Name(),
		ProxyPort: proxy.Port(),
		Result:    "ok",
	}
}

func (self *Manager) DeregisterProxy(req *message.ProxyRequest) {
	name := req.ProxyName
	self.mutex.Lock()
	proxy, ok := self.proxies[name]
	if ok {
		delete(self.proxies, name)
	}
	self.mutex.Unlock()
	if !ok {
		return
	}
	proxy.Close()
	log.Info("%s deregister proxy:%s", self, proxy)
}

func (self *Manager) PushPipeConn(conn net.Conn) error {
	defer func() {
		if err := recover(); err != nil {
			log.Error("panic:%s", err)
		}
	}()
	if self.IsClosed() {
		return ErrMgrClosed
	}
	select {
	case self.pipeConnPool <- conn:
		log.Debug("%s push a new pipe to pool", self)
	default:
		log.Warn("%s pipe pool is full", self)
		return errors.New("pipe pool is full")
	}
	return nil
}

func (self *Manager) PopPipeConn() (conn net.Conn, err error) {
	var ok bool
	select {
	case conn, ok = <-self.pipeConnPool:
		if !ok {
			err = ErrMgrClosed
			return
		}
		log.Debug("%s pop a pipe", self.ClientId)
	default:
		// pool is empty
		// send pipe request to client to create new pipe
		req := &message.PipeMessage{}
		if err = self.SendMassage(req); err != nil {
			return
		}

		select {
		case conn, ok = <-self.pipeConnPool:
			if !ok {
				err = ErrMgrClosed
				return
			}
			log.Debug("%s pop a pipe", self.ClientId)
		case <-time.After(time.Second * 5):
			err = errors.New("await get new pipe timeout")
			log.Warn("%s await get new pipe timeout", self.ClientId)
		}
	}
	return
}

func (self *Manager) awaitReceive() {
	defer func() {
		if err := recover(); err != nil {
			log.Error("panic:%s", err)
		}
	}()
	if self.IsClosed() {
		return
	}
	for {
		if res, err := message.Get(self.conn); err != nil {
			if err == io.EOF {
				log.Debug("%s connect closed", self)
			} else {
				log.Warn("%s get message error:%s", self, err)
			}
			self.Close()
			return
		} else {
			if self.IsClosed() {
				return
			}
			self.recvChan <- res
		}
	}
}

func (self *Manager) awaitSend() {
	for {
		resp, ok := <-self.sendChan
		if !ok {
			log.Debug("send channel close")
			break
		}
		err := message.Send(resp, self.conn)
		if err != nil {
			log.Warn("%s send message error:%s", self, err)
			break
		}
	}
}

func (self *Manager) getMessage() (msg message.Message, err error) {
	var ok bool
	select {
	case msg, ok = <-self.recvChan:
		if !ok {
			err = ErrRecvChClosed
		}
	case <-self.heartbeatTimer.C:
		if time.Since(self.lastBeatTime) > self.heartbeatInterval {
			log.Info("%s wait receiver heartbeat timeout", self)
			err = ErrHBTimeout
		}
	}
	return
}

func (self *Manager) SendMassage(msg message.Message) error {
	defer func() {
		if err := recover(); err != nil {
			log.Error("panic:%s", err)
		}
	}()
	if self.IsClosed() {
		return ErrMgrClosed
	}
	self.sendChan <- msg
	return nil
}

func (self *Manager) IsClosed() bool {
	self.closedMutex.RLock()
	defer self.closedMutex.RUnlock()
	return self.closed
}

func (self *Manager) Close() {
	if self.IsClosed() {
		return
	}
	self.closedMutex.Lock()
	self.closed = true
	self.closedMutex.Unlock()

	self.conn.Close()

	close(self.sendChan)
	close(self.recvChan)
	close(self.pipeConnPool)

	self.mutex.Lock()
	for k, proxy := range self.proxies {
		proxy.Close()
		delete(self.proxies, k)
	}
	self.mutex.Unlock()

	log.Debug("close %s", self)
}

func (self *Manager) String() string {
	return fmt.Sprintf("<Manager:%s>", self.ClientId)
}
