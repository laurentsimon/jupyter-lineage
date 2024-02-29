package proxy

import (
	"context"
	"sync"

	"github.com/laurentsimon/jupyter-lineage/pkg/logger"
	"github.com/laurentsimon/jupyter-lineage/pkg/repository"
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
	context    context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	logger     logger.Logger
	repoClient repository.Client
}

func New(bindings []AddressBinding, logger logger.Logger, repoClient repository.Client) (*Proxy, error) {
	ctx, cancel := context.WithCancel(context.Background())
	return &Proxy{
		logger:     logger,
		repoClient: repoClient,
		bindings:   bindings, // TODO: Make a copy.
		context:    ctx,
		cancel:     cancel,
	}, nil
}

// See https://okanexe.medium.com/the-complete-guide-to-tcp-ip-connections-in-golang-1216dae27b5a
// https://coderwall.com/p/wohavg/creating-a-simple-tcp-server-in-go
func (p *Proxy) Start() error {
	var e error
	// Start all the listeners and connectors.
	for i, _ := range p.bindings {
		binding := &p.bindings[i]
		startErr := make(chan error, 1)
		// srcToDstData := make(chan []byte)
		// srcToDstDataErr := make(chan error, 1)
		// dstToSrcData := make(chan []byte)
		// dstToSrcDataErr := make(chan error, 1)
		// srcToDstQuit := make(chan struct{})
		// dstToSrcQuit := make(chan struct{})
		// Connector.
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			connect(p.context, *binding, p.logger, startErr)
		}()
		e = <-startErr
		// If there was an error starting, finish immediatly.
		if e != nil {
			p.logger.Errorf("connect %q binding: %v", binding.Name, e)
			p.Finish()
			break
		}
		p.logger.Infof("connect %q successful", binding.Name)

		// Listener.
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			// listen(p.context, *binding, &p.listeners[i], p.logger, p.repoClient,
			// 	srcToDstData, dstToSrcData, srcToDstQuit, dstToSrcQuit, startErr, srcToDstDataErr, dstToSrcDataErr)
			listen(p.context, *binding, p.logger, p.repoClient, startErr)
		}()
		e = <-startErr
		// If there was an error starting, finish immediatly.
		if e != nil {
			p.logger.Errorf("listen %q binding: %v", binding.Name, e)
			p.Finish()
			break
		}
		p.logger.Infof("listen %q successful", binding.Name)
		p.logger.Infof("binding %q successful", binding.Name)
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
	// for i, _ := range p.bindings {
	// 	binding := &p.bindings[i]
	// 	listener := &p.listeners[i]
	// 	connector := &p.connectors[i]
	// 	cclose(*listener, binding.Name, p.logger)
	// 	// TODO: use mutex here: tODO
	// 	cclose(*connector, binding.Name, p.logger)
	// 	// TODO: close forwarders.
	// }

	// Wait for routines to exit.
	p.wg.Wait()
	return nil
}
