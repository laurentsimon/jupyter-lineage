package http

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/elazarl/goproxy"
	"github.com/laurentsimon/jupyter-lineage/pkg/logger"
)

type Proxy struct {
	wg      sync.WaitGroup
	logger  logger.Logger
	server  *http.Server
	handler handler
}

/*
	import os

os.environ['HTTP_PROXY'] = 'http://proxy_url:proxy_port'
os.environ['HTTPS_PROXY'] = 'http://proxy_url:proxy_port'
*/

// TODO: Take as variadic options including handlers.
// We'll have our own default ones that people can use
// like: DenyHostHandler, AllowHostHandler, HuggingfaceDatasetHandler, etc
func New(address string, logger logger.Logger) (*Proxy, error) {
	httpProxy := goproxy.NewProxyHttpServer()
	proxy := &Proxy{
		server: &http.Server{
			Addr:    address,
			Handler: httpProxy,
		},
		logger: logger,
	}
	handler := handler{
		logger:       logger,
		allowedHosts: []string{"www.google.com"},
	}
	httpProxy.OnRequest().DoFunc(func(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		return handler.onRequest(r, ctx)
	})
	httpProxy.OnResponse().DoFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
		return handler.onResponse(resp, ctx)
	})
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
