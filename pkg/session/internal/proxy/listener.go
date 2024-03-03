package proxy

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/laurentsimon/jupyter-lineage/pkg/logger"
	"github.com/laurentsimon/jupyter-lineage/pkg/session/internal/conduit"
)

type shared struct {
	mu    sync.Mutex
	conns []net.Conn
}

// TODO: channel for error
func listen(ctx context.Context, binding AddressBinding, logger logger.Logger,
	//srcToDstData, dstToSrcData chan []byte, srcToDstQuit, dstToSrcQuit chan struct{},
	conduit *conduit.Conduit, startErr /*, srcToDstErr, dstToSrcErr*/ chan error) {
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
	var share shared
	cache := bytes.NewBuffer(nil)
	timer := time.NewTimer(0 * time.Second)
	<-timer.C

	// Start listening.
	go accept(logger, listenConn, binding, conduit, &share)

L:
	for !done {
		select {
		case <-ctx.Done():
			logger.Infof("exiting listener for %q", binding.Name)
			done = false
			break L
		case <-timer.C:
			logger.Debugf("listener %q cache data to forward: %q", binding.Name, cache.Bytes())
			share.mu.Lock()
			logger.Debugf("listener len conns: %d", len(share.conns))
			if len(share.conns) == 0 {
				logger.Warnf("listener no client connected")
				logger.Infof("listener %q caching data %q", binding.Name, cache.Bytes())
				timer = time.NewTimer(2 * time.Second)
				share.mu.Unlock()
				continue
			}
			// timer = test(logger, cache, binding.Name, &share)
			// if timer != nil {
			// 	share.mu.Unlock()
			// 	continue
			// }
			index := len(share.conns) - 1
			for index >= 0 {
				conn := &(share.conns)[index]
				logger.Debugf("conn %p", *conn)
				if err = connWrite(*conn, cache.Bytes()); err != nil {
					logger.Debugf("listener write %q on conn %d: %v", binding.Name, index, err)
					index -= 1
					continue
				}
				logger.Debugf("listener %q forwarded data %q on conn %d", binding.Name, cache.Bytes(), index)
				// The buffer may be the cache, and we want to be sure we only send data once.
				cache.Reset()
				break
			}
			share.mu.Unlock()
			if err != nil {
				timer = time.NewTimer(2 * time.Second)
			}
			// if err != nil {
			// 	logger.Fatalf("listener %q forwarded data %q failed: %v", binding.Name, cache.Bytes(), err)
			// }

		case data := <-conduit.Src():
			logger.Debugf("listener %q recv to forward: %q", binding.Name, data)
			// Use any of the connections to send. Traverse backward because newr connections are
			// at the back.
			_, err := cache.Write(data)
			if err != nil {
				logger.Fatalf("listener %q write: %v", binding.Name, cache.Bytes(), err)
			}
			share.mu.Lock()
			//timer = forward(logger, cache, binding.Name, &share)
			logger.Debugf("listener len conns: %d", len(share.conns))
			if len(share.conns) == 0 {
				logger.Warnf("listener no client connected")
				logger.Infof("listener %q caching data %q", binding.Name, cache.Bytes())
				timer = time.NewTimer(2 * time.Second)
				share.mu.Unlock()
				continue
			}

			index := len(share.conns) - 1
			for index >= 0 {
				conn := &(share.conns)[index]
				logger.Debugf("conn %p", *conn)
				if err = connWrite(*conn, cache.Bytes()); err != nil {
					logger.Debugf("listener write %q on conn %d: %v", binding.Name, index, err)
					index -= 1
					continue
				}
				logger.Debugf("listener %q forwarded data %q on conn %d", binding.Name, cache.Bytes(), index)
				// The buffer may be the cache, and we want to be sure we only send data once.
				cache.Reset()
				break
			}
			if err != nil {
				timer = time.NewTimer(2 * time.Second)
			}
			share.mu.Unlock()

			// if err != nil {
			// 	logger.Fatalf("listener %q forwarded data %q failed: %v", binding.Name, cache.Bytes(), err)
			// }

		default:
			continue // TODO sleep
		}
	}
}

func forward(logger logger.Logger, data *bytes.Buffer, name string, share *shared) *time.Timer {
	logger.Debugf("listener len conns: %d", len(share.conns))
	if len(share.conns) == 0 {
		logger.Warnf("listener no client connected")
		logger.Infof("listener %q caching data %q", name, data)
		return time.NewTimer(2 * time.Second)
	}
	// var trim []int
	// index := len(share.conns) - 1
	// var err error
	// for index >= 0 {
	// 	conn := &(share.conns)[index]
	// 	logger.Debugf("conn forward %p", *conn)
	// 	if err = connWrite(*conn, data.Bytes()); err != nil {
	// 		logger.Debugf("listener write %q on conn %d: %v", name, index, err)
	// 		if isClosedConnError(err) {
	// 			trim = append(trim, index)
	// 		}
	// 		index -= 1
	// 		continue
	// 	}
	// 	logger.Debugf("listener %q forwarded data %q on conn %d", name, data.Bytes(), index)
	// 	// The buffer may be the cache, and we want to be sure we only send data once.
	// 	data.Reset()
	// 	break
	// }

	var err error
	if err = connWrite(share.conns[0], data.Bytes()); err != nil {
		logger.Debugf("listener write %q on conn %d: %v", name, 0, err)
	}

	// Remove closed connections.
	// for i, _ := range trim {
	// 	share.conns = removeIndex(share.conns, i)
	// }
	if err != nil {
		return time.NewTimer(2 * time.Second)
	}
	return nil
}

func removeIndex(s []net.Conn, index int) []net.Conn {
	return append(s[:index], s[index+1:]...)
}

func accept(logger logger.Logger, listener net.Listener, binding AddressBinding, conduit *conduit.Conduit, share *shared) {
	// Accept the connection.
	logger.Infof("listening for %q", binding.Name)
	var wg sync.WaitGroup
	for {
		logger.Infof("accept for %q", binding.Name)
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
		// TODO: add to manager
		// map[string] {toDst, toSrc channels + 2 mutexes to set the }
		// Keep track of the connections.
		share.mu.Lock()
		share.conns = append(share.conns, conn)
		share.mu.Unlock()
		// Handle the connection.
		wg.Add(1)
		go func() {
			defer wg.Done()
			// NOTE: no need to synchronize access to counter because there's always at most
			// one client running.
			handleClient(logger, conn, conduit, binding.Name)
			/*srcToDstData, dstToSrcData, srcToDstQuit, dstToSrcQuit, srcToDstErr, dstToSrcErr*/
		}()
	}
	// We need to close all connections that are waiting on read.
	share.mu.Lock()
	closeConns(logger, share.conns, binding.Name)
	share.mu.Unlock()
	wg.Wait()
}

// TODO: keep track of errors
func handleClient(logger logger.Logger, clientConn net.Conn, conduit *conduit.Conduit, name string) {
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
		// TODO: we need a full packet here
		logger.Debugf("listener %q recv: %q", name, buffer[:n])
		// Forward.
		conduit.Dst() <- buffer[:n]

		// srcToDstData <- buffer[:n]
		// err = <-srcToDstErr
		// if err != nil {
		// 	// TODO: handle reconnection. Do that in caller by rec-creating
		// 	// a listern / connector combo.
		// 	// We probably just need to set a flag here to indicate re-try.
		// 	logger.Errorf("connector write on %q: %v", name, err)
		// 	break
		// }

	}
	logger.Debugf("handleConn exit %q", name)
	//srcToDstQuit <- struct{}{}
}
