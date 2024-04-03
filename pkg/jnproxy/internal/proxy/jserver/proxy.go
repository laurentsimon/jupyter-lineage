package jserver

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/laurentsimon/jupyter-lineage/pkg/logger"
	"github.com/laurentsimon/jupyter-lineage/pkg/repository"
)

type AddressBinding struct {
	Name string
	Src  string
	Dst  string
}

// https://eli.thegreenplace.net/2020/graceful-shutdown-of-a-tcp-server-in-go/
// https://okanexe.medium.com/the-complete-guide-to-tcp-ip-connections-in-golang-1216dae27b5a
// https://shantanubansal.medium.com/how-to-terminate-goroutines-in-go-effective-methods-and-examples-f796dcede512
type Proxy struct {
	binding    AddressBinding
	listener   net.Listener
	conns      []net.Conn
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	logger     logger.Logger
	repoClient repository.Client
	mu         sync.Mutex
	counter    *atomic.Uint64
}

func New(binding AddressBinding, logger logger.Logger, repoClient repository.Client, counter *atomic.Uint64) (*Proxy, error) {
	ctx, cancel := context.WithCancel(context.Background())
	proxy := &Proxy{
		binding:    binding,
		ctx:        ctx,
		cancel:     cancel,
		logger:     logger,
		repoClient: repoClient,
		counter:    counter,
	}
	return proxy, nil
}

func (p *Proxy) Start() error {
	if p.listener != nil {
		return fmt.Errorf("[jserver]: proxy already running")
	}
	// TODO: use ctx
	listener, err := net.Listen("tcp", p.binding.Src)
	if err != nil {
		return fmt.Errorf("[jserver]: listen (%q): %w", p.binding.Name, err)
	}
	p.listener = listener
	p.wg.Add(1)
	go p.serve()
	return nil
}

func (p *Proxy) serve() {
	defer p.wg.Done()

	for {
		src, err := p.listener.Accept()
		if err != nil {
			if p.isCancelled() {
				p.logger.Infof("[jserver]: serve (%q) exiting", p.lstID(p.listener))
				return
			}
			continue
		}
		p.logger.Errorf("[jserver]: serve (%q) accept from %v", p.lstID(p.listener), src.RemoteAddr().String())
		p.setConnSettings(src)

		dst, err := net.Dial("tcp", p.binding.Dst)
		if err != nil {
			p.logger.Errorf("[jserver]: serve (%q) dial: %v", p.connID(src), err)
			p.closeConn(src)
			continue
		}
		p.logger.Errorf("[jserver]: serve (%q) connect to %v", p.lstID(p.listener), dst.RemoteAddr().String())
		p.setConnSettings(dst)

		// WARNING: There is a race condition here. If Stop() is called,
		// it may close all recorded connections except the one we're adding here
		// because it may be recorded *after* other connections are closed.
		// This would eventually block Stop() on the group Wait().
		if !p.recordConns(src, dst) {
			p.closeConn(src)
			p.closeConn(dst)
			p.logger.Infof("[jserver]: serve (%q) (%q) exiting", p.connID(src), p.connID(dst))
			return
		}
		p.wg.Add(1)
		go func() {
			p.forward(src, dst, true)
			p.wg.Done()
		}()
		p.wg.Add(1)
		go func() {
			p.forward(dst, src, false)
			p.wg.Done()
		}()
	}
}

func (p *Proxy) recordConns(src, dst net.Conn) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	// WARNING: It's important we check the cancellation while holding the mutex.
	if p.isCancelled() {
		p.logger.Infof("[jserver]: serve (%q) recordConns cancelled", p.connID(src))
		p.logger.Infof("[jserver]: serve (%q) recordConns cancelled", p.connID(dst))
		return false
	}
	p.conns = append(p.conns, src, dst)
	return true
}

func (p *Proxy) isCancelled() bool {
	select {
	case <-p.ctx.Done():
		return true
	default:
	}
	return false
}

func (p *Proxy) Stop() error {
	p.mu.Lock()
	p.cancel()
	p.closeLst(p.listener)
	for _, conn := range p.conns {
		p.closeConn(conn)
	}
	p.mu.Unlock()
	p.wg.Wait()
	return nil
}

func (p *Proxy) forward(src, dst net.Conn, record bool) {
	// If this function returns, we close both ends of the connection.
	defer p.closeConn(src)
	defer p.closeConn(dst)
	buf := make([]byte, 2048)
	for {
		n, err := src.Read(buf)
		if err != nil && err != io.EOF {
			p.logger.Errorf("[jserver]: forward (%q -> %q) read: %v", p.connID(src), p.connID(dst), err)
			return
		}
		if n == 0 {
			p.logger.Warnf("[jserver]: forward (%q -> %q) return", p.connID(src), p.connID(dst))
			return
		}
		p.logger.Debugf("[jserver]: forward (%q -> %q) received: %q", p.connID(src), p.connID(dst), string(buf[:n]))

		p.counter.Add(1)

		if record {
			// Record the data. We do that _before_ data is actually sent because a malicious
			// kernel could close the connection and act as if the data was not received.
			fn := fmt.Sprintf("%s/%016x_%s", p.binding.Name, p.counter.Load(), time.Now().UTC().Format(time.RFC3339))
			if err := p.repoClient.CreateFile(fn, buf[:n]); err != nil {
				p.logger.Errorf("[jserver]: forward create file %q: %v", fn, err)
				return
			}
		}

		// Copy data to dst.
		_, err = dst.Write(buf[:n])
		if err != nil {
			p.logger.Errorf("[jserver]: forward (%q -> %q) write: %v", p.connID(src), p.connID(dst), err)
			return
		}
		// TODO: Keep a record of the write result.
	}
}

func (p *Proxy) connID(conn net.Conn) string {
	return fmt.Sprintf("%s/%s", p.binding.Name, conn.RemoteAddr().String())
}

func (p *Proxy) lstID(lst net.Listener) string {
	return fmt.Sprintf("%s/%s", p.binding.Name, lst.Addr().String())
}

func (p *Proxy) closeLst(lst net.Listener) {
	p.logger.Debugf("[jserver]: (%q) close", p.lstID(lst))
	lst.Close()
}

func (p *Proxy) closeConn(conn net.Conn) {
	p.logger.Debugf("jserver(%q) close", p.connID(conn))
	conn.Close()
}

type setNoDelayer interface {
	SetNoDelay(bool) error
}

func (p *Proxy) setConnSettings(conn net.Conn) {
	if c, ok := conn.(setNoDelayer); ok {
		// https://pkg.go.dev/net#TCPConn.SetNoDelay
		p.logger.Debugf("enable nagle (%q)", p.connID(conn))
		c.SetNoDelay(true)
	}
	if err := conn.(*net.TCPConn).SetKeepAlive(true); err == nil {
		p.logger.Debugf("keep alive (%q)", p.connID(conn))
	}
}
