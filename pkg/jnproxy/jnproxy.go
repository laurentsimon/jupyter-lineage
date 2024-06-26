package jnproxy

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/laurentsimon/jupyter-lineage/pkg/errs"
	httphandler "github.com/laurentsimon/jupyter-lineage/pkg/jnproxy/handler/http"
	logimpl "github.com/laurentsimon/jupyter-lineage/pkg/jnproxy/internal/logger"
	"github.com/laurentsimon/jupyter-lineage/pkg/jnproxy/internal/proxy"
	httpproxy "github.com/laurentsimon/jupyter-lineage/pkg/jnproxy/internal/proxy/http"
	"github.com/laurentsimon/jupyter-lineage/pkg/jnproxy/internal/proxy/jserver"
	slsaimpl "github.com/laurentsimon/jupyter-lineage/pkg/jnproxy/internal/slsa"
	"github.com/laurentsimon/jupyter-lineage/pkg/logger"
	"github.com/laurentsimon/jupyter-lineage/pkg/repository"
	"github.com/laurentsimon/jupyter-lineage/pkg/slsa"
)

type state uint

const (
	stateNew state = iota + 1
	stateStarted
	stateFinished
)

type JNProxy struct {
	state        state
	repoClient   repository.Client
	proxies      []proxy.Proxy
	logger       logger.Logger
	counter      atomic.Uint64
	startTime    time.Time
	provenance   []byte
	ca           *CA
	httpHandlers []httphandler.Handler
}

type Option func(*JNProxy) error

/*
import os
os.environ['HTTP_PROXY'] = 'localhost:9999'
os.environ['HTTPS_PROXY'] = 'localhost:9999'

from urllib.request import urlopen

import requests
import urllib
response = requests.get("http://localhost:8082/v3/nodes", data=query)
print response.json()

import urllib3

# Creating a PoolManager instance for sending requests.
http = urllib3.PoolManager()

# Sending a GET request and getting back response as HTTPResponse object.
resp = http.request("GET", "http://www.google.com")

# Print the returned data.
print(resp.data)
*/
func New(jServerConfig JServerConfig, httpConfig HttpConfig, repoClient repository.Client, options ...Option) (*JNProxy, error) {
	// If https://go.googlesource.com/proposal/+/master/design/draft-iofs.md is ever implemented and merged,
	// we'll update the API to take an fs interface.
	srcConfig := jServerConfig.src()
	dstConfig := jServerConfig.dst()
	addressBinding := []jserver.AddressBinding{
		{
			Name: "shell",
			Src:  address(srcConfig.IP, srcConfig.Ports.Shell),
			Dst:  address(dstConfig.IP, dstConfig.Ports.Shell),
		},
		{
			Name: "stdin",
			Src:  address(srcConfig.IP, srcConfig.Ports.Stdin),
			Dst:  address(dstConfig.IP, dstConfig.Ports.Stdin),
		},
		{
			Name: "iopub",
			Src:  address(srcConfig.IP, srcConfig.Ports.IOPub),
			Dst:  address(dstConfig.IP, dstConfig.Ports.IOPub),
		},
		{
			Name: "control",
			Src:  address(srcConfig.IP, srcConfig.Ports.Control),
			Dst:  address(dstConfig.IP, dstConfig.Ports.Control),
		},
		{
			Name: "heartbeat",
			Src:  address(srcConfig.IP, srcConfig.Ports.Heartbeat),
			Dst:  address(dstConfig.IP, dstConfig.Ports.Heartbeat),
		},
	}

	// TODO: Update this to be in our own repository with better ACLs / permissions.
	jnproxy := JNProxy{
		state:      stateNew,
		repoClient: repoClient,
		logger:     logimpl.Logger{},
	}

	// Set optional parameters.
	for _, option := range options {
		err := option(&jnproxy)
		if err != nil {
			return nil, err
		}
	}

	// Set the proxy last, since we need to have the logger setup.
	for i := range addressBinding {
		b := &addressBinding[i]
		proxy, err := jserver.New(*b, jnproxy.repoClient, &jnproxy.counter, jserver.WithLogger(jnproxy.logger))
		if err != nil {
			return nil, err
		}
		jnproxy.proxies = append(jnproxy.proxies, proxy)
	}

	// Create the http proxy.
	opts := []httpproxy.Option{
		httpproxy.WithLogger(jnproxy.logger),
		httpproxy.WithHandlers(jnproxy.httpHandlers),
	}
	if jnproxy.ca != nil {
		opts = append(opts, httpproxy.WithCA(httpproxy.CA{
			Certificate: jnproxy.ca.Certificate,
			Key:         jnproxy.ca.Key,
		}))
	}
	for i := range httpConfig.addr {
		addr := &httpConfig.addr[i]
		httpProxy, err := httpproxy.New(*addr, opts...)
		if err != nil {
			return nil, err
		}
		jnproxy.proxies = append(jnproxy.proxies, httpProxy)
	}

	return &jnproxy, nil
}

