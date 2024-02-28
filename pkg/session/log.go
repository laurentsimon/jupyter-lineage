package session

import (
	"fmt"
	"os"
)

type log struct {
}

func (l log) Fatalf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "FATAL:"+format+"\n", a...)
	os.Exit(1)
}

func (l log) Errorf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "ERROR:"+format+"\n", a...)
}

func (l log) Infof(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "INFO:"+format+"\n", a...)
}

func (l log) Warnf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "WARN:"+format+"\n", a...)
}

func (l log) Debugf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "DEBUG:"+format+"\n", a...)
}
