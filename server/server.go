package server

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/shaopson/forwarder/message"
	"github.com/shaopson/forwarder/pipe/crypto"
	log "github.com/shaopson/grlog"
	"math/rand"
	"net"
	"sync"
	"time"
)

const MessageReadTimeout = time.Second * 5

type Config struct {
	Ip        string `ini:"bind_ip"`
	Port      string `ini:"bind_port"`
	AuthToken string `ini:"auth_token"`
	LogFile   string `ini:"log_file"`
	LogLevel  string `ini:"log_level"`
}

type Server struct {
	config      *Config
	authToken   string
	address     string
	managers    map[string]*Manager
	mutex       sync.RWMutex
	mgrDoneChan chan *Manager
}

func NewServer(cfg *Config) *Server {
	return &Server{
		config:      cfg,
		address:     net.JoinHostPort(cfg.Ip, cfg.Port),
		authToken:   cfg.AuthToken,
		managers:    make(map[string]*Manager),
		mgrDoneChan: make(chan *Manager, 1),
	}
}

func (self *Server) Run() error {
	listener, err := net.Listen("tcp", self.address)
	if err != nil {
		return err
	}
	defer listener.Close()
	log.Info("forwarder server listening on %s", self.address)
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Error("server close:%s", err)
			return err
		}
		log.Info("accept connection from %s", conn.RemoteAddr())
		go self.handleConnect(conn)
	}
}

func (self *Server) handleConnect(conn net.Conn) {

	conn, err := crypto.NewConn(conn, "forwarder")
	if err != nil {
		log.Error("crypto connection error:%s", err)
		conn.Close()
		return
	}

	_ = conn.SetReadDeadline(time.Now().Add(MessageReadTimeout))
	msg, err := message.Read(conn)
	if err != nil {
		log.Warn("read message failure from connection[%s]:%s", conn.RemoteAddr(), err)
		conn.Close()
		return
	}
	_ = conn.SetReadDeadline(time.Time{})

	switch req := msg.(type) {
	case *message.AuthRequest:
		if err := self.auth(req); err != nil {
			resp := &message.AuthResponse{
				Result: err.Error(),
			}
			message.Send(resp, conn)
			conn.Close()
			log.Info("client '%s' auth failure:%s, close connection", conn.RemoteAddr(), err)
			return
		}
		mgr := self.registerManager(req, conn)
		log.Info("client '%s' auth success, client id:%s", conn.RemoteAddr(), mgr.ClientId)

		resp := &message.AuthResponse{
			ClientId: mgr.ClientId,
			Result:   "ok",
		}
		message.Send(resp, conn)

	case *message.ProxyRequest:
		self.mutex.RLock()
		defer self.mutex.RUnlock()
		mgr, ok := self.managers[req.ClientId]
		if !ok {
			resp := &message.ProxyResponse{
				Result: fmt.Sprint("client not register"),
			}
			message.Send(resp, conn)
			log.Warn("reject proxy request reason: unregister client %s", req.ClientId)
			conn.Close()
			return
		}
		resp := &message.ProxyResponse{
			ProxyName: req.ProxyName,
			Result:    "ok",
		}
		_, err := mgr.RegisterProxy(req, conn)
		if err != nil {
			resp.Result = err.Error()
			message.Send(resp, conn)
			conn.Close()
			log.Warn("client '%s' register proxy error:%s", mgr.ClientId, resp.Result)
			return
		}
		message.Send(resp, conn)
	}
}

func (self *Server) auth(msg *message.AuthRequest) error {
	log.Info("auth {hostname:%s os:%s}", msg.HostName, msg.Os)
	if msg.AuthToken != self.config.AuthToken {
		return errors.New("auth code error")
	}
	//log
	return nil
}

func (self *Server) registerManager(req *message.AuthRequest, conn net.Conn) *Manager {
	mgr := NewManager(self.genKey(), conn, req)

	self.mutex.Lock()
	defer self.mutex.Unlock()
	self.managers[mgr.ClientId] = mgr

	go func() {
		mgr.Run()
		mgr.Close()
		self.deregisterManager(mgr.ClientId)
	}()
	return mgr
}

func (self *Server) deregisterManager(clientId string) {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	delete(self.managers, clientId)
}

func (self *Server) genKey() string {
	buf := make([]byte, 16)
	for {
		rand.Read(buf)
		key := base64.RawURLEncoding.EncodeToString(buf)
		self.mutex.RLock()
		_, ok := self.managers[key]
		self.mutex.RUnlock()
		if !ok {
			return key
		}
	}
}
