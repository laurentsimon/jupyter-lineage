package proxy

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/elazarl/goproxy"
	"github.com/laurentsimon/jupyter-lineage/pkg/logger"
)

type Proxy struct {
	wg     sync.WaitGroup
	logger logger.Logger
	server *http.Server
}

func New(address string, logger logger.Logger) (*Proxy, error) {
	proxy := &Proxy{
		server: &http.Server{
			Addr:    address,
			Handler: goproxy.NewProxyHttpServer(),
		},
	}
	return proxy, nil
}

func (p *Proxy) Start() error {
	if p.server == nil {
		return fmt.Errorf("http:proxy not ready")
	}
	p.wg.Add(1)
	go p.serve()
	return nil
}

func (p *Proxy) Stop() error {
	if p.server == nil {
		return fmt.Errorf("http:proxy not ready")
	}
	if err := p.server.Shutdown(context.Background()); err != nil {
		p.logger.Warnf("http:shutdown error: %v", err)
	}
	p.wg.Wait()
	return nil
}

func (p *Proxy) serve() {
	defer p.wg.Done()
	if err := p.server.ListenAndServe(); err != http.ErrServerClosed {
		p.logger.Fatalf("http:serve error: %v", err)
	}
	p.logger.Infof("http:serve exiting")
}
