package server

import (
	"fmt"
	"github.com/shaopson/forwarder/message"
	"github.com/shaopson/forwarder/server/proxy"
	log "github.com/shaopson/grlog"
	"io"
	"net"
	"sync"
	"time"
)

var HeartBeatTimeout = time.Second * 60
var HeartbeatInterval = time.Second * 30

type Manager struct {
	ClientId  string
	Meta      *message.AuthRequest
	conn      net.Conn
	proxies   map[string]proxy.Proxy
	readChan  chan message.Message
	sendChan  chan message.Message
	mutex     sync.RWMutex
	heartbeat <-chan time.Time
	lastBeat  time.Time
}

func NewManager(ClientId string, conn net.Conn, msg *message.AuthRequest) *Manager {
	mgr := &Manager{
		ClientId: ClientId,
		Meta:     msg,
		conn:     conn,
		proxies:  make(map[string]proxy.Proxy),
		readChan: make(chan message.Message, 2),
		//sendChan:  make(chan message.Message, 2),
	}
	return mgr
}

func (self *Manager) Run() {
	log.Info("%s start", self)
	//go self.awaitSend()
	go self.awaitRead()

	self.heartbeat = time.Tick(HeartbeatInterval)
	self.lastBeat = time.Now()

	for {
		select {
		case msg, ok := <-self.readChan:
			if !ok {
				log.Error("%s read chan closed", self)
				return
			}
			switch m := msg.(type) {
			case *message.Ping:
				log.Debug("receive ping form %s", m.ClientId)
				self.lastBeat = time.Now()
				pong := &message.Pong{
					Timestamp: self.lastBeat.Unix(),
				}
				message.Send(pong, self.conn)
				log.Debug("send pong to %s", m.ClientId)
			case *message.CloseProxy:
				log.Info("close proxy request:{clientId:%s name:%s}", m.ClientId, m.ProxyName)
				self.DeregisterProxy(m.ProxyName)
			default:
				log.Warn("receive unknown type message")
			}
		case <-self.heartbeat:
			if time.Since(self.lastBeat) > HeartBeatTimeout {
				log.Error("%s heartbeat timeout", self)
				return
			}
		}
	}
}

func (self *Manager) RegisterProxy(msg *message.ProxyRequest, conn net.Conn) (p proxy.Proxy, err error) {
	self.mutex.Lock()
	defer self.mutex.Unlock()

	old, ok := self.proxies[msg.ProxyName]
	if ok {
		log.Info("proxy name '%s' conflict, close old proxy", msg.ProxyName)
		old.Close()
		delete(self.proxies, old.Name())
	}
	laddr := net.JoinHostPort(msg.ProxyIp, msg.ProxyPort)
	raddr := net.JoinHostPort(msg.RemoteIp, msg.RemotePort)
	switch msg.ProxyType {
	case "tcp":
		p, err = proxy.NewTCPProxy(msg.ProxyName, laddr, raddr, conn)
	case "udp":
		p, err = proxy.NewUDPProxy(msg.ProxyName, laddr, raddr, conn)
	default:
		return nil, fmt.Errorf("not support proxy type '%s'", msg.ProxyType)
	}
	if err != nil {
		return nil, err
	}

	log.Info("'%s' register %s [local]%s == [remote]%s", self.ClientId, p, laddr, raddr)

	go func() {
		// waiting proxy response send done
		time.Sleep(time.Second)
		p.Run()
		self.DeregisterProxy(p.Name())
	}()

	self.proxies[p.Name()] = p

	return p, err
}

func (self *Manager) DeregisterProxy(name string) {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	if p, ok := self.proxies[name]; ok {
		p.Close()
		delete(self.proxies, name)
	}
}

func (self *Manager) Close() {
	self.conn.Close()
	self.mutex.Lock()
	defer self.mutex.Unlock()
	//close proxies
	for k, p := range self.proxies {
		p.Close()
		delete(self.proxies, k)
	}
	log.Info("%s closed", self)
}

func (self *Manager) sendMessage(msg message.Message) {
	self.sendChan <- msg
}

func (self *Manager) readMessage() (msg message.Message, err error) {
	var ok bool
	select {
	case msg, ok = <-self.readChan:
		if !ok {
			err = fmt.Errorf("read chan closed")
		}
	case <-self.heartbeat:
		if time.Since(self.lastBeat) > HeartBeatTimeout {
			err = fmt.Errorf("heartbeat timeout")
		}
	}
	return
}

func (self *Manager) awaitRead() {
	defer close(self.readChan)
	for {
		if msg, err := message.Read(self.conn); err != nil {
			if err == io.EOF {
				log.Info("connection %s closed", self.conn.RemoteAddr())
			} else {
				log.Error("read message failure from connection %s:%s", self.conn.RemoteAddr(), err)
			}
			return
		} else {
			self.readChan <- msg
		}
	}
}

func (self *Manager) awaitSend() {
	for msg := range self.sendChan {
		if err := message.Send(msg, self.conn); err != nil {
			log.Error("send message to %s failure:%s", self.conn.RemoteAddr(), err)
			return
		}
	}
}

func (self *Manager) String() string {
	return fmt.Sprintf("manager '%s'", self.ClientId)
}
