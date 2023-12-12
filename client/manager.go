package client

import (
	"errors"
	"github.com/shaopson/forwarder/client/proxy"
	"github.com/shaopson/forwarder/message"
	"github.com/shaopson/forwarder/pipe/crypto"
	log "github.com/shaopson/grlog"
	"io"
	"net"
	"sync"
	"time"
)

var HeartBeatTimeout = time.Second * 60
var HeartbeatInterval = time.Second * 30

type Manager struct {
	ClientId     string
	proxyConfigs map[string]*proxy.Config
	authToken    string
	conn         net.Conn
	proxies      map[string]proxy.Proxy
	mutex        sync.RWMutex
	readChan     chan message.Message
	heartbeat    <-chan time.Time
	lastBeat     time.Time
}

func NewManager(clientId string, conn net.Conn, config *Config) *Manager {
	return &Manager{
		ClientId:     clientId,
		proxyConfigs: config.Proxies,
		authToken:    config.AuthToken,
		conn:         conn,
		proxies:      make(map[string]proxy.Proxy),
		readChan:     make(chan message.Message, 2),
	}
}

func (self *Manager) Run() {
	log.Info("manager start")

	go self.awaitRead()
	go self.registerAllProxy(self.proxyConfigs)

	self.heartbeat = time.Tick(HeartbeatInterval)
	self.lastBeat = time.Now()

	for {
		select {
		case <-self.heartbeat:
			if time.Since(self.lastBeat) > HeartBeatTimeout {
				log.Error("heartbeat timeout")
				return
			}
			ping := &message.Ping{
				ClientId:  self.ClientId,
				Timestamp: time.Now().Unix(),
			}
			message.Send(ping, self.conn)
			log.Debug("send ping")
		case msg, ok := <-self.readChan:
			if !ok {
				log.Error("read chan closed")
				return
			}
			switch msg.(type) {
			case *message.Pong:
				self.lastBeat = time.Now()
				log.Debug("received pong")
			}
		}
	}
}

func (self *Manager) registerAllProxy(configs map[string]*proxy.Config) {
	failures := make(map[string]*proxy.Config)
	for name, cfg := range self.proxyConfigs {
		time.Sleep(time.Millisecond * 200)
		if err := self.registerProxy(cfg); err != nil {
			log.Error("register proxy '%s' failure:%s", name, err)
			log.Info("proxy %s waiting try again", name)
			failures[name] = cfg
		} else {
			log.Info("register proxy '%s' success", name)
		}
	}
	time.Sleep(time.Second)
	for name, cfg := range failures {
		time.Sleep(time.Millisecond * 200)
		if err := self.registerProxy(cfg); err != nil {
			log.Error("again to register proxy '%s' failure:%s", name, err)
		} else {
			log.Info("register proxy '%s' success")
		}
	}
}

func (self *Manager) registerProxy(cfg *proxy.Config) error {
	addr := self.conn.RemoteAddr().String()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	conn, err = crypto.NewConn(conn, self.authToken)
	if err != nil {
		return err
	}

	req := &message.ProxyRequest{
		ClientId:   self.ClientId,
		ProxyName:  cfg.Name,
		ProxyType:  cfg.ProxyType,
		ProxyIp:    cfg.RemoteIp,
		ProxyPort:  cfg.RemotePort,
		RemoteIp:   cfg.LocalIp,
		RemotePort: cfg.LocalPort,
	}
	message.Send(req, conn)

	conn.SetReadDeadline(time.Now().Add(MessageReadTimeout))
	msg, err := message.Read(conn)
	if err != nil {
		return err
	}
	conn.SetReadDeadline(time.Time{})

	if resp, ok := msg.(*message.ProxyResponse); !ok {
		return errors.New("invalid proxy response")
	} else if resp.Result != "ok" {
		return errors.New(resp.Result)
	}

	//new proxy
	var pry proxy.Proxy
	switch cfg.ProxyType {
	case "tcp":
		pry, err = proxy.NewTCPProxy(cfg, conn)
	case "udp":
		pry, err = proxy.NewUDPProxy(cfg, conn)
	}
	if err != nil {
		return err
	}
	self.mutex.Lock()
	defer self.mutex.Unlock()
	old, ok := self.proxies[cfg.Name]
	if ok {
		old.Close()
		delete(self.proxies, cfg.Name)
	}
	self.proxies[cfg.Name] = pry

	go func() {
		pry.Run()
		self.closeProxy(cfg.Name)
	}()

	return nil
}

func (self *Manager) closeProxy(name string) {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	if p, ok := self.proxies[name]; ok {
		delete(self.proxies, name)
		p.Close()
	}

	msg := &message.CloseProxy{
		ClientId:  self.ClientId,
		ProxyName: name,
	}
	message.Send(msg, self.conn)
	log.Info("manager close proxy '%s'", name)
}

func (self *Manager) Close() {
	//close proxies
	for name, _ := range self.proxies {
		self.closeProxy(name)
	}
	self.conn.Close()
	log.Info("manager close")
}

func (self *Manager) awaitRead() {
	defer close(self.readChan)
	for {
		if msg, err := message.Read(self.conn); err != nil {
			if err == io.EOF {
				log.Info("server connection closed")
			} else {
				log.Error("read message failure from connection %s:%s", self.conn.RemoteAddr(), err)
			}
			return
		} else {
			self.readChan <- msg
		}
	}
}
