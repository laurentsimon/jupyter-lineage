package proxy

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/laurentsimon/jupyter-lineage/pkg/logger"
)

func connect(ctx context.Context, binding AddressBinding, logger logger.Logger,
	startErr chan error) {
	// srcToDstData, dstToSrcData chan []byte, startErr, srcToDstErr, dstToSrcErr chan error) {
	// TODO: like listener. hadleClient just nees to be changed with a read()
	var wg sync.WaitGroup
	var done bool
	var err error
	var conn net.Conn
	quit := make(chan struct{})

	// Start reading.
	conn, err = read(&wg, logger, binding, quit)
	startErr <- err
	if err != nil {
		return
	}
L:
	for !done {
		select {
		case <-ctx.Done():
			logger.Infof("connector exit for %q", binding.Name)
			done = true
			break L
		case <-quit:
			// Re-start reading.
			// todo: use non-blocking timer
			try := 0
			for {
				logger.Infof("connector re-start attempt %d for %q", try, binding.Name)
				conn, err = read(&wg, logger, binding, quit)
				// No error, done.
				if err == nil {
					break
				}
				// Error: retry.
				logger.Warnf("connector restart %q due to error: %v", binding.Name, err)
				try += 1
				if try >= 10 {
					done = true
					logger.Infof("connector exit for %q due to error: %v", binding.Name, err)
					break L
				}
				time.Sleep(1 * time.Second)
			}

			logger.Infof("connector restarted %q", binding.Name)
		default:
			// TODO: sleep
			continue
		}
		// TODO: read
		// case <-srcToDstQuit:
		// 	logger.Infof("exiting connector for %q (%q - %q) due to write channel close", binding.Name,
		// 		conn.LocalAddr().String(), conn.RemoteAddr().String())
		// 	return
		// case data, ok := <-srcToDstData:
		// 	if !ok {
		// 		logger.Infof("exiting connector for %q (%q - %q) due to write channel close (2)", binding.Name,
		// 			conn.LocalAddr().String(), conn.RemoteAddr().String())
		// 		//srcToDstErr <- fmt.Errorf("write channel %q closed")
		// 		return
		// 	}
		// 	logger.Debugf("TRY write for %q: %q", binding.Name, data)
		// 	n, err := conn.Write(data)
		// 	if n != len(data) {
		// 		logger.Errorf("write for %q: send %d bytes but wrote %d bytes", binding.Name, len(data), n)
		// 		srcToDstErr <- err
		// 		return
		// 	}
		// 	if err != nil {
		// 		logger.Errorf("write for %q: %q", binding.Name, data)
		// 		srcToDstErr <- err
		// 		return
		// 	}
		// 	logger.Debugf("write for %q: %v", binding.Name, data)
		// 	srcToDstErr <- nil
		// }
	}
	cclose(conn, binding.Name, logger)
	wg.Wait()
}

func read(wg *sync.WaitGroup, logger logger.Logger, binding AddressBinding, quit chan struct{}) (net.Conn, error) {
	conn, err := net.Dial("tcp", binding.Dst)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	if err := setConnSettings(logger, conn); err != nil {
		conn.Close()
		return nil, fmt.Errorf("connection settings: %w", err)
	}

	// Auto close on function exit.
	// TODO: need mutex
	//(*connector) = conn

	wg.Add(1)
	go func() {
		defer wg.Done()
		// NOTE: no need to synchronize access to counter because there's always at most
		// one client running.
		go handleRead(logger, quit, conn, binding.Name)
		/*srcToDstData, dstToSrcData, srcToDstQuit, dstToSrcQuit, srcToDstErr, dstToSrcErr*/
	}()
	return conn, nil
}

func handleRead(logger logger.Logger, quit chan struct{}, conn net.Conn, name string) {
	defer conn.Close()
	buffer := make([]byte, 2048)
	for {
		// Read data from the client
		n, err := conn.Read(buffer)
		if err != nil {
			logger.Errorf("connector read error on %q: %v", name, err)
			break
		}
		logger.Debugf("connector recv %q: %q", name, buffer[:n])
	}
	quit <- struct{}{}
}
