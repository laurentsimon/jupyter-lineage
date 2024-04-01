package logger

import (
	"fmt"
	"os"
)

type Logger struct {
}

func (l Logger) Fatalf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "FATAL: "+format+"\n", a...)
	os.Exit(1)
}

func (l Logger) Errorf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "ERROR: "+format+"\n", a...)
}

func (l Logger) Infof(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "INFO: "+format+"\n", a...)
}

func (l Logger) Warnf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "WARN: "+format+"\n", a...)
}

func (l Logger) Debugf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "DEBUG: "+format+"\n", a...)
}
