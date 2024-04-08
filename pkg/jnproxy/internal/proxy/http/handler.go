package http

import handler "github.com/laurentsimon/jupyter-lineage/pkg/jnproxy/handler/http"

func WithHandlers(handlers []handler.Handler) Option {
	return func(p *Proxy) error {
		return p.setHandlers(handlers)
	}
}

func (p *Proxy) setHandlers(handlers []handler.Handler) error {
	p.handlers = handlers
	return nil
}
