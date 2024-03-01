package proxy

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/laurentsimon/jupyter-lineage/pkg/logger"
	"github.com/laurentsimon/jupyter-lineage/pkg/repository"
	"github.com/laurentsimon/jupyter-lineage/pkg/session/internal/conduit"
)

func connect(ctx context.Context, binding AddressBinding, logger logger.Logger,
	repoClient repository.Client, conduit *conduit.Conduit, startErr chan error) {
	// srcToDstData, dstToSrcData chan []byte, startErr, srcToDstErr, dstToSrcErr chan error) {
	// TODO: like listener. hadleClient just nees to be changed with a read()
	var wg sync.WaitGroup
	var done bool
	var err error
	var conn net.Conn
	var counter uint64
	quit := make(chan struct{})
	try := 1
	timer := time.NewTimer(0 * time.Second)
	<-timer.C

	conn, err = read(&wg, logger, binding, quit, conduit)
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
		case data := <-conduit.Dst():
			// TDO: may need to split packets here to follow ZQM protocol.
			logger.Debugf("connector %q recv to forward: %q", binding.Name, data)
			fn := fmt.Sprintf("%s/%016x_%s", binding.Name, counter, time.Now().UTC().Format(time.RFC3339))

			if conn == nil {
				// TODO: gracefully
				logger.Fatalf("connector write %q: no connector", binding.Name)
			}
			// Send data.
			if err := connWrite(conn, data); err != nil {
				// TODO: gracefully
				logger.Fatalf("connector write %q: %v", binding.Name, err)
			}
			logger.Debugf("connector %q forwarded data: %q", binding.Name, data)
			// c, err := slsa.Format(buffer[:n])
			// if err != nil {
			// 	logger.Fatalf("slsa format %q: []%v: %v", fn, buffer[:n], err)
			// }
			if err := repoClient.CreateFile(fn, data); err != nil {
				// TODO: handle gracefully? Need to return and set an err
				// for the caller to check.
				logger.Fatalf("create file %q: %v", fn, err)
			}
			counter += 1

		case <-timer.C:
			logger.Infof("connector re-start attempt %d for %q", try, binding.Name)
			// TODO: add to manager, 1 mutex per direction
			conn, err = read(&wg, logger, binding, quit, conduit)
			// No error, done.
			if err == nil {
				logger.Infof("connector restarted %q", binding.Name)
				try = 1
				continue
			}
			// Error: retry.
			logger.Warnf("connector restart %q due to error: %v", binding.Name, err)
			try += 1
			if try >= 10 {
				done = true
				logger.Infof("connector exit for %q due to error: %v", binding.Name, err)
				break L
			}
			timer = time.NewTimer(time.Duration(try) * time.Second)
			logger.Infof("connector re-start attempt %d for %q in %ds", try, binding.Name, try)
		case <-quit:
			// Re-start reading.
			timer = time.NewTimer(time.Duration(try) * time.Second)
			logger.Infof("connector re-start attempt %d for %q in %ds", try, binding.Name, try)
		default:
			// TODO: sleep
			continue
		}
	}
	cclose(conn, binding.Name, logger)
	wg.Wait()
}

func read(wg *sync.WaitGroup, logger logger.Logger, binding AddressBinding, quit chan struct{}, conduit *conduit.Conduit) (net.Conn, error) {
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
		go handleRead(logger, quit, conn, conduit, binding.Name)
		/*srcToDstData, dstToSrcData, srcToDstQuit, dstToSrcQuit, srcToDstErr, dstToSrcErr*/
	}()
	return conn, nil
}

func handleRead(logger logger.Logger, quit chan struct{}, conn net.Conn, conduit *conduit.Conduit, name string) {
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

		conduit.Src() <- buffer[:n]
	}
	quit <- struct{}{}
}
