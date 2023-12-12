package proxy

type Proxy interface {
	Run()
	Close()
	Name() string
	LocalAddr() string
	RemoteAddr() string
}
