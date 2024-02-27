package session

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
)

type proxy struct {
	srcMetadata NetworkMetadata
	dstMetadata NetworkMetadata
	// NOTE: see https://shantanubansal.medium.com/how-to-terminate-goroutines-in-go-effective-methods-and-examples-f796dcede512
	// on ways to terminate a g routine.
	context          context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
	listenerMetadata listenerMetadata
}

type listenerMetadata struct {
	shell     net.Listener
	stdin     net.Listener
	iopub     net.Listener
	control   net.Listener
	heartbeat net.Listener
}

func proxyNew(srcMetadata, dstMetadata NetworkMetadata) (*proxy, error) {
	ctx, cancel := context.WithCancel(context.Background())
	return &proxy{
		srcMetadata: srcMetadata,
		dstMetadata: dstMetadata,
		context:     ctx,
		cancel:      cancel,
	}, nil
}

// See https://okanexe.medium.com/the-complete-guide-to-tcp-ip-connections-in-golang-1216dae27b5a
// https://coderwall.com/p/wohavg/creating-a-simple-tcp-server-in-go
func (p *proxy) Start() error {
	// TODO: use a loop for all ports
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		listenOnPort(p.context, p.srcMetadata.IP, p.srcMetadata.Ports.Shell, "shell", &p.listenerMetadata.shell)
	}()
	// TODO: start other listening routines.
	return nil
}

func (p *proxy) Finish() error {
	// Cancel all routines.
	p.cancel()

	// Close all listening connections. Tis is needed if the go routines
	// are waiting on a listening socket.
	// TODO: race condition need to be handled because we currently
	// create the listening connection in a go routine. We must either:
	// - create it in main thread
	// - use a mutex
	if p.listenerMetadata.shell != nil {
		if err := p.listenerMetadata.shell.Close(); err != nil && !isClosedConnError(err) {
			fmt.Printf("close shell error: %v\n", err)
		}
	}
	// TODO: close all outgoing connections.

	// Wait for routines to exit.
	p.wg.Wait()
	return nil
}

// TODO: channel for error
func listenOnPort(ctx context.Context, ip string, port uint, name string, listener *net.Listener) {
	// Start listening.
	listenConn, err := net.Listen("tcp", address(ip, port))
	if err != nil {
		log.Fatal(fmt.Errorf("listen: %w", err))
		// TODO: communicate error bac to caller
	}
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
			fmt.Printf("Exiting listener for %q\n", name)
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
				fmt.Printf("accept %q: %v\n", name, err)
				continue
			}
			fmt.Printf("Connection from %q\n", clientConn.RemoteAddr().String())
			// Keep track of the connections.
			clientConns = append(clientConns, clientConn)
			// Handle the connection.
			wg.Add(1)
			go func() {
				defer wg.Done()
				handleClient(clientConn, name)
			}()
		}
	}
	closeClientConns(clientConns, name)
	wg.Wait()
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

func closeClientConns(clientConns []net.Conn, name string) {
	for i, _ := range clientConns {
		fmt.Printf("Closing client connection for %q...\n", name)
		conn := &clientConns[i]
		if err := (*conn).Close(); err != nil && !isClosedConnError(err) {
			fmt.Printf("error close client connection for %q: %v\n", name, err)
		}
	}
}

// TODO: keep track of errors
func handleClient(clientConn net.Conn, name string) {
	defer clientConn.Close()
	// Create a buffer to read data into
	buffer := make([]byte, 2048)

	for {
		// Read data from the client.
		n, err := clientConn.Read(buffer)
		if err != nil {
			fmt.Printf("read error on %q: %v\n", name, err)
			return
		}

		// Process and use the data (here, we'll just print it)
		fmt.Printf("read %q: %q\n", name, buffer[:n])
	}
}

func address(ip string, port uint) string {
	return fmt.Sprintf("%s:%d", ip, port)
}
