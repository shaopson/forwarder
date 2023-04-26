package main

import (
	"fmt"
	"github.com/dev-shao/iprp/log"
	"github.com/dev-shao/iprp/message"
	"github.com/dev-shao/iprp/pipe"
	"io"
	"net"
	"sync"
	"time"
)

type Manager struct {
	Client            *Client
	ClientId          string
	proxyConfigs      map[string]*ProxyConfig
	conn              net.Conn
	pipeConnPool      chan net.Conn
	agents            map[string]*Agent
	recvChan          chan message.Message
	sendChan          chan message.Message
	heartbeatInterval time.Duration
	lastBeatTime      time.Time
	closed            bool
	closedMutex       sync.RWMutex
	mutex             sync.RWMutex
}

func NewManager(conn net.Conn, client *Client, cfgs map[string]*ProxyConfig) *Manager {
	manager := &Manager{
		Client:            client,
		ClientId:          client.ClientId,
		conn:              conn,
		proxyConfigs:      make(map[string]*ProxyConfig),
		agents:            make(map[string]*Agent),
		recvChan:          make(chan message.Message, 10),
		sendChan:          make(chan message.Message, 10),
		heartbeatInterval: time.Second * 20,
	}
	for name, proxy := range cfgs {
		manager.proxyConfigs[name] = proxy
	}
	return manager
}

func (self *Manager) Run() {
	defer self.Close()
	go self.awaitReceive()
	go self.awaitSend()

	for _, cfg := range self.proxyConfigs {
		err := self.RegisterProxy(cfg)
		if err != nil {
			log.Warn("register proxy error:%s", err)
		}
	}

	ticker := time.NewTicker(self.heartbeatInterval)
	self.lastBeatTime = time.Now().Add(self.heartbeatInterval / 2)
	for {
		select {
		case <-ticker.C:
			if time.Since(self.lastBeatTime) > self.heartbeatInterval {
				log.Warn("heartbeat timeout")
				return
			}
			req := &message.HeartbeatRequest{
				ClientId:  self.ClientId,
				Timestamp: time.Now().Unix(),
			}
			self.sendChan <- req
			log.Debug("send heartbeat")
		case msg, ok := <-self.recvChan:
			if !ok {
				log.Debug("receiver channel closed")
				return
			}

			switch v := msg.(type) {
			case *message.PipeMessage:
				conn, err := self.Client.NewPipeConn()
				if err != nil {
					log.Warn(" create new pipe connect fail:%s", err)
				} else {
					go self.awaitWork(conn)
				}
			case *message.ProxyResponse:
				if v.Result == "ok" {
					log.Info("register proxy[%s] success", v.ProxyName)
					self.mutex.RLock()
					agent, ok := self.agents[v.ProxyName]
					self.mutex.RUnlock()
					if ok {
						agent.Start()
					} else {
						log.Warn("agent not found:%s", v.ProxyName)
					}
				} else {
					log.Warn("register proxy[%s] fail:%s", v.ProxyName, v.Result)
				}
			case *message.HeartbeatResponse:
				log.Debug("heartbeat response")
				self.lastBeatTime = time.Now()
			default:
				log.Warn("unknown message")
			}
		}
	}

}

func (self *Manager) awaitWork(conn net.Conn) {
	defer conn.Close()
	msg, err := message.Get(conn)
	if err != nil {
		log.Debug("work connect closed")
		return
	}
	workMsg, ok := msg.(*message.WorkMessage)
	if !ok {
		return
	}
	log.Info("new work:%s", workMsg)
	//find proxy
	self.mutex.RLock()
	agent, ok := self.agents[workMsg.ProxyName]
	self.mutex.RUnlock()
	if !ok {
		log.Warn("[agent not found:%s", workMsg.ProxyName)
		return
	}
	if !agent.Started {
		log.Warn("agent not started:%s", agent.Name)
		return
	}
	dstConn, err := agent.Connect()
	if err != nil {
		log.Warn("agent connect target service fail:%s", err)
		return
	}
	pipe.Join(conn, dstConn)

}

func (self *Manager) RegisterProxy(cfg *ProxyConfig) error {
	agent, err := NewAgent(cfg)
	if err != nil {
		log.Warn("create new proxy agent error:%s", err)
		return err
	}
	req := &message.ProxyRequest{
		ProxyName: agent.Name,
		ProxyType: agent.Type,
		ProxyPort: agent.RemotePort,
		Enable:    true,
	}
	err = message.Send(req, self.conn)
	if err != nil {
		return err
	}
	//
	self.mutex.Lock()
	self.agents[agent.Name] = agent
	self.mutex.Unlock()
	log.Info("register new proxy:%s", agent.Name)
	return nil
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
		if msg, err := message.Get(self.conn); err != nil {
			if err == io.EOF {
				log.Debug("manager connect closed")
			} else {
				log.Warn("get message error:%s", err)
			}
			break
		} else {
			if self.IsClosed() {
				break
			}
			self.recvChan <- msg
		}
	}
}

func (self *Manager) awaitSend() {
	for {
		msg, ok := <-self.sendChan
		if !ok { //manager closed
			break
		}
		err := message.Send(msg, self.conn)
		if err != nil {
			log.Warn("send message error:%s", err)
			break
		}
	}
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
	close(self.recvChan)
	close(self.sendChan)
	self.conn.Close()

	log.Info("%s close", self)
}

func (self *Manager) String() string {
	return fmt.Sprintf("<Manager:%s>", self.ClientId)
}