func address(ip string, port uint) string {
	return fmt.Sprintf("%s:%d", ip, port)
}

func (s *JNProxy) Start() error {
	if s.state != stateNew {
		return fmt.Errorf("%w: state %q", errs.ErrorInvalid, s.state)
	}

	if err := s.repoClient.Init(); err != nil {
		return err
	}

	// Start proxies last.
	for i := range s.proxies {
		p := s.proxies[i]
		if err := p.Start(); err != nil {
			return err
		}
	}

	// Update the JNProxy state.
	s.state = stateStarted
	s.startTime = time.Now()
	return nil
}

func (s *JNProxy) Stop() error {
	// TODO: don't return early on error, innstead try to clean up as much as we can.
	if s.state == stateFinished {
		return fmt.Errorf("%w: state %q", errs.ErrorInvalid, s.state)
	}
	for i := range s.proxies {
		p := s.proxies[i]
		if err := p.Stop(); err != nil {
			s.logger.Errorf("proxy stop: %v", err)
		}
	}

	// if err := s.repoClient.Close(); err != nil {
	// 	s.logger.Errorf("repo close: %v", err)
	// }
	s.state = stateFinished
	return nil
}

func (s *JNProxy) Provenance(builder slsa.Builder, subjects []slsa.Subject, repoURI string) ([]byte, error) {
	if s.state != stateFinished {
		return nil, fmt.Errorf("%w: state %q", errs.ErrorInvalid, s.state)
	}
	if s.provenance != nil {
		return s.provenance, nil
	}
	// digestSet, err := s.repoClient.Digest()
	// if err != nil {
	// 	return nil, err
	// }
	digestSet := slsa.DigestSet{"bla": "val"}
	repo := slsa.ResourceDescriptor{
		DigestSet: digestSet,
		URI:       repoURI,
	}
	// Get runtime dependencies.
	var deps []slsa.ResourceDescriptor
	for i := range s.proxies {
		p := s.proxies[i]
		if p.Type() == proxy.TypeUserSource {
			continue
		}
		d, err := p.Dependencies()
		if err != nil {
			return nil, err
		}
		deps = append(deps, d...)
	}

	prov, err := slsaimpl.New(builder, subjects, repo,
		slsaimpl.WithStartTime(s.startTime),
		slsaimpl.WithFinishTime(time.Now()),
		slsaimpl.AddDependencies(deps),
	)
	if err != nil {
		return nil, err
	}
	// TODO: Add dependencies collected from the http proxies.
	s.provenance, err = prov.ToBytes()
	if err != nil {
		return nil, err
	}
	return append([]byte{}, s.provenance...), nil
}

func WithLogger(l logger.Logger) Option {
	return func(s *JNProxy) error {
		return s.setLogger(l)
	}
}

func (s *JNProxy) setLogger(l logger.Logger) error {
	s.logger = l
	return nil
}

// TODO: HMAC keys
