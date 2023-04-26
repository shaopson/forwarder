package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/dev-shao/iprp/log"
	"github.com/dev-shao/iprp/message"
	"github.com/hashicorp/yamux"
	"io"
	"math/rand"
	"net"
	"sync"
	"time"
)

type ServerConfig struct {
	Ip       string `ini:"bind_ip"`
	Port     string `ini:"bind_port"`
	AuthCode string `ini:"auth_code"`
	LogFile  string `ini:"log_file"`
}

type Server struct {
	config   *ServerConfig
	address  string
	listener net.Listener
	managers map[string]*Manager
	mutex    sync.RWMutex
	mgrDone  chan *Manager
}

func NewServer(config *ServerConfig) *Server {
	srv := &Server{
		config:   config,
		address:  net.JoinHostPort(config.Ip, config.Port),
		managers: make(map[string]*Manager),
		mgrDone:  make(chan *Manager, 1),
	}
	return srv
}

func (self *Server) Run() error {
	listener, err := net.Listen("tcp", self.address)
	if err != nil {
		return err
	}
	self.listener = listener
	defer listener.Close()

	log.Info("server listen on %s", self.address)
	go self.awaitManagerDone()

	for {
		conn, err := self.listener.Accept()
		if err != nil {
			log.Warn("listener close:%s", err)
			return err
		}
		go self.Handle(conn)
	}
}

func (self *Server) Handle(conn net.Conn) {
	//todo: if not use yamux

	cfg := yamux.DefaultConfig()
	cfg.LogOutput = io.Discard
	session, err := yamux.Server(conn, cfg)
	if err != nil {
		log.Warn("Create yamux session fail:%s", err)
		return
	}
	for {
		stream, err := session.AcceptStream()
		if err != nil {
			log.Debug("yamux session close", err)
			return
		}
		log.Debug("Accept new stream:%d", stream.StreamID())
		go self.HandleStream(stream)
	}
}

func (self *Server) HandleStream(stream *yamux.Stream) {
	_ = stream.SetReadDeadline(time.Now().Add(time.Second * 5))
	msg, err := message.Get(stream)
	if err != nil {
		log.Error("Get message from new stream fail:%s", err)
		stream.Close()
		return
	}
	_ = stream.SetReadDeadline(time.Time{})

	switch v := msg.(type) {
	case *message.LoginRequest:
		err := self.Login(v)
		if err != nil {
			resp := &message.LoginResponse{
				Result: fmt.Sprintf("login fail:%s", err),
			}
			message.Send(resp, stream)
			stream.Close()
			return
		}
		log.Info("New client login; address:%s version:%s auth code:%s Os:%s", stream.RemoteAddr(), v.Version, v.AuthCode, v.Os)
		mgr := self.RegisterManager(stream, v)
		resp := &message.LoginResponse{
			ClientId: mgr.ClientId,
			Result:   "ok",
		}
		message.Send(resp, stream)
	case *message.PipeMessage:
		self.mutex.RLock()
		mgr, ok := self.managers[v.ClientId]
		self.mutex.RUnlock()
		if !ok {
			stream.Close()
		} else if err = mgr.PushPipeConn(stream); err != nil {
			stream.Close()
		}
	default:
		log.Warn("Unknown message from new stream:%d", stream.StreamID())
		stream.Close()
	}
}

func (self *Server) Login(req *message.LoginRequest) error {
	if self.config.AuthCode != req.AuthCode {
		return errors.New("auth code error")
	}
	//todo: compare version

	return nil
}

func (self *Server) RegisterManager(conn net.Conn, req *message.LoginRequest) *Manager {
	manager := NewManager(self.genId(), conn, self.mgrDone, req)

	go manager.Run()

	self.mutex.Lock()
	self.managers[manager.ClientId] = manager
	self.mutex.Unlock()

	return manager
}

func (self *Server) genId() (key string) {
	b := make([]byte, 16)
	for {
		rand.Read(b)
		key = base64.RawURLEncoding.EncodeToString(b)[:16]
		self.mutex.RLock()
		_, ok := self.managers[key]
		self.mutex.RUnlock()
		if !ok {
			return
		}
	}
}

func (self *Server) awaitManagerDone() {
	for {
		mgr := <-self.mgrDone
		self.mutex.Lock()
		delete(self.managers, mgr.ClientId)
		self.mutex.Unlock()
		mgr.Close()
		log.Info("%s is done", mgr)
	}
}
