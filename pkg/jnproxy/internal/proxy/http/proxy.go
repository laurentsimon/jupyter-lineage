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

// TODO: https://github.com/elazarl/goproxy/blob/master/examples/goproxy-eavesdropper/main.go#L27
// https://github.com/elazarl/goproxy/tree/master/examples/goproxy-transparent and https://github.com/elazarl/goproxy/blob/master/examples/goproxy-transparent/proxy.sh
// TODO: SUpport config for transparent vs non-transparent, possibly based on host / address.

// TODO: Take as variadic options including handlers.
// We'll have our own default ones that people can use
// like: DenyHostHandler, AllowHostHandler, HuggingfaceDatasetHandler, etc

// TOD: Fork the project?
// 1. Code can't run multiple instances of a proxy, because of the use of global vars https://github.com/elazarl/goproxy/blob/7cc037d33fb57d20c2fa7075adaf0e2d2862da78/https.go#L33-L37
// 2. No support for custom signers https://github.com/elazarl/goproxy/blob/7cc037d33fb57d20c2fa7075adaf0e2d2862da78/https.go#L476
// 3. No custom logger supported https://github.com/elazarl/goproxy/blob/7cc037d33fb57d20c2fa7075adaf0e2d2862da78/ctx.go#L61-L80.
// 4. Insecure default Ca verifications:
//		https://github.com/elazarl/goproxy/blob/master/certs.go#L20 used in https://github.com/elazarl/goproxy/blob/master/https.go#L467
//		https://github.com/elazarl/goproxy/blob/master/proxy.go#L219 https://github.com/elazarl/goproxy/blob/7cc037d33fb57d20c2fa7075adaf0e2d2862da78/https.go#L33-L37

func New(address string, logger logger.Logger) (*Proxy, error) {
	// Create the http proxy.
	httpProxy, err := createHttpProxy(logger)
	if err != nil {
		return nil, err
	}
	// Create self.
	proxy := &Proxy{
		server: &http.Server{
			Addr:    address,
			Handler: httpProxy,
		},
		logger: logger,
	}
	// Create the handler.
	handler := handler{
		logger:       logger,
		allowedHosts: []string{"www.google.com"},
	}
	// Set callbacks.
	httpProxy.OnRequest().DoFunc(func(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		return handler.onRequest(r, ctx)
	})
	httpProxy.OnResponse().DoFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
		return handler.onResponse(resp, ctx)
	})
	return proxy, nil
}

func createHttpProxy(logger logger.Logger) (*goproxy.ProxyHttpServer, error) {
	httpProxy := goproxy.NewProxyHttpServer()
	httpProxy.Logger = httpLogger{
		// Pass our logger to the proxy.
		// Unfortuantely, it does not support different types of logging :/
		logger: logger,
	}
	// https://pkg.go.dev/net/http#ProxyFromEnvironment
	// WARNING: By default, the proxy does not verify the destination certificate,
	// see https://github.com/elazarl/goproxy/blob/master/proxy.go#L219 so we
	// must overwrite the TLSClientConfig
	httpProxy.Tr = &http.Transport{Proxy: http.ProxyFromEnvironment}
	httpProxy.CertStore = newCertStorage()
	// TODO: Set a CA.
	if err := setCA([]byte{}, []byte{}); err != nil {
		return nil, err
	}
	httpProxy.Verbose = true
	return httpProxy, nil
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
		p.logger.Warnf("[http]: shutdown error: %v", err)
	}
	p.wg.Wait()
	return nil
}

func (p *Proxy) serve() {
	defer p.wg.Done()
	if err := p.server.ListenAndServe(); err != http.ErrServerClosed {
		p.logger.Fatalf("[http]: serve error: %v", err)
	}
	p.logger.Infof("[http]: serve exiting")
}
