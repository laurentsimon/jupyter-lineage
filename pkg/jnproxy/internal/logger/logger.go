package logger

import (
	"fmt"
	"os"
	"time"
)

type Logger struct {
}

func (l Logger) Fatalf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "FATAL/%s: "+format+"\n", []any{time.Now().UTC().Format(time.RFC3339), a})
	os.Exit(1)
}

func (l Logger) Errorf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "ERROR/%s: "+format+"\n", []any{time.Now().UTC().Format(time.RFC3339), a})
}

func (l Logger) Infof(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "INFO/%s: "+format+"\n", []any{time.Now().UTC().Format(time.RFC3339), a})
}

func (l Logger) Warnf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "WARN/%s: "+format+"\n", []any{time.Now().UTC().Format(time.RFC3339), a})
}

func (l Logger) Debugf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "DEBUG/%s: "+format+"\n", []any{time.Now().UTC().Format(time.RFC3339), a})
}
