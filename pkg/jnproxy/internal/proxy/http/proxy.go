package http

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"sync"

	"github.com/laurentsimon/jupyter-lineage/pkg/slsa"

	"github.com/elazarl/goproxy"
	handler "github.com/laurentsimon/jupyter-lineage/pkg/jnproxy/handler/http"
	logimpl "github.com/laurentsimon/jupyter-lineage/pkg/jnproxy/internal/logger"
	"github.com/laurentsimon/jupyter-lineage/pkg/jnproxy/internal/proxy"
	"github.com/laurentsimon/jupyter-lineage/pkg/logger"
)

type Proxy struct {
	wg           sync.WaitGroup
	logger       logger.Logger
	server       *http.Server
	handlers     []handler.Handler
	callbacks    sync.Map
	dependencies []slsa.ResourceDescriptor
	mu           sync.Mutex // To add dependencies
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

	// Set optional parameters.
	for _, option := range options {
		err := option(proxy)
		if err != nil {
			return nil, err
		}
	}

	// Create the http proxy.
	if err := proxy.createHttpProxy(); err != nil {
		return nil, err
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
	/* TODO: handler will need:
	set of allow / deny, regex, etc list
	a callback (stateful?) to stream the data back (using session ID)
	need to know when request is over, and return a resource descriptor.
	*/
	// Set callbacks.
	httpProxy.OnRequest().DoFunc(func(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		if p.handlers == nil {
			p.logger.Debugf("[http] no handler installed (%q)", r.Host)
			return r, nil
		}
		for _, h := range p.handlers {
			req, resp, ok, err := h.OnRequest(r, handler.Context{ID: ctx.Session, Logger: p.logger})
			if err != nil {
				// TODO: More logging.
				p.logger.Errorf("[http] handler (%q) OnRequest (%q) error: %v", h.Name(), r.Host, err)
				continue
			}
			if resp != nil {
				p.logger.Debugf("[http] handler (%q) created a request (%q)", h.Name(), r.Host)
				return req, resp
			}
			if !ok {
				p.logger.Debugf("[http] handler (%q) not interested in host (%q)", h.Name(), r.Host)
				continue
			}
			// Keep track of the handler to call back.
			p.callbacks.Store(ctx.Session, h)
			return req, resp
		}
		return r, nil
	})
	httpProxy.OnResponse().DoFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
		if p.handlers == nil {
			return resp
		}
		defer p.callbacks.Delete(ctx.Session)
		// TODO(#5): Support chunked encoding.
		tf, ok := resp.Header["Transfer-Encoding"]
		if ok && slices.Contains(tf, "chunked") {
			return handler.NewResponse(ctx.Req, handler.ContentTypeText, http.StatusInternalServerError, "chunked not supported")
		}
		val, ok := p.callbacks.Load(ctx.Session)
		if !ok {
			// TODO: configurable what to do here.
			p.logger.Debugf("[http] host (%q) has not handler", ctx.Req.Host)
		}
		v, ok := val.(handler.Handler)
		if !ok {
			p.logger.Errorf("[http] map contains a non handler type (%T)", val)
			return goproxy.NewResponse(ctx.Req, goproxy.ContentTypeText, http.StatusInternalServerError, "InternalServerError")
		}
		p.logger.Debugf("[http] handler (%q) handling response (%q)", v.Name(), ctx.Req.Host)
		r, err := v.OnResponse(resp, handler.Context{ID: ctx.Session, Req: ctx.Req, Logger: p.logger})
		if err != nil {
			p.logger.Errorf("[http] handler (%q) OnResponse (%q) error: %v", v.Name(), ctx.Req.Host, err)
			return goproxy.NewResponse(ctx.Req, goproxy.ContentTypeText, http.StatusInternalServerError, "InternalServerError")
		}
		deps, err := v.Dependencies(handler.Context{Logger: p.logger})
		if err != nil {
			p.logger.Errorf("[http] handler (%q) Dependencies (%q) error: %v", v.Name(), ctx.Req.Host, err)
			return goproxy.NewResponse(ctx.Req, goproxy.ContentTypeText, http.StatusInternalServerError, "InternalServerError")
		}
		if err := p.recordDependencies(deps); err != nil {
			p.logger.Errorf("[http] handler (%q) record dependencies (%q) error: %v", v.Name(), ctx.Req.Host, err)
			return goproxy.NewResponse(ctx.Req, goproxy.ContentTypeText, http.StatusInternalServerError, "InternalServerError")
		}
		return r
	})
	httpProxy.OnRequest().HandleConnect(goproxy.AlwaysMitm)

	p.server.Handler = httpProxy
	return nil
}

func (p *Proxy) recordDependencies(deps []slsa.ResourceDescriptor) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.dependencies = append(p.dependencies, deps...)
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

func (p *Proxy) Type() proxy.Type {
	return proxy.TypeRuntime
}

func (p *Proxy) Dependencies() ([]slsa.ResourceDescriptor, error) {
	return p.dependencies, nil
}

func (p *Proxy) serve() {
	defer p.wg.Done()
	if err := p.server.ListenAndServe(); err != http.ErrServerClosed {
		p.logger.Fatalf("[http]: serve error: %v", err)
	}
	p.logger.Infof("[http]: serve exiting")
}
