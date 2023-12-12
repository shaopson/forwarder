package client

import (
	"errors"
	"github.com/shaopson/forwarder/client/proxy"
	"github.com/shaopson/forwarder/message"
	"github.com/shaopson/forwarder/pipe/crypto"
	log "github.com/shaopson/grlog"
	"net"
	"os"
	"time"
)

const MessageReadTimeout = time.Second * 5

type Config struct {
	ServerIp   string `ini:"server_ip"`
	ServerPort string `ini:"server_port"`
	AuthToken  string `ini:"auth_token"`
	LogFile    string `ini:"log_file"`
	Proxies    map[string]*proxy.Config
}

type Client struct {
	config    *Config
	authToken string
	clientId  string
}

func NewClient(cfg *Config) *Client {
	client := &Client{
		config:    cfg,
		authToken: cfg.AuthToken,
	}
	return client
}

func (self *Client) Run() error {
	addr := net.JoinHostPort(self.config.ServerIp, self.config.ServerPort)
	tryCount := 0
	for {
		tryCount += 1
		if tryCount > 5 {
			return errors.New("exceeded maximum number of try, exit")
		}
		log.Info("connecting to server %s", addr)
		conn, err := net.DialTimeout("tcp", addr, time.Second*10)
		if err == nil {
			conn, err = crypto.NewConn(conn, self.authToken)
			if err != nil {
				log.Error("crypto connection error:%s", err)
				continue
			}
			err = self.auth(conn)
			if err == nil {
				manager := NewManager(self.clientId, conn, self.config)
				manager.Run()
				//manager exit
				manager.Close()
				tryCount = 0
				time.Sleep(time.Second)
				continue
			} else {
				conn.Close()
				log.Warn("auth failure:%s", err)
			}
		} else {
			log.Warn("connect server failure, %s", err)
		}
		log.Info("try again after 5 second [%d/5]", tryCount)
		time.Sleep(time.Second * 5)
	}
}

func (self *Client) auth(conn net.Conn) error {
	hostname, _ := os.Hostname()
	req := message.AuthRequest{
		AuthToken: self.authToken,
		HostName:  hostname,
		Os:        os.Getenv("GOOS"),
	}
	if err := message.Send(req, conn); err != nil {
		return err
	}

	conn.SetReadDeadline(time.Now().Add(MessageReadTimeout))
	msg, err := message.Read(conn)
	if err != nil {
		return err
	}
	conn.SetReadDeadline(time.Time{})

	if resp, ok := msg.(*message.AuthResponse); !ok {
		return errors.New("invalid auth response")
	} else if resp.Result != "ok" {
		return errors.New(resp.Result)
	} else {
		self.clientId = resp.ClientId
	}
	log.Info("auth success")
	return nil
}
