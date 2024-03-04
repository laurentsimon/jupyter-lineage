package session

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/laurentsimon/jupyter-lineage/pkg/errs"
	"github.com/laurentsimon/jupyter-lineage/pkg/logger"
	"github.com/laurentsimon/jupyter-lineage/pkg/repository"
	logimpl "github.com/laurentsimon/jupyter-lineage/pkg/session/internal/logger"
	"github.com/laurentsimon/jupyter-lineage/pkg/session/internal/proxy"
	slsaimpl "github.com/laurentsimon/jupyter-lineage/pkg/session/internal/slsa"
	"github.com/laurentsimon/jupyter-lineage/pkg/slsa"
)

type state uint

const (
	stateNew state = iota + 1
	stateStarted
	stateFinished
)

// See https://jupyter-client.readthedocs.io/en/stable/messaging.html
type Ports struct {
	Shell     uint
	Stdin     uint
	IOPub     uint
	Control   uint
	Heartbeat uint
}

type NetworkMetadata struct {
	IP    string
	Ports Ports
}

type Session struct {
	srcMetadata NetworkMetadata
	dstMetadata NetworkMetadata
	state       state
	repoClient  repository.Client
	proxies     []*proxy.Proxy
	logger      logger.Logger
	counter     atomic.Uint64
	startTime   time.Time
	provenance  []byte
}

type Option func(*Session) error

func New(srcMeta, dstMeta NetworkMetadata, repoClient repository.Client, options ...Option) (*Session, error) {
	// If https://go.googlesource.com/proposal/+/master/design/draft-iofs.md is ever implemented and merged,
	// we'll update the API to take an fs interface.
	addressBinding := []proxy.AddressBinding{
		{
			Name: "shell",
			Src:  address(srcMeta.IP, srcMeta.Ports.Shell),
			Dst:  address(dstMeta.IP, dstMeta.Ports.Shell),
		},
		{
			Name: "stdin",
			Src:  address(srcMeta.IP, srcMeta.Ports.Stdin),
			Dst:  address(dstMeta.IP, dstMeta.Ports.Stdin),
		},
		{
			Name: "iopub",
			Src:  address(srcMeta.IP, srcMeta.Ports.IOPub),
			Dst:  address(dstMeta.IP, dstMeta.Ports.IOPub),
		},
		{
			Name: "control",
			Src:  address(srcMeta.IP, srcMeta.Ports.Control),
			Dst:  address(dstMeta.IP, dstMeta.Ports.Control),
		},
		{
			Name: "heartbeat",
			Src:  address(srcMeta.IP, srcMeta.Ports.Heartbeat),
			Dst:  address(dstMeta.IP, dstMeta.Ports.Heartbeat),
		},
	}
	// TODO: Update this to be in our own repository with better ACLs / permissions.
	session := Session{
		srcMetadata: srcMeta,
		dstMetadata: dstMeta,
		state:       stateNew,
		repoClient:  repoClient,
	}

	// Set optional parameters.
	for _, option := range options {
		err := option(&session)
		if err != nil {
			return nil, err
		}
	}
	// Set the default logger
	if err := session.setDefaultLogger(); err != nil {
		return nil, err
	}

	// Set the proxy last, since we need to have the logger setup.
	for i, _ := range addressBinding {
		b := &addressBinding[i]
		proxy, err := proxy.New(*b, session.logger, session.repoClient, &session.counter)
		if err != nil {
			return nil, err
		}
		session.proxies = append(session.proxies, proxy)
	}

	return &session, nil
}

func address(ip string, port uint) string {
	return fmt.Sprintf("%s:%d", ip, port)
}

func (s *Session) Start() error {
	if s.state != stateNew {
		return fmt.Errorf("%w: state %q", errs.ErrorInvalid, s.state)
	}

	if err := s.repoClient.Init(); err != nil {
		return err
	}

	// Start proxies last.
	for i, _ := range s.proxies {
		p := s.proxies[i]
		if err := p.Start(); err != nil {
			return err
		}
	}

	// Update the session state.
	s.state = stateStarted
	s.startTime = time.Now()
	return nil
}

func (s *Session) Stop() error {
	// TODO: don't return early on error, innstead try to clean up as much as we can.
	if s.state == stateFinished {
		return fmt.Errorf("%w: state %q", errs.ErrorInvalid, s.state)
	}
	for i, _ := range s.proxies {
		p := s.proxies[i]
		if err := p.Stop(); err != nil {
			s.logger.Errorf("proxy stop: %v", err)
		}
	}

	if err := s.repoClient.Close(); err != nil {
		s.logger.Errorf("repo close: %v", err)
	}
	s.state = stateFinished
	return nil
}

// todo: support adding dependencies.
func (s *Session) Provenance(builder slsa.Builder, subjects []slsa.Subject, repoURI string) ([]byte, error) {
	if s.state != stateFinished {
		return nil, fmt.Errorf("%w: state %q", errs.ErrorInvalid, s.state)
	}
	if s.provenance != nil {
		return s.provenance, nil
	}
	digestSet, err := s.repoClient.Digest()
	if err != nil {
		return nil, err
	}
	repo := slsaimpl.Dependency{
		DigestSet: digestSet,
		URI:       repoURI,
	}
	prov, err := slsaimpl.New(builder, subjects, repo,
		slsaimpl.WithStartTime(s.startTime),
		slsaimpl.WithFinishTime(time.Now()),
	)
	if err != nil {
		return nil, err
	}
	s.provenance, err = prov.ToBytes()
	if err != nil {
		return nil, err
	}
	return append([]byte{}, s.provenance...), nil
}

func (s *Session) setDefaultLogger() error {
	if s.logger != nil {
		return nil
	}
	s.logger = logimpl.Logger{}
	return nil
}

func WithLogger(l logger.Logger) Option {
	return func(s *Session) error {
		return s.setLogger(l)
	}
}

func (s *Session) setLogger(l logger.Logger) error {
	s.logger = l
	return nil
}

// TODO: HMAC keys
