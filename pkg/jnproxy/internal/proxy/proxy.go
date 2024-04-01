package proxy

type Proxy interface {
	Start() error
	Stop() error
}
