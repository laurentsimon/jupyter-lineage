package proxy

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/laurentsimon/jupyter-lineage/pkg/logger"
	"github.com/laurentsimon/jupyter-lineage/pkg/repository"
	"github.com/laurentsimon/jupyter-lineage/pkg/session/internal/slsa"
)

// TODO: move this file to internal

type AddressBinding struct {
	Name string
	Src  string
	Dst  string
}

type Proxy struct {
	bindings []AddressBinding
	// NOTE: see https://shantanubansal.medium.com/how-to-terminate-goroutines-in-go-effective-methods-and-examples-f796dcede512
	// on ways to terminate a g routine.
	context   context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	listeners []net.Listener
	//forwarders []net.Conn
	logger     logger.Logger
	repoClient repository.Client
}

type setNoDelayer interface {
	SetNoDelay(bool) error
}

type closer interface {
	Close() error
}

func New(bindings []AddressBinding, logger logger.Logger, repoClient repository.Client) (*Proxy, error) {
	ctx, cancel := context.WithCancel(context.Background())
	return &Proxy{
		logger:     logger,
		repoClient: repoClient,
		bindings:   bindings, // TODO: Make a copy.
		context:    ctx,
		cancel:     cancel,
		listeners:  make([]net.Listener, len(bindings)),
	}, nil
}

// See https://okanexe.medium.com/the-complete-guide-to-tcp-ip-connections-in-golang-1216dae27b5a
// https://coderwall.com/p/wohavg/creating-a-simple-tcp-server-in-go
func (p *Proxy) Start() error {
	var e error
	// Start all the listeners and forwarders.
	for i, _ := range p.bindings {
		binding := &p.bindings[i]
		p.wg.Add(1)
		err := make(chan error, 1)
		go func() {
			defer p.wg.Done()
			listen(p.context, *binding, &p.listeners[i], p.logger, p.repoClient, err)
		}()
		e = <-err
		// If there was an error starting, finish immediatly.
		if e != nil {
			p.logger.Errorf("start %q binding: %v", binding.Name, e)
			p.Finish()
			break
		}
		p.logger.Infof("binding %q started successfully", binding.Name)
	}
	return e
}

func (p *Proxy) Finish() error {
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
		close(*listener, binding.Name, p.logger)
		// TODO: close forwarders.
	}

	// Wait for routines to exit.
	p.wg.Wait()
	return nil
}

// TODO: channel for error
func listen(ctx context.Context, binding AddressBinding, listener *net.Listener, logger logger.Logger,
	repoClient repository.Client, errRet chan error) {
	// Start listening.
	listenConn, err := net.Listen("tcp", binding.Src)
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
			logger.Infof("exiting listener for %q", binding.Name)
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
				logger.Errorf("accept %q: %v", binding.Name, err)
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
				handleClient(logger, repoClient, clientConn, binding.Name)
			}()
		}
	}
	closeClientConns(logger, clientConns, binding.Name)
	wg.Wait()

}

func close(closer closer, name string, logger logger.Logger) {
	if closer == nil {
		return
	}
	if err := closer.Close(); err != nil && !isClosedConnError(err) {
		logger.Errorf("close for %T %q: %v", closer, name, err)
	}
}

func closeClientConns(logger logger.Logger, clientConns []net.Conn, name string) {
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
func handleClient(logger logger.Logger, repoClient repository.Client, clientConn net.Conn, name string) {
	defer clientConn.Close()
	// Create a buffer to read data into
	buffer := make([]byte, 2048)
	counter := uint64(0)
	for {
		// Read data from the client.
		n, err := clientConn.Read(buffer)
		if err != nil {
			// NOTE: closed connection will get an error and return.
			logger.Errorf("read error on %q: %v", name, err)
			return
		}

		// Process and use the data (here, we'll just print it)
		logger.Debugf("read %q: %q", name, buffer[:n])
		fn := fmt.Sprintf("%v/%016x", name, counter)
		c, err := slsa.Format(buffer[:n])
		if err != nil {
			logger.Fatalf("slsa format %q: []%v: %v", fn, buffer[:n], err)
		}
		if err := repoClient.CreateFile(fn, c); err != nil {
			// TODO: handle gracefully? Need to return and set an err
			// for the caller to check.
			logger.Fatalf("create file %q: %v", fn, err)
		}
		counter += 1
	}
}
