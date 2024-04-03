package http

import "github.com/laurentsimon/jupyter-lineage/pkg/logger"

type httpLogger struct {
	logger logger.Logger
}

// TODO: Fix. There's a Logf function goproxyCtx has.
func (l httpLogger) Printf(format string, v ...interface{}) {
	l.logger.Infof("[goproxy]"+format, v...)
}
