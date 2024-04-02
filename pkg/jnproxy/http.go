package proxy

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
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

/*
	import os

os.environ['HTTP_PROXY'] = 'http://proxy_url:proxy_port'
os.environ['HTTPS_PROXY'] = 'http://proxy_url:proxy_port'
*/
func New(address string, logger logger.Logger) (*Proxy, error) {
	httpProxy := goproxy.NewProxyHttpServer()
	httpProxy.OnResponse(goproxy.ReqHostIs("www.google.com")).DoFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
		b, _ := ioutil.ReadAll(resp.Body)
		// TODO: handle error
		logger.Debugf("http: received (%q): %s", "www.google.com", string(b))
		resp.Body.Close()

		resp.Body = ioutil.NopCloser(bytes.NewBufferString(string(b)))
		return resp
	})
	proxy := &Proxy{
		server: &http.Server{
			Addr:    address,
			Handler: httpProxy,
		},
		logger: logger,
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
