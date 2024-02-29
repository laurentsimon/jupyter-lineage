package proxy

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/laurentsimon/jupyter-lineage/pkg/logger"
	"github.com/laurentsimon/jupyter-lineage/pkg/repository"
)

// TODO: channel for error
func listen(ctx context.Context, binding AddressBinding, logger logger.Logger,
	repoClient repository.Client, //srcToDstData, dstToSrcData chan []byte, srcToDstQuit, dstToSrcQuit chan struct{},
	startErr /*, srcToDstErr, dstToSrcErr*/ chan error) {
	listenConn, err := net.Listen("tcp", binding.Src)
	if err != nil {
		startErr <- fmt.Errorf("listen: %w", err)
		return
	}
	// Communicate to caller thhat we started successfully.
	startErr <- nil
	// Auto close on function exit.
	defer listenConn.Close()
	//(*listener) = listenConn

	// TODO: no delay https://github.com/jpillora/go-tcp-proxy/blob/master/proxy.go

	var done bool

	// Start listening.
	go accept(logger, listenConn, binding, repoClient)

L:
	for !done {
		select {
		case <-ctx.Done():
			logger.Infof("exiting listener for %q", binding.Name)
			done = false
			break L
		default:
			continue // TODO sleep
		}
	}
}

func accept(logger logger.Logger, listener net.Listener, binding AddressBinding, repoClient repository.Client) {
	// Accept the connection.
	logger.Infof("listening for %q", binding.Name)
	var wg sync.WaitGroup
	var conns []net.Conn
	var shared sharedVariables
	for {
		conn, err := listener.Accept()
		if err != nil {
			// TODO: need to check the kind of error
			// and log info when closing properly if caller set to close.
			// TODO: use log iterface from caller
			//log.Fatal(err)
			logger.Errorf("accept %q: %v", binding.Name, err)
			break
		}
		logger.Infof("connection from %q", conn.RemoteAddr().String())
		if err := setConnSettings(logger, conn); err != nil {
			logger.Errorf("connection settings %q: %v", binding.Name, err)
			continue
		}
		// Keep track of the connections.
		conns = append(conns, conn)
		// Handle the connection.
		wg.Add(1)
		go func() {
			defer wg.Done()
			// NOTE: no need to synchronize access to counter because there's always at most
			// one client running.
			handleClient(logger, repoClient, conn, binding.Name, &shared)
			/*srcToDstData, dstToSrcData, srcToDstQuit, dstToSrcQuit, srcToDstErr, dstToSrcErr*/
		}()
	}
	// We need to close all connections that are waiting on read.
	closeConns(logger, conns, binding.Name)
	wg.Wait()
}

// TODO: keep track of errors
func handleClient(logger logger.Logger, repoClient repository.Client, clientConn net.Conn, name string,
	shared *sharedVariables) {
	/*, srcToDstData, dstToSrcData chan []byte, srcToDstQuit, dstToSrcQuit chan struct{},
	srcToDstErr, dstToSrcErr chan error*/
	defer clientConn.Close()
	//defer close(srcToDstData)
	// Create a buffer to read data into
	buffer := make([]byte, 2048)

	for {
		// Read data from the client.
		logger.Debugf("listener reading on %q", name)
		n, err := clientConn.Read(buffer)
		if err != nil {
			// NOTE: closed connection will get an error and return.
			logger.Errorf("listener read on %q: %v", name, err)
			break
		}

		// Forward.
		// srcToDstData <- buffer[:n]
		// err = <-srcToDstErr
		// if err != nil {
		// 	// TODO: handle reconnection. Do that in caller by rec-creating
		// 	// a listern / connector combo.
		// 	// We probably just need to set a flag here to indicate re-try.
		// 	logger.Errorf("connector write on %q: %v", name, err)
		// 	break
		// }

		// Process and use the data (here, we'll just print it)
		logger.Debugf("listener recv %q: %q", name, buffer[:n])
		fn := fmt.Sprintf("%s/%016x_%s", name, shared.counter(), time.Now().UTC().Format(time.RFC3339))

		// c, err := slsa.Format(buffer[:n])
		// if err != nil {
		// 	logger.Fatalf("slsa format %q: []%v: %v", fn, buffer[:n], err)
		// }
		if err := repoClient.CreateFile(fn, buffer[:n]); err != nil {
			// TODO: handle gracefully? Need to return and set an err
			// for the caller to check.
			logger.Fatalf("create file %q: %v", fn, err)
		}
		shared.counterInc()
	}
	logger.Debugf("handleConn exit %q", name)
	//srcToDstQuit <- struct{}{}
}
