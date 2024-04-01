package proxy

import "context"

type Proxy struct {
	ctx    context.Context
	cancel context.CancelFunc
}

func New() (*Proxy, error) {
	ctx, cancel := context.WithCancel(context.Background())
	proxy := &Proxy{
		ctx:    ctx,
		cancel: cancel,
	}
	return proxy, nil
}

func (p *Proxy) Start() error {
	return nil
}

func (p *Proxy) Stop() error {
	return nil
}
