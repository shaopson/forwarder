package main

import (
	"errors"
	"github.com/dev-shao/iprp/log"
	"github.com/dev-shao/iprp/message"
	"github.com/hashicorp/yamux"
	"io"
	"net"
	"os"
	"time"
)

type ClientConfig struct {
	ServerIp     string `ini:"server_ip"`
	ServerPort   string `ini:"server_port"`
	AuthCode     string `ini:"auth_code"`
	PipeCount    int    `ini:"pipe_count"`
	LogFile      string `ini:"log_file"`
	ProxyConfigs map[string]*ProxyConfig
}

type Client struct {
	Config     *ClientConfig
	ClientId   string
	serverAddr string
	conn       net.Conn
	session    *yamux.Session
}

func NewClient(cfg *ClientConfig) *Client {
	return &Client{
		Config:     cfg,
		serverAddr: net.JoinHostPort(cfg.ServerIp, cfg.ServerPort),
	}
}

func (self *Client) Run() error {
	tryCount := 0

	for {
		tryCount += 1
		if tryCount > 5 {
			return errors.New("The number of login failures over the limit")
		}
		conn, err := self.connectServer()
		if err == nil {
			err = self.Login(conn)
			if err == nil {
				manager := NewManager(conn, self, self.Config.ProxyConfigs)
				manager.Run()
				tryCount = 0
				log.Info("%s over", manager)
				time.Sleep(time.Second)
				continue
			} else {
				log.Warn("login fail:%s", err)
			}
		} else {
			log.Warn("connect server fail:%s", err)
		}
		log.Info("try again after 6 second [%d/5]", tryCount)
		time.Sleep(time.Second * 6)
	}
}

func (self *Client) connectServer() (net.Conn, error) {
	conn, err := net.Dial("tcp", self.serverAddr)
	if err != nil {
		return nil, err
	}
	self.conn = conn
	//todo:if not use yamux
	//return conn, nil

	cfg := yamux.DefaultConfig()
	cfg.LogOutput = io.Discard
	session, err := yamux.Client(conn, cfg)
	if err != nil {
		return nil, err
	}
	self.session = session
	stream, err := session.OpenStream()
	if err != nil {
		return nil, err
	}
	return stream, nil
}

func (self *Client) NewPipeConn() (net.Conn, error) {
	//todo: if not use yamux
	log.Debug("create new pipe connect")
	stream, err := self.session.OpenStream()
	if err != nil {
		return nil, err
	}
	msg := &message.PipeMessage{
		ClientId: self.ClientId,
	}
	err = message.Send(msg, stream)
	if err != nil {
		log.Warn("Send pipe message fail:%s", err)
		stream.Close()
		return nil, err
	}
	return stream, nil
}

func (self *Client) Login(conn net.Conn) (err error) {
	defer func() {
		if err != nil {
			conn.Close()
		}
	}()

	req := &message.LoginRequest{
		Version:   1,
		AuthCode:  self.Config.AuthCode,
		PipeCount: self.Config.PipeCount,
		HostName:  "",
		Os:        os.Getenv("GOOS"),
	}
	err = message.Send(req, conn)
	if err != nil {
		return
	}
	conn.SetReadDeadline(time.Now().Add(time.Second * 5))
	msg, err := message.Get(conn)
	if err != nil {
		log.Warn("Get login result error:%s", err)
		return
	}
	conn.SetReadDeadline(time.Time{})
	if resp, ok := msg.(*message.LoginResponse); !ok {
		return errors.New("invalid login response")
	} else if resp.Result != "ok" {
		return errors.New(resp.Result)
	} else {
		self.ClientId = resp.ClientId
	}
	log.Info("login success")
	return nil
}
