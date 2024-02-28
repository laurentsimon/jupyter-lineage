package session

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
)

// TODO: move this file to internal

type addressBinding struct {
	name string
	src  string
	dst  string
}

type proxy struct {
	bindings []addressBinding
	// NOTE: see https://shantanubansal.medium.com/how-to-terminate-goroutines-in-go-effective-methods-and-examples-f796dcede512
	// on ways to terminate a g routine.
	context   context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	listeners []net.Listener
	logger    Logger
	// TODO: forwarders
}

type setNoDelayer interface {
	SetNoDelay(bool) error
}

type closer interface {
	Close() error
}

func proxyNew(bindings []addressBinding, logger Logger) (*proxy, error) {
	ctx, cancel := context.WithCancel(context.Background())
	return &proxy{
		logger:    logger,
		bindings:  bindings, // TODO: Make a copy.
		context:   ctx,
		cancel:    cancel,
		listeners: make([]net.Listener, len(bindings)),
	}, nil
}

// See https://okanexe.medium.com/the-complete-guide-to-tcp-ip-connections-in-golang-1216dae27b5a
// https://coderwall.com/p/wohavg/creating-a-simple-tcp-server-in-go
func (p *proxy) Start() error {
	var e error
	// Start all the listeners and forwarders.
	for i, _ := range p.bindings {
		binding := &p.bindings[i]
		p.wg.Add(1)
		err := make(chan error, 1)
		go func() {
			defer p.wg.Done()
			listenOnPort(p.context, *binding, &p.listeners[i], p.logger, err)
		}()
		e = <-err
		// If there was an error starting, finish immediatly.
		if e != nil {
			p.logger.Errorf("start %q binding: %v", binding.name, e)
			p.Finish()
			break
		}
		p.logger.Infof("binding %q started successfully", binding.name)
	}
	return e
}

func (p *proxy) Finish() error {
	// Cancel all routines.
	p.cancel()

	// THis is needed if the go routines
	// are waiting on a listening socket.
	// TODO: race condition need to be handled because we currently
	// create the listening connection in a go routine. We must either:
	// - create it in main thread
	// - use a mutex
	// May not be required to do yet.
	// Warning: race when closing because start failed
	for i, _ := range p.bindings {
		binding := &p.bindings[i]
		listener := &p.listeners[i]
		close(*listener, binding.name, p.logger)
		// TODO: close forwarders.
	}

	// Wait for routines to exit.
	p.wg.Wait()
	return nil
}

// TODO: channel for error
func listenOnPort(ctx context.Context, binding addressBinding, listener *net.Listener, logger Logger, errRet chan error) {
	// Start listening.
	listenConn, err := net.Listen("tcp", binding.src)
	if err != nil {
		errRet <- fmt.Errorf("listen: %w", err)
		return
	}
	// Communicate to caller thhat we started successfully.
	errRet <- nil
	// Auto close on function exit.
	defer listenConn.Close()
	(*listener) = listenConn

	// TODO: no delay https://github.com/jpillora/go-tcp-proxy/blob/master/proxy.go
	var wg sync.WaitGroup
	var done bool
	var clientConns []net.Conn
L:
	for !done {
		select {
		case <-ctx.Done():
			logger.Infof("exiting listener for %q", binding.name)
			done = false
			break L
		default:
			// Accept the connection.
			clientConn, err := listenConn.Accept()
			if err != nil {
				// TODO: need to check the kind of error
				// and log info when closing properly if caller set to close.
				// TODO: use log iterface from caller
				//log.Fatal(err)
				logger.Errorf("accept %q: %v", binding.name, err)
				continue
			}
			if conn, ok := clientConn.(setNoDelayer); ok {
				// https://pkg.go.dev/net#TCPConn.SetNoDelay
				logger.Debugf("enable Nagle's algo on %q", clientConn.RemoteAddr().String())
				conn.SetNoDelay(true)
			}
			logger.Infof("connection from %q", clientConn.RemoteAddr().String())
			// Keep track of the connections.
			clientConns = append(clientConns, clientConn)
			// Handle the connection.
			wg.Add(1)
			go func() {
				defer wg.Done()
				handleClient(logger, clientConn, binding.name)
			}()
		}
	}
	closeClientConns(logger, clientConns, binding.name)
	wg.Wait()

}

func close(closer closer, name string, logger Logger) {
	if closer == nil {
		return
	}
	if err := closer.Close(); err != nil && !isClosedConnError(err) {
		logger.Errorf("close for %T %q: %v", closer, name, err)
	}
}

func closeClientConns(logger Logger, clientConns []net.Conn, name string) {
	for i, _ := range clientConns {
		logger.Debugf("closing client connection for %q...", name)
		conn := &clientConns[i]
		close(*conn, name, logger)
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

// TODO: keep track of errors
func handleClient(logger Logger, clientConn net.Conn, name string) {
	defer clientConn.Close()
	// Create a buffer to read data into
	buffer := make([]byte, 2048)

	for {
		// Read data from the client.
		n, err := clientConn.Read(buffer)
		if err != nil {
			logger.Errorf("read error on %q: %v", name, err)
			return
		}

		// Process and use the data (here, we'll just print it)
		logger.Debugf("read %q: %q", name, buffer[:n])
	}
}
