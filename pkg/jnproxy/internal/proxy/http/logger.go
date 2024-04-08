package http

import "github.com/laurentsimon/jupyter-lineage/pkg/logger"

type httpLogger struct {
	logger logger.Logger
}

// TODO: Fix. There's a Logf function goproxyCtx has.
func (l httpLogger) Printf(format string, v ...interface{}) {
	l.logger.Infof("[goproxy]"+format, v...)
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
