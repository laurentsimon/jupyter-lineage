package logger

import (
	"fmt"
	"io"
	"os"
)

type Option func(*Logger) error

type Logger struct {
	writer io.Writer
}

func New(opts ...Option) (*Logger, error) {
	logger := new(Logger)

	for _, option := range opts {
		err := option(logger)
		if err != nil {
			return nil, err
		}
	}
	if logger.writer == nil {
		logger.writer = os.Stderr
	}
	return logger, nil
}

func WithWriter(w io.Writer) Option {
	return func(l *Logger) error {
		return l.setWriter(w)
	}
}

func (l *Logger) setWriter(w io.Writer) error {
	l.writer = w
	return nil
}

func (l Logger) Fatalf(format string, a ...any) {
	l.writer.Write([]byte(fmt.Sprintf("FATAL: "+format+"\n", a...)))
	os.Exit(1)
}

func (l Logger) Errorf(format string, a ...any) {
	l.writer.Write([]byte(fmt.Sprintf("ERROR: "+format+"\n", a...)))
}

func (l Logger) Infof(format string, a ...any) {
	l.writer.Write([]byte(fmt.Sprintf("INFO: "+format+"\n", a...)))
}

func (l Logger) Warnf(format string, a ...any) {
	l.writer.Write([]byte(fmt.Sprintf("WARN: "+format+"\n", a...)))
}

func (l Logger) Debugf(format string, a ...any) {
	l.writer.Write([]byte(fmt.Sprintf("DEBUG: "+format+"\n", a...)))
}
