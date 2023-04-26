package main

import "net"

type ProxyConfig struct {
	Name       string
	Type       string `ini:"type"`
	LocalIp    string `ini:"local_ip"`
	LocalPort  string `ini:"local_port"`
	RemotePort string `ini:"remote_port"`
}

type Agent struct {
	Config     *ProxyConfig
	Name       string
	Type       string
	LocalIp    string
	LocalPort  string
	RemotePort string
	Started    bool
}

func NewAgent(cfg *ProxyConfig) (*Agent, error) {
	agent := &Agent{
		Config:     cfg,
		Name:       cfg.Name,
		Type:       cfg.Type,
		LocalIp:    cfg.LocalIp,
		LocalPort:  cfg.LocalPort,
		RemotePort: cfg.RemotePort,
	}
	return agent, nil
}

func (self *Agent) Connect() (net.Conn, error) {
	addr := net.JoinHostPort(self.LocalIp, self.LocalPort)
	return net.Dial("tcp", addr)
}

func (self *Agent) Start() {
	self.Started = true
}
