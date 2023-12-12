package proxy

type Config struct {
	Name       string
	ProxyType  string `ini:"type"`
	RemoteIp   string `ini:"remote_ip"`
	RemotePort string `ini:"remote_port"`
	LocalIp    string `ini:"local_ip"`
	LocalPort  string `ini:"local_port"`
}

type Proxy interface {
	Run()
	Close()
	LocalAddr() string
	RemoteAddr() string
	Name() string
}
