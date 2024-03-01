package proxy

import (
	"fmt"
	"net"
	"strings"

	"github.com/laurentsimon/jupyter-lineage/pkg/logger"
)

type closer interface {
	Close() error
}

type setNoDelayer interface {
	SetNoDelay(bool) error
}

// TODO: create class for connection.

func setConnSettings(logger logger.Logger, conn net.Conn) error {
	if c, ok := conn.(setNoDelayer); ok {
		// https://pkg.go.dev/net#TCPConn.SetNoDelay
		logger.Debugf("enable Nagle's algo on %q", conn.RemoteAddr().String())
		c.SetNoDelay(true)
	}
	if err := conn.(*net.TCPConn).SetKeepAlive(true); err != nil {
		return fmt.Errorf("keep alive: %w", err)
	}
	return nil
}

func cclose(closer closer, name string, logger logger.Logger) {
	if closer == nil {
		return
	}
	if err := closer.Close(); err != nil && !isClosedConnError(err) {
		logger.Errorf("close for %T %q: %v", closer, name, err)
	}
}

func isClosedConnError(err error) bool {
	if err == nil {
		return false
	}
	// see https://github.com/golang/go/blob/ccbc725f2d678255df1bd326fa511a492aa3a0aa/src/internal/poll/fd.go#L20-L24.
	if strings.Contains(err.Error(), "use of closed network connection") {
		return true
	}
	return false
}

func closeConns(logger logger.Logger, conns []net.Conn, name string) {
	for i, _ := range conns {
		logger.Debugf("closing connection for %q...", name)
		conn := &conns[i]
		cclose(*conn, name, logger)
	}
}

func connWrite(conn net.Conn, data []byte) error {
	_, err := conn.Write(data)
	// if n != len(data) {
	// 	return fmt.Errorf("conn write: %d bytes expected, %d bytes written", len(data), n)
	// }
	if err != nil {
		return fmt.Errorf("conn write: %w", err)
	}
	return nil
}
