package http

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/elazarl/goproxy"
	logimpl "github.com/laurentsimon/jupyter-lineage/pkg/jnproxy/internal/logger"
	"github.com/laurentsimon/jupyter-lineage/pkg/logger"
)

type Proxy struct {
	wg     sync.WaitGroup
	logger logger.Logger
	server *http.Server
}

type Option func(*Proxy) error

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
//		https://github.com/elazarl/goproxy/blob/master/certs.go#L20 used in https://github.com/elazarl/goproxy/blob/master/https.go#L467. FIXED.
//		https://github.com/elazarl/goproxy/blob/master/proxy.go#L219 https://github.com/elazarl/goproxy/blob/7cc037d33fb57d20c2fa7075adaf0e2d2862da78/https.go#L33-L37
//		https://github.com/elazarl/goproxy/blob/master/https.go#L204
// 5. Only support P256 https://github.com/elazarl/goproxy/blob/master/signer.go#L87
// 6. Own PRNG https://github.com/elazarl/goproxy/blob/master/counterecryptor.go#L20. FIXED.

func New(address string, options ...Option) (*Proxy, error) {
	// Create self.
	proxy := &Proxy{
		server: &http.Server{
			Addr: address,
		},
		logger: logimpl.Logger{},
	}

	// Create the http proxy.
	if err := proxy.createHttpProxy(); err != nil {
		return nil, err
	}

	// Set optional parameters.
	for _, option := range options {
		err := option(proxy)
		if err != nil {
			return nil, err
		}
	}

	return proxy, nil
}

func (p *Proxy) createHttpProxy() error {
	httpProxy := goproxy.NewProxyHttpServer()
	httpProxy.Logger = httpLogger{
		// Pass our logger to the proxy.
		// Unfortuantely, it does not support different types of logging :/
		logger: p.logger,
	}
	// https://pkg.go.dev/net/http#ProxyFromEnvironment
	// WARNING: By default, the proxy does not verify the destination certificate,
	// see https://github.com/elazarl/goproxy/blob/master/proxy.go#L219 so we
	// must overwrite the TLSClientConfig.
	httpProxy.Tr = &http.Transport{Proxy: http.ProxyFromEnvironment}
	httpProxy.Verbose = true

	// Set the custom handler.
	handler := handler{
		logger:     p.logger,
		allowHosts: []string{"www.google.com"},
	}
	// Set callbacks.
	httpProxy.OnRequest().DoFunc(func(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		return handler.onRequest(r, ctx)
	})
	httpProxy.OnResponse().DoFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
		return handler.onResponse(resp, ctx)
	})
	httpProxy.OnRequest().HandleConnect(goproxy.AlwaysMitm)

	p.server.Handler = httpProxy
	return nil
}

func WithLogger(l logger.Logger) Option {
	return func(p *Proxy) error {
		return p.setLogger(l)
	}
}

func (p *Proxy) setLogger(l logger.Logger) error {
	p.logger = l
	return nil
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
