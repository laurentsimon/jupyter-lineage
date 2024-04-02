package logger

import (
	"fmt"
	"os"
	"time"
)

type Logger struct {
}

func (l Logger) Fatalf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "FATAL/"+time.Now().UTC().Format(time.RFC3339)+": "+format+"\n", a...)
	os.Exit(1)
}

func (l Logger) Errorf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "ERROR/"+time.Now().UTC().Format(time.RFC3339)+": "+format+"\n", a...)
}

func (l Logger) Infof(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "INFO/"+time.Now().UTC().Format(time.RFC3339)+": "+format+"\n", a...)
}

func (l Logger) Warnf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "WARN/"+time.Now().UTC().Format(time.RFC3339)+": "+format+"\n", a...)
}

func (l Logger) Debugf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "DEBUG/"+time.Now().UTC().Format(time.RFC3339)+": "+format+"\n", a...)
}
