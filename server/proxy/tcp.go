package proxy

import (
	"errors"
	"fmt"
	"github.com/hashicorp/yamux"
	"github.com/shaopson/forwarder/pipe"
	log "github.com/shaopson/grlog"
	"io"
	"net"
)

type TCPProxy struct {
	name       string
	localAddr  string
	remoteAddr string
	conn       net.Conn
	session    *yamux.Session
	listener   net.Listener
	pipePool   chan net.Conn
	pipeCount  int
}

func NewTCPProxy(name string, laddr, raddr string, conn net.Conn) (*TCPProxy, error) {
	listener, err := net.Listen("tcp", laddr)
	if err != nil {
		log.Error("start tcp proxy '%s' error:%s", name, err)
		return nil, err
	}
	proxy := &TCPProxy{
		name:       name,
		localAddr:  laddr,
		remoteAddr: raddr,
		conn:       conn,
		listener:   listener,
		//pipePool:    make(chan net.Conn, 8),
	}
	return proxy, nil
}

func (self *TCPProxy) Run() {
	cfg := yamux.DefaultConfig()
	cfg.LogOutput = io.Discard
	session, err := yamux.Client(self.conn, cfg)
	if err != nil {
		log.Error("%s init yamux client connect error:%s", self, err)
		return
	}
	self.session = session
	log.Info("%s listen on %s", self, self.localAddr)

	//go self.awaitPushPipe()

	for {
		conn, err := self.listener.Accept()
		if err != nil {
			log.Error("%s listener close:%s", self, err)
			return
		}
		log.Info("%s accept a new connection from %s", self, conn.RemoteAddr())
		go self.handleConnect(conn)
	}
}

func (self *TCPProxy) handleConnect(conn net.Conn) {
	//get pipe
	//dest, err := self.getPipe()
	stream, err := self.session.OpenStream()
	if err != nil {
		log.Error("open stream error:%s", err)
		conn.Close()
		return
	}

	//todo：encrypt pipe

	pipe.Join(conn, stream)

}

func (self *TCPProxy) getPipe() (net.Conn, error) {
	select {
	case stream, ok := <-self.pipePool:
		if !ok {
			return nil, errors.New("pipe pool closed")
		}
		return stream, nil
	default:
		return self.session.OpenStream()
	}
}

func (self *TCPProxy) awaitPushPipe() {
	defer close(self.pipePool)
	for {
		stream, err := self.session.OpenStream()
		if err != nil {
			log.Error("%s create new pipe failure:%s", self, err)
			return
		}
		self.pipePool <- stream
	}
}

func (self *TCPProxy) Close() {
	self.listener.Close()
	self.session.Close()
	self.conn.Close()
	log.Info("%s close", self)
}

func (self *TCPProxy) Name() string {
	return self.name
}

func (self *TCPProxy) LocalAddr() string {
	return self.localAddr
}

func (self *TCPProxy) RemoteAddr() string {
	return self.remoteAddr
}

func (self *TCPProxy) String() string {
	return fmt.Sprintf("tcp proxy '%s'", self.name)
}
